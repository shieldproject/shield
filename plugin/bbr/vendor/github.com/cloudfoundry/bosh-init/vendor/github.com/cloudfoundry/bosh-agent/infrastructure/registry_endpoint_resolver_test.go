package infrastructure_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/infrastructure"
	fakeinf "github.com/cloudfoundry/bosh-agent/infrastructure/fakes"
)

var _ = Describe("RegistryEndpointResolver", func() {
	Describe("LookupHost", func() {
		var (
			dnsServers               []string
			registryEndpointResolver DNSResolver
			delegate                 *fakeinf.FakeDNSResolver
		)

		BeforeEach(func() {
			dnsServers = []string{"fake-dns-server-ip"}
			delegate = &fakeinf.FakeDNSResolver{}
			registryEndpointResolver = NewRegistryEndpointResolver(delegate)
		})

		Context("when registry endpoint is successfully resolved", func() {
			BeforeEach(func() {
				delegate.RegisterRecord(fakeinf.FakeDNSRecord{
					DNSServers: dnsServers,
					Host:       "fake-registry.com",
					IP:         "fake-registry-ip",
				})
			})

			Context("when registry endpoint has a port", func() {
				It("returns the successfully resolved registry endpoint with port", func() {
					resolvedEndpoint, err := registryEndpointResolver.LookupHost(dnsServers, "http://fake-registry.com:8877")
					Expect(err).ToNot(HaveOccurred())
					Expect(resolvedEndpoint).To(Equal("http://fake-registry-ip:8877"))
				})
			})

			Context("when registry endpoint does not have a port", func() {
				It("returns the successfully resolved registry endpoint", func() {
					resolvedEndpoint, err := registryEndpointResolver.LookupHost(dnsServers, "http://fake-registry.com")
					Expect(err).ToNot(HaveOccurred())
					Expect(resolvedEndpoint).To(Equal("http://fake-registry-ip"))
				})
			})
		})

		Context("when registry endpoint is not successfully resolved", func() {
			BeforeEach(func() {
				delegate.LookupHostErr = errors.New("fake-lookup-host-err")
			})

			It("returns error because it failed to resolve registry endpoint", func() {
				resolvedEndpoint, err := registryEndpointResolver.LookupHost(dnsServers, "http://fake-registry.com")
				Expect(resolvedEndpoint).To(BeEmpty())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-lookup-host-err"))
			})
		})
	})
})
