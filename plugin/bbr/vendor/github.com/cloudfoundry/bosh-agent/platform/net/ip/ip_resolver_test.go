package ip_test

import (
	"errors"
	gonet "net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/platform/net/ip"
)

type NotIPNet struct{}

func (i NotIPNet) String() string  { return "" }
func (i NotIPNet) Network() string { return "" }

var _ = Describe("ipResolver", func() {
	var (
		ipResolver Resolver
		addrs      []gonet.Addr
		funcError  error
	)

	BeforeEach(func() {
		addrs = []gonet.Addr{}
		funcError = nil
		ifaceToAddrs := func(_ string) ([]gonet.Addr, error) { return addrs, funcError }
		ipResolver = NewResolver(ifaceToAddrs)
	})

	Describe("GetPrimaryIPv4", func() {
		Context("when interface exists", func() {
			It("returns first ipv4 address from associated interface", func() {
				addrs = []gonet.Addr{
					NotIPNet{},
					&gonet.IPNet{IP: gonet.IPv6linklocalallrouters},
					&gonet.IPNet{IP: gonet.ParseIP("127.0.0.1"), Mask: gonet.CIDRMask(16, 32)},
					&gonet.IPNet{IP: gonet.ParseIP("127.0.0.10"), Mask: gonet.CIDRMask(24, 32)},
				}

				ip, err := ipResolver.GetPrimaryIPv4("fake-iface-name")
				Expect(err).ToNot(HaveOccurred())
				Expect(ip.String()).To(Equal("127.0.0.1/16"))
			})

			It("returns error if associated interface does not have any addresses", func() {
				addrs = []gonet.Addr{}

				ip, err := ipResolver.GetPrimaryIPv4("fake-iface-name")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("No addresses found for interface"))
				Expect(ip).To(BeNil())
			})

			It("returns error if associated interface only has non-IPNet addresses", func() {
				addrs = []gonet.Addr{NotIPNet{}}

				ip, err := ipResolver.GetPrimaryIPv4("fake-iface-name")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Failed to find primary IPv4 address for interface"))
				Expect(ip).To(BeNil())
			})

			It("returns error if associated interface only has ipv6 addresses", func() {
				addrs = []gonet.Addr{&gonet.IPNet{IP: gonet.IPv6linklocalallrouters}}

				ip, err := ipResolver.GetPrimaryIPv4("fake-iface-name")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Failed to find primary IPv4 address for interface"))
				Expect(ip).To(BeNil())
			})
		})

		Context("when interface does not exist", func() {
			It("returns error", func() {
				funcError = errors.New("fake-network-func-error")

				ip, err := ipResolver.GetPrimaryIPv4("whatever")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-network-func-error"))
				Expect(err.Error()).To(ContainSubstring("Looking up addresses for interface"))
				Expect(ip).To(BeNil())
			})
		})
	})
})
