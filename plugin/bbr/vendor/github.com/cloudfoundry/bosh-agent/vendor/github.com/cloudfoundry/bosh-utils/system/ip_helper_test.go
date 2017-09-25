package system_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-utils/system"
)

var _ = Describe("CalculateNetworkAndBroadcast", func() {
	Context("invalid", func() {
		It("returns error if bad ip address", func() {
			_, _, err := CalculateNetworkAndBroadcast("192.168.195", "255.255.255.0")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Invalid IP '192.168.195'"))
		})

		It("returns error if bad netmask", func() {
			_, _, err := CalculateNetworkAndBroadcast("192.168.195.0", "255.255.255")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Invalid netmask '255.255.255'"))
		})
	})

	Context("ipv4", func() {
		It("calculates network and broadcast", func() {
			network, broadcast, err := CalculateNetworkAndBroadcast("192.168.195.6", "255.255.255.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(network).To(Equal("192.168.195.0"))
			Expect(broadcast).To(Equal("192.168.195.255"))
		})
	})

	Context("ipv6", func() {
		It("returns an empty network and broadcast", func() {
			network, broadcast, err := CalculateNetworkAndBroadcast("fd6b:6e04:558d:ebe::1", "ffff:ffff:ffff:ffff::")
			Expect(err).ToNot(HaveOccurred())
			Expect(network).To(Equal(""))
			Expect(broadcast).To(Equal(""))
		})
	})
})
