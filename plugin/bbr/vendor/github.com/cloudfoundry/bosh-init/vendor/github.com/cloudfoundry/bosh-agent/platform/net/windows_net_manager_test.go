package net_test

import (
	"errors"
	"fmt"
	"math/rand"
	gonet "net"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-golang/clock/fakeclock"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"

	. "github.com/cloudfoundry/bosh-agent/platform/net"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randomMAC() string {
	hw := make(gonet.HardwareAddr, 6)
	for i := 0; i < len(hw); i++ {
		hw[i] = byte(rand.Intn(1<<8 - 1))
	}
	return hw.String()
}

type fakeMACAddressDetector struct {
	macs map[string]string
}

func (m *fakeMACAddressDetector) MACAddresses() (map[string]string, error) {
	return m.macs, nil
}

var _ = Describe("WindowsNetManager", func() {
	var (
		clock                         *fakeclock.FakeClock
		runner                        *fakesys.FakeCmdRunner
		netManager                    Manager
		interfaceConfigurationCreator InterfaceConfigurationCreator
	)
	macAddressDetector := new(fakeMACAddressDetector)

	setupMACs := func(networks ...boshsettings.Network) error {
		m := make(map[string]string)
		for i, net := range networks {
			if net.Mac != "" {
				m[net.Mac] = fmt.Sprintf("Eth_HW %d", i)
			} else {
				m[randomMAC()] = fmt.Sprintf("Eth_Rand %d", i)
			}
		}
		macAddressDetector.macs = m
		return nil
	}

	BeforeEach(func() {
		runner = fakesys.NewFakeCmdRunner()
		clock = fakeclock.NewFakeClock(time.Now())
		logger := boshlog.NewLogger(boshlog.LevelNone)
		interfaceConfigurationCreator = NewInterfaceConfigurationCreator(logger)
		netManager = NewWindowsNetManager(runner, interfaceConfigurationCreator, macAddressDetector, logger, clock)
	})

	setupNetworking := func(networks boshsettings.Networks) error {
		// Allow 5 seconds to pass so that the Sleep() in the function can pass.
		go clock.WaitForWatcherAndIncrement(5 * time.Second)
		return netManager.SetupNetworking(networks, nil)
	}

	Describe("Setting NIC settings", func() {
		network1 := boshsettings.Network{
			Type:    "manual",
			DNS:     []string{"8.8.8.8"},
			Default: []string{"gateway", "dns"},
			IP:      "192.168.50.50",
			Gateway: "192.168.50.0",
			Netmask: "255.255.255.0",
			Mac:     "00:0C:29:0B:69:7A",
		}

		network2 := boshsettings.Network{
			Type:    "manual",
			DNS:     []string{"8.8.8.8"},
			Default: []string{},
			IP:      "192.168.20.20",
			Gateway: "192.168.20.0",
			Netmask: "255.255.255.0",
			Mac:     "99:55:C3:5A:52:7A",
		}

		vip := boshsettings.Network{
			Type: "vip",
		}

		It("sets the IP address and netmask on all interfaces, and the gateway on the default gateway interface", func() {
			setupMACs(network1, network2)
			err := setupNetworking(boshsettings.Networks{"net1": network1, "net2": network2, "vip": vip})
			Expect(err).ToNot(HaveOccurred())

			Expect(runner.RunCommands).To(
				ContainElement([]string{"-Command", fmt.Sprintf(NicSettingsTemplate, network1.Mac, network1.IP, network1.Netmask, network1.Gateway)}))
			Expect(runner.RunCommands).To(
				ContainElement([]string{"-Command", fmt.Sprintf(NicSettingsTemplate, network2.Mac, network2.IP, network2.Netmask, "")}))
		})

		It("ignores VIP networks", func() {
			err := setupNetworking(boshsettings.Networks{"vip": vip})
			Expect(err).ToNot(HaveOccurred())
			Expect(runner.RunCommands).To(ContainElement([]string{"-Command", ResetDNSTemplate}))
		})

		It("returns an error when configuring fails", func() {
			setupMACs(network1)
			runner.AddCmdResult(
				"-Command "+fmt.Sprintf(NicSettingsTemplate, network1.Mac, network1.IP, network1.Netmask, network1.Gateway),
				fakesys.FakeCmdResult{Error: errors.New("fake-err")},
			)

			err := setupNetworking(boshsettings.Networks{"static-1": network1})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Configuring interface: fake-err"))
		})
	})

	Context("when there is a network marked default for DNS", func() {
		It("configures DNS with a single DNS server", func() {
			network := boshsettings.Network{
				Type:    "manual",
				DNS:     []string{"8.8.8.8"},
				Default: []string{"gateway", "dns"},
			}

			err := setupNetworking(boshsettings.Networks{"net1": network})
			Expect(err).ToNot(HaveOccurred())

			Expect(runner.RunCommands).To(ContainElement(
				[]string{"-Command", fmt.Sprintf(SetDNSTemplate, strings.Join(network.DNS, `","`))}))
		})

		It("configures DNS with multiple DNS servers", func() {
			network := boshsettings.Network{
				Type:    "manual",
				DNS:     []string{"127.0.0.1", "8.8.8.8"},
				Default: []string{"gateway", "dns"},
			}
			err := setupNetworking(boshsettings.Networks{"manual-1": network})
			Expect(err).ToNot(HaveOccurred())

			Expect(runner.RunCommands).To(ContainElement(
				[]string{"-Command", fmt.Sprintf(SetDNSTemplate, strings.Join(network.DNS, `","`))}))
		})

		It("resets DNS without any DNS servers", func() {
			network := boshsettings.Network{
				Type:    "manual",
				Default: []string{"gateway", "dns"},
			}

			err := setupNetworking(boshsettings.Networks{"static-1": network})
			Expect(err).ToNot(HaveOccurred())

			Expect(runner.RunCommands).To(ContainElement(
				[]string{"-Command", ResetDNSTemplate}))
		})

		It("returns error if configuring DNS servers fails", func() {
			network := boshsettings.Network{
				Type:    "manual",
				DNS:     []string{"127.0.0.1", "8.8.8.8"},
				Default: []string{"gateway", "dns"},
			}

			runner.AddCmdResult(
				"-Command "+fmt.Sprintf(SetDNSTemplate, strings.Join(network.DNS, `","`)),
				fakesys.FakeCmdResult{Error: errors.New("fake-err")},
			)
			err := setupNetworking(boshsettings.Networks{"static-1": network})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Setting DNS servers: fake-err"))
		})

		It("returns error if resetting DNS servers fails", func() {
			network := boshsettings.Network{Type: "manual"}

			runner.AddCmdResult(
				"-Command "+ResetDNSTemplate,
				fakesys.FakeCmdResult{Error: errors.New("fake-err")},
			)

			err := setupNetworking(boshsettings.Networks{"static-1": network})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Setting DNS servers: fake-err"))
		})
	})

	Context("when there is no network marked default for DNS", func() {
		It("configures DNS with DNS servers if there is only one network", func() {
			network := boshsettings.Network{
				Type:    "manual",
				DNS:     []string{"127.0.0.1", "8.8.8.8"},
				Default: []string{"gateway"},
			}
			err := setupNetworking(boshsettings.Networks{"static-1": network})
			Expect(err).ToNot(HaveOccurred())

			Expect(runner.RunCommands).To(Equal(
				[][]string{[]string{"-Command", fmt.Sprintf(SetDNSTemplate, strings.Join(network.DNS, `","`))}}))
		})

		It("resets DNS without any DNS servers if there are multiple networks", func() {
			network1 := boshsettings.Network{
				Type:    "manual",
				DNS:     []string{"8.8.8.8"},
				Default: []string{"gateway"},
			}

			network2 := boshsettings.Network{
				Type:    "manual",
				DNS:     []string{"8.8.8.8"},
				Default: []string{"gateway"},
			}

			setupMACs(network1, network2)
			err := setupNetworking(boshsettings.Networks{"man-1": network1, "man-2": network2})
			Expect(err).ToNot(HaveOccurred())

			Expect(runner.RunCommands).To(Equal(
				[][]string{[]string{"-Command", ResetDNSTemplate}}))
		})
	})

	Context("when there is no non-vip network marked default for DNS", func() {
		It("resets DNS without any DNS servers", func() {
			network1 := boshsettings.Network{
				Type:    "manual",
				Default: []string{"gateway"},
			}

			network2 := boshsettings.Network{
				Type:    "vip",
				DNS:     []string{"8.8.8.8"},
				Default: []string{"gateway", "dns"},
			}

			err := setupNetworking(boshsettings.Networks{"static-1": network1, "vip-1": network2})
			Expect(err).ToNot(HaveOccurred())

			Expect(runner.RunCommands).To(Equal([][]string{[]string{"-Command", ResetDNSTemplate}}))
		})
	})

	Context("when there are no networks", func() {
		It("resets DNS", func() {
			err := setupNetworking(boshsettings.Networks{})
			Expect(err).ToNot(HaveOccurred())

			Expect(runner.RunCommands).To(Equal([][]string{[]string{"-Command", ResetDNSTemplate}}))
		})
	})
})
