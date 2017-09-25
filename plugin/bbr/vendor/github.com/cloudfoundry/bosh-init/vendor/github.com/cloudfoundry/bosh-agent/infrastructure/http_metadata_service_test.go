package infrastructure_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	fakeinf "github.com/cloudfoundry/bosh-agent/infrastructure/fakes"
	fakeplat "github.com/cloudfoundry/bosh-agent/platform/fakes"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	. "github.com/cloudfoundry/bosh-agent/infrastructure"
)

var _ = Describe("HTTPMetadataService", describeHTTPMetadataService)

func describeHTTPMetadataService() {
	var (
		metadataHeaders map[string]string
		dnsResolver     *fakeinf.FakeDNSResolver
		platform        *fakeplat.FakePlatform
		logger          boshlog.Logger
		metadataService MetadataService
	)

	BeforeEach(func() {
		metadataHeaders = make(map[string]string)
		metadataHeaders["key"] = "value"
		dnsResolver = &fakeinf.FakeDNSResolver{}
		platform = fakeplat.NewFakePlatform()
		logger = boshlog.NewLogger(boshlog.LevelNone)
		metadataService = NewHTTPMetadataService("fake-metadata-host", metadataHeaders, "/user-data", "/instanceid", "/ssh-keys", dnsResolver, platform, logger)
	})

	ItEnsuresMinimalNetworkSetup := func(subject func() (string, error)) {
		Context("when no networks are configured", func() {
			BeforeEach(func() {
				platform.GetConfiguredNetworkInterfacesInterfaces = []string{}
			})

			It("sets up DHCP network", func() {
				_, err := subject()
				Expect(err).ToNot(HaveOccurred())

				Expect(platform.SetupNetworkingCalled).To(BeTrue())
				Expect(platform.SetupNetworkingNetworks).To(Equal(boshsettings.Networks{
					"eth0": boshsettings.Network{
						Type: "dynamic",
					},
				}))
			})

			Context("when setting up DHCP fails", func() {
				BeforeEach(func() {
					platform.SetupNetworkingErr = errors.New("fake-network-error")
				})

				It("returns an error", func() {
					_, err := subject()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-network-error"))
				})
			})
		})
	}

	Describe("IsAvailable", func() {
		It("returns true", func() {
			Expect(metadataService.IsAvailable()).To(BeTrue())
		})
	})

	Describe("GetPublicKey", func() {
		var (
			ts          *httptest.Server
			sshKeysPath string
		)

		BeforeEach(func() {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer GinkgoRecover()

				Expect(r.Method).To(Equal("GET"))
				Expect(r.URL.Path).To(Equal("/ssh-keys"))
				Expect(r.Header.Get("key")).To(Equal("value"))

				w.Write([]byte("fake-public-key"))
			})
			ts = httptest.NewServer(handler)
		})

		AfterEach(func() {
			ts.Close()
		})

		Context("when the ssh keys path is present", func() {
			BeforeEach(func() {
				sshKeysPath = "/ssh-keys"
				metadataService = NewHTTPMetadataService(ts.URL, metadataHeaders, "/user-data", "/instanceid", sshKeysPath, dnsResolver, platform, logger)
			})

			It("returns fetched public key", func() {
				publicKey, err := metadataService.GetPublicKey()
				Expect(err).NotTo(HaveOccurred())
				Expect(publicKey).To(Equal("fake-public-key"))
			})

			ItEnsuresMinimalNetworkSetup(func() (string, error) {
				return metadataService.GetPublicKey()
			})
		})

		Context("when the ssh keys path is not present", func() {
			BeforeEach(func() {
				sshKeysPath = ""
				metadataService = NewHTTPMetadataService(ts.URL, metadataHeaders, "/user-data", "/instanceid", sshKeysPath, dnsResolver, platform, logger)
			})

			It("returns an empty ssh key", func() {
				publicKey, err := metadataService.GetPublicKey()
				Expect(err).NotTo(HaveOccurred())
				Expect(publicKey).To(BeEmpty())
			})
		})
	})

	Describe("GetInstanceID", func() {
		var (
			ts             *httptest.Server
			instanceIDPath string
		)

		BeforeEach(func() {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer GinkgoRecover()

				Expect(r.Method).To(Equal("GET"))
				Expect(r.URL.Path).To(Equal("/instanceid"))
				Expect(r.Header.Get("key")).To(Equal("value"))

				w.Write([]byte("fake-instance-id"))
			})
			ts = httptest.NewServer(handler)
		})

		AfterEach(func() {
			ts.Close()
		})

		Context("when the instance ID path is present", func() {
			BeforeEach(func() {
				instanceIDPath = "/instanceid"
				metadataService = NewHTTPMetadataService(ts.URL, metadataHeaders, "/user-data", instanceIDPath, "/ssh-keys", dnsResolver, platform, logger)
			})

			It("returns fetched instance id", func() {
				instanceID, err := metadataService.GetInstanceID()
				Expect(err).NotTo(HaveOccurred())
				Expect(instanceID).To(Equal("fake-instance-id"))
			})

			ItEnsuresMinimalNetworkSetup(func() (string, error) {
				return metadataService.GetInstanceID()
			})
		})

		Context("when the instance ID path is not present", func() {
			BeforeEach(func() {
				instanceIDPath = ""
				metadataService = NewHTTPMetadataService(ts.URL, metadataHeaders, "/user-data", instanceIDPath, "/ssh-keys", dnsResolver, platform, logger)
			})

			It("returns an empty instance ID", func() {
				instanceID, err := metadataService.GetInstanceID()
				Expect(err).NotTo(HaveOccurred())
				Expect(instanceID).To(BeEmpty())
			})
		})
	})

	Describe("GetServerName", func() {
		var (
			ts         *httptest.Server
			serverName *string
		)

		handlerFunc := func(w http.ResponseWriter, r *http.Request) {
			defer GinkgoRecover()

			Expect(r.Method).To(Equal("GET"))
			Expect(r.URL.Path).To(Equal("/user-data"))
			Expect(r.Header.Get("key")).To(Equal("value"))

			var jsonStr string

			if serverName == nil {
				jsonStr = `{}`
			} else {
				jsonStr = fmt.Sprintf(`{"server":{"name":"%s"}}`, *serverName)
			}

			w.Write([]byte(jsonStr))
		}

		BeforeEach(func() {
			serverName = nil

			handler := http.HandlerFunc(handlerFunc)
			ts = httptest.NewServer(handler)
			metadataService = NewHTTPMetadataService(ts.URL, metadataHeaders, "/user-data", "/instanceid", "/ssh-keys", dnsResolver, platform, logger)
		})

		AfterEach(func() {
			ts.Close()
		})

		Context("when the server name is present in the JSON", func() {
			BeforeEach(func() {
				name := "fake-server-name"
				serverName = &name
			})

			It("returns the server name", func() {
				name, err := metadataService.GetServerName()
				Expect(err).ToNot(HaveOccurred())
				Expect(name).To(Equal("fake-server-name"))
			})

			ItEnsuresMinimalNetworkSetup(func() (string, error) {
				return metadataService.GetServerName()
			})
		})

		Context("when the server name is not present in the JSON", func() {
			BeforeEach(func() {
				serverName = nil
			})

			It("returns an error", func() {
				name, err := metadataService.GetServerName()
				Expect(err).To(HaveOccurred())
				Expect(name).To(BeEmpty())
			})
		})
	})

	Describe("GetRegistryEndpoint", func() {
		var (
			ts          *httptest.Server
			registryURL *string
			dnsServer   *string
		)

		handlerFunc := func(w http.ResponseWriter, r *http.Request) {
			defer GinkgoRecover()

			Expect(r.Method).To(Equal("GET"))
			Expect(r.URL.Path).To(Equal("/user-data"))
			Expect(r.Header.Get("key")).To(Equal("value"))

			var jsonStr string

			if dnsServer == nil {
				jsonStr = fmt.Sprintf(`{"registry":{"endpoint":"%s"}}`, *registryURL)
			} else {
				jsonStr = fmt.Sprintf(`{
					"registry":{"endpoint":"%s"},
					"dns":{"nameserver":["%s"]}
				}`, *registryURL, *dnsServer)
			}

			w.Write([]byte(jsonStr))
		}

		BeforeEach(func() {
			url := "http://fake-registry.com"
			registryURL = &url
			dnsServer = nil

			handler := http.HandlerFunc(handlerFunc)
			ts = httptest.NewServer(handler)
			metadataService = NewHTTPMetadataService(ts.URL, metadataHeaders, "/user-data", "/instanceid", "/ssh-keys", dnsResolver, platform, logger)
		})

		AfterEach(func() {
			ts.Close()
		})

		ItEnsuresMinimalNetworkSetup(func() (string, error) {
			return metadataService.GetRegistryEndpoint()
		})

		Context("when metadata contains a dns server", func() {
			BeforeEach(func() {
				server := "fake-dns-server-ip"
				dnsServer = &server
			})

			Context("when registry endpoint is successfully resolved", func() {
				BeforeEach(func() {
					dnsResolver.RegisterRecord(fakeinf.FakeDNSRecord{
						DNSServers: []string{"fake-dns-server-ip"},
						Host:       "http://fake-registry.com",
						IP:         "http://fake-registry-ip",
					})
				})

				It("returns the successfully resolved registry endpoint", func() {
					endpoint, err := metadataService.GetRegistryEndpoint()
					Expect(err).ToNot(HaveOccurred())
					Expect(endpoint).To(Equal("http://fake-registry-ip"))
				})
			})

			Context("when registry endpoint is not successfully resolved", func() {
				BeforeEach(func() {
					dnsResolver.LookupHostErr = errors.New("fake-lookup-host-err")
				})

				It("returns error because it failed to resolve registry endpoint", func() {
					endpoint, err := metadataService.GetRegistryEndpoint()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-lookup-host-err"))
					Expect(endpoint).To(BeEmpty())
				})
			})
		})

		Context("when metadata does not contain dns servers", func() {
			It("returns fetched registry endpoint", func() {
				endpoint, err := metadataService.GetRegistryEndpoint()
				Expect(err).NotTo(HaveOccurred())
				Expect(endpoint).To(Equal("http://fake-registry.com"))
			})
		})
	})

	Describe("GetNetworks", func() {
		It("returns nil networks, since you don't need them for bootstrapping since your network must be set up before you can get the metadata", func() {
			Expect(metadataService.GetNetworks()).To(BeNil())
		})
	})
}
