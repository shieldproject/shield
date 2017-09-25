package ip_test

import (
	. "github.com/cloudfoundry/bosh-agent/platform/net/ip"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("InterfaceAddressesProvider", func() {
	var (
		interfaceAddressesProvider InterfaceAddressesProvider
	)

	BeforeEach(func() {
		interfaceAddressesProvider = NewSystemInterfaceAddressesProvider()
	})

	It("returns current system interfaces IP addresses", func() {
		ifaces, err := interfaceAddressesProvider.Get()
		Expect(err).ToNot(HaveOccurred())

		var loopBackInterface InterfaceAddress
		for _, iface := range ifaces {
			ip, err := iface.GetIP()
			Expect(err).ToNot(HaveOccurred())

			if ip == "127.0.0.1" {
				loopBackInterface = iface
			}
		}

		Expect(loopBackInterface).ToNot(BeNil())
		// lo is on linux, lo0 on mac, Loopback Pseudo-Interface 1 windows
		Expect([]string{"lo", "lo0", "Loopback Pseudo-Interface 1"}).To(ContainElement(loopBackInterface.GetInterfaceName()))
	})
})
