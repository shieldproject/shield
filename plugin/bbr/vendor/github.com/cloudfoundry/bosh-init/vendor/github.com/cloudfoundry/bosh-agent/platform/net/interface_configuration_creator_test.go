package net_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/platform/net"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("InterfaceConfigurationCreator", describeInterfaceConfigurationCreator)

func describeInterfaceConfigurationCreator() {
	var (
		interfaceConfigurationCreator   InterfaceConfigurationCreator
		staticNetwork                   boshsettings.Network
		staticNetworkWithoutMAC         boshsettings.Network
		staticNetworkWithDefaultGateway boshsettings.Network
		dhcpNetwork                     boshsettings.Network
	)

	BeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		interfaceConfigurationCreator = NewInterfaceConfigurationCreator(logger)
		dhcpNetwork = boshsettings.Network{
			Type:    "dynamic",
			Default: []string{"dns"},
			DNS:     []string{"8.8.8.8", "9.9.9.9"},
			Mac:     "fake-dhcp-mac-address",
		}
		staticNetwork = boshsettings.Network{
			IP:      "1.2.3.4",
			Netmask: "255.255.255.0",
			Gateway: "3.4.5.6",
			Mac:     "fake-static-mac-address",
		}
		staticNetworkWithDefaultGateway = boshsettings.Network{
			IP:      "5.6.7.8",
			Netmask: "255.255.255.0",
			Default: []string{"gateway"},
			Gateway: "5.6.7.1",
			Mac:     "fake-static-mac-address-with-default-gateway",
		}
		staticNetworkWithoutMAC = boshsettings.Network{
			Type:    "manual",
			IP:      "1.2.3.4",
			Netmask: "255.255.255.0",
			Gateway: "3.4.5.6",
		}
	})

	Describe("CreateInterfaceConfigurations", func() {
		var networks boshsettings.Networks
		var interfacesByMAC map[string]string

		BeforeEach(func() {
			networks = boshsettings.Networks{}
			interfacesByMAC = map[string]string{}
		})

		Context("One network", func() {
			Context("And the network has a MAC address", func() {
				BeforeEach(func() {
					networks["foo"] = staticNetwork
				})

				Context("And the MAC address matches an interface", func() {
					BeforeEach(func() {
						interfacesByMAC[staticNetwork.Mac] = "static-interface-name"
					})

					It("creates an interface configuration when matching interface exists", func() {
						staticInterfaceConfigurations, dhcpInterfaceConfigurations, err := interfaceConfigurationCreator.CreateInterfaceConfigurations(networks, interfacesByMAC)
						Expect(err).ToNot(HaveOccurred())

						Expect(staticInterfaceConfigurations).To(Equal([]StaticInterfaceConfiguration{
							StaticInterfaceConfiguration{
								Name:                "static-interface-name",
								Address:             "1.2.3.4",
								Netmask:             "255.255.255.0",
								Network:             "1.2.3.0",
								IsDefaultForGateway: false,
								Broadcast:           "1.2.3.255",
								Mac:                 "fake-static-mac-address",
								Gateway:             "3.4.5.6",
							},
						}))

						Expect(len(dhcpInterfaceConfigurations)).To(Equal(0))
					})
				})

				Context("And the MAC address has no matching an interface", func() {
					BeforeEach(func() {
						interfacesByMAC["some-other-mac"] = "static-interface-name"
					})

					It("retuns an error", func() {
						_, _, err := interfaceConfigurationCreator.CreateInterfaceConfigurations(networks, interfacesByMAC)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("No device found"))
						Expect(err.Error()).To(ContainSubstring(staticNetwork.Mac))
						Expect(err.Error()).To(ContainSubstring("foo"))
					})
				})
			})

			Context("Does not have a MAC address", func() {
				BeforeEach(func() {
					networks["foo"] = staticNetworkWithoutMAC
				})

				Context("And at least one device is available", func() {
					BeforeEach(func() {
						interfacesByMAC["fake-any-mac-address"] = "any-interface-name"
					})

					It("creates an interface configuration even with the MAC address from first interface with device", func() {
						staticInterfaceConfigurations, dhcpInterfaceConfigurations, err := interfaceConfigurationCreator.CreateInterfaceConfigurations(networks, interfacesByMAC)

						Expect(err).ToNot(HaveOccurred())

						Expect(staticInterfaceConfigurations).To(Equal([]StaticInterfaceConfiguration{
							StaticInterfaceConfiguration{
								Name:                "any-interface-name",
								Address:             "1.2.3.4",
								Netmask:             "255.255.255.0",
								Network:             "1.2.3.0",
								IsDefaultForGateway: false,
								Broadcast:           "1.2.3.255",
								Mac:                 "fake-any-mac-address",
								Gateway:             "3.4.5.6",
							},
						}))

						Expect(len(dhcpInterfaceConfigurations)).To(Equal(0))
					})
				})

				Context("And there are no network devices", func() {
					BeforeEach(func() {
						interfacesByMAC = map[string]string{}
					})

					It("retuns an error", func() {
						_, _, err := interfaceConfigurationCreator.CreateInterfaceConfigurations(networks, interfacesByMAC)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("Number of network settings '1' is greater than the number of network devices '0'"))
					})
				})
			})
		})

		Context("Multiple networks", func() {
			Context("when the number of networks matches the number of devices", func() {
				Context("and every interface has a matching networks, by MAC address", func() {
					BeforeEach(func() {
						networks["foo"] = staticNetwork
						networks["bar"] = dhcpNetwork
						networks["baz"] = staticNetworkWithDefaultGateway
						interfacesByMAC[staticNetwork.Mac] = "static-interface-name"
						interfacesByMAC[dhcpNetwork.Mac] = "dhcp-interface-name"
						interfacesByMAC[staticNetworkWithDefaultGateway.Mac] = "static-interface-name-with-default-gateway"
					})

					It("creates interface configurations for each network when matching interfaces exist", func() {
						staticInterfaceConfigurations, dhcpInterfaceConfigurations, err := interfaceConfigurationCreator.CreateInterfaceConfigurations(networks, interfacesByMAC)
						Expect(err).ToNot(HaveOccurred())

						Expect(staticInterfaceConfigurations).To(ConsistOf([]StaticInterfaceConfiguration{
							StaticInterfaceConfiguration{
								Name:                "static-interface-name",
								Address:             "1.2.3.4",
								Netmask:             "255.255.255.0",
								Network:             "1.2.3.0",
								Broadcast:           "1.2.3.255",
								IsDefaultForGateway: false,
								Mac:                 "fake-static-mac-address",
								Gateway:             "3.4.5.6",
							},
							StaticInterfaceConfiguration{
								Name:                "static-interface-name-with-default-gateway",
								Address:             "5.6.7.8",
								Netmask:             "255.255.255.0",
								Network:             "5.6.7.0",
								IsDefaultForGateway: true,
								Broadcast:           "5.6.7.255",
								Mac:                 "fake-static-mac-address-with-default-gateway",
								Gateway:             "5.6.7.1",
							},
						}))

						Expect(dhcpInterfaceConfigurations).To(Equal([]DHCPInterfaceConfiguration{
							DHCPInterfaceConfiguration{
								Name: "dhcp-interface-name",
							},
						}))
					})
				})

				Context("and some networks have no MAC address", func() {
					BeforeEach(func() {
						networks["foo"] = staticNetworkWithoutMAC
						networks["bar"] = dhcpNetwork
						interfacesByMAC["some-other-mac"] = "other-interface-name"
						interfacesByMAC[dhcpNetwork.Mac] = "dhcp-interface-name"
					})

					It("creates interface configurations for each network when matching interfaces exist, and sets non-matching interfaces as DHCP", func() {
						staticInterfaceConfigurations, dhcpInterfaceConfigurations, err := interfaceConfigurationCreator.CreateInterfaceConfigurations(networks, interfacesByMAC)
						Expect(err).ToNot(HaveOccurred())

						Expect(staticInterfaceConfigurations).To(BeEmpty())

						Expect(dhcpInterfaceConfigurations).To(ConsistOf(
							DHCPInterfaceConfiguration{
								Name: "dhcp-interface-name",
							},
							DHCPInterfaceConfiguration{
								Name: "other-interface-name",
							},
						))
					})
				})

				Context("and some networks MAC addresses that don't match", func() {
					BeforeEach(func() {
						networks["foo"] = staticNetwork
						networks["bar"] = dhcpNetwork
						interfacesByMAC["some-other-mac"] = "static-interface-name"
						interfacesByMAC[dhcpNetwork.Mac] = "dhcp-interface-name"
					})

					It("retuns an error", func() {
						_, _, err := interfaceConfigurationCreator.CreateInterfaceConfigurations(networks, interfacesByMAC)
						Expect(err).To(HaveOccurred())
					})
				})
			})
		})

		Context("when the number of networks does not match the number of devices", func() {
			BeforeEach(func() {
				networks["foo"] = staticNetwork
				networks["bar"] = dhcpNetwork
				networks["baz"] = staticNetworkWithoutMAC

				interfacesByMAC["some-other-mac"] = "static-interface-name"
				interfacesByMAC[dhcpNetwork.Mac] = "dhcp-interface-name"
			})

			It("retuns an error", func() {
				_, _, err := interfaceConfigurationCreator.CreateInterfaceConfigurations(networks, interfacesByMAC)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	It("wraps errors calculating Network and Broadcast addresses", func() {
		invalidNetwork := boshsettings.Network{
			Type:    "manual",
			IP:      "not an ip",
			Netmask: "not a valid mask",
			Mac:     "invalid-network-mac-address",
		}
		interfacesByMAC := map[string]string{
			"invalid-network-mac-address": "static-interface-name",
		}

		_, _, err := interfaceConfigurationCreator.CreateInterfaceConfigurations(boshsettings.Networks{"foo": invalidNetwork}, interfacesByMAC)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Invalid ip or netmask"))
	})
}
