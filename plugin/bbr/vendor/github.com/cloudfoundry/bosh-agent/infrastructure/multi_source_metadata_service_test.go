package infrastructure_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/infrastructure"
	fakeinf "github.com/cloudfoundry/bosh-agent/infrastructure/fakes"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
)

var _ = Describe("MultiSourceMetadataService", describeMultiSourceMetadataService)

func describeMultiSourceMetadataService() {
	var (
		metadataService MetadataService
		service1        fakeinf.FakeMetadataService
		service2        fakeinf.FakeMetadataService
	)

	BeforeEach(func() {
		service1 = fakeinf.FakeMetadataService{
			Available:        false,
			PublicKey:        "fake-public-key-1",
			InstanceID:       "fake-instance-id-1",
			ServerName:       "fake-server-name-1",
			RegistryEndpoint: "fake-registry-endpoint-1",
			Networks:         boshsettings.Networks{"net-1": boshsettings.Network{}},
		}

		service2 = fakeinf.FakeMetadataService{
			Available:        false,
			PublicKey:        "fake-public-key-2",
			InstanceID:       "fake-instance-id-2",
			ServerName:       "fake-server-name-2",
			RegistryEndpoint: "fake-registry-endpoint-2",
			Networks:         boshsettings.Networks{"net-2": boshsettings.Network{}},
		}
	})

	Context("when the first service is available", func() {
		BeforeEach(func() {
			service1.Available = true
			metadataService = NewMultiSourceMetadataService(service1, service2)
		})

		Describe("GetPublicKey", func() {
			It("returns public key from the available service", func() {
				publicKey, err := metadataService.GetPublicKey()
				Expect(err).NotTo(HaveOccurred())
				Expect(publicKey).To(Equal("fake-public-key-1"))
			})
		})

		Describe("GetInstanceID", func() {
			It("returns instance ID from the available service", func() {
				instanceID, err := metadataService.GetInstanceID()
				Expect(err).NotTo(HaveOccurred())
				Expect(instanceID).To(Equal("fake-instance-id-1"))
			})
		})

		Describe("GetServerName", func() {
			It("returns server name from the available service", func() {
				serverName, err := metadataService.GetServerName()
				Expect(err).NotTo(HaveOccurred())
				Expect(serverName).To(Equal("fake-server-name-1"))
			})
		})

		Describe("GetRegistryEndpoint", func() {
			It("returns registry endpoint from the available service", func() {
				registryEndpoint, err := metadataService.GetRegistryEndpoint()
				Expect(err).NotTo(HaveOccurred())
				Expect(registryEndpoint).To(Equal("fake-registry-endpoint-1"))
			})
		})

		Describe("GetNetworks", func() {
			It("returns network settings from the available service", func() {
				networks, err := metadataService.GetNetworks()
				Expect(err).NotTo(HaveOccurred())
				Expect(networks).To(Equal(boshsettings.Networks{"net-1": boshsettings.Network{}}))
			})
		})
	})

	Context("when the first service is unavailable", func() {
		BeforeEach(func() {
			service1.Available = false
			service2.Available = true
			metadataService = NewMultiSourceMetadataService(service1, service2)
		})

		Describe("GetPublicKey", func() {
			It("returns public key from the available service", func() {
				publicKey, err := metadataService.GetPublicKey()
				Expect(err).NotTo(HaveOccurred())
				Expect(publicKey).To(Equal("fake-public-key-2"))
			})
		})

		Describe("GetInstanceID", func() {
			It("returns instance ID from the available service", func() {
				instanceID, err := metadataService.GetInstanceID()
				Expect(err).NotTo(HaveOccurred())
				Expect(instanceID).To(Equal("fake-instance-id-2"))
			})
		})

		Describe("GetServerName", func() {
			It("returns server name from the available service", func() {
				serverName, err := metadataService.GetServerName()
				Expect(err).NotTo(HaveOccurred())
				Expect(serverName).To(Equal("fake-server-name-2"))
			})
		})

		Describe("GetRegistryEndpoint", func() {
			It("returns registry endpoint from the available service", func() {
				registryEndpoint, err := metadataService.GetRegistryEndpoint()
				Expect(err).NotTo(HaveOccurred())
				Expect(registryEndpoint).To(Equal("fake-registry-endpoint-2"))
			})
		})

		Describe("GetNetworks", func() {
			It("returns network settings from the available service", func() {
				networks, err := metadataService.GetNetworks()
				Expect(err).NotTo(HaveOccurred())
				Expect(networks).To(Equal(boshsettings.Networks{"net-2": boshsettings.Network{}}))
			})
		})
	})
}
