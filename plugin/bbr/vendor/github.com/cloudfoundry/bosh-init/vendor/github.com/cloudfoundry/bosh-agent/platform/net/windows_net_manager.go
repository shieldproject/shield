package net

import (
	"fmt"
	gonet "net"
	"strings"
	"time"

	"github.com/pivotal-golang/clock"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type MACAddressDetector interface {
	MACAddresses() (map[string]string, error)
}

func NewMACAddressDetector() MACAddressDetector {
	return macAddressDetector{}
}

type macAddressDetector struct{}

func (m macAddressDetector) MACAddresses() (map[string]string, error) {
	ifs, err := gonet.Interfaces()
	if err != nil {
		return nil, bosherr.WrapError(err, "Detecting Mac Addresses")
	}
	macs := make(map[string]string, len(ifs))
	for _, f := range ifs {
		macs[f.HardwareAddr.String()] = f.Name
	}
	return macs, nil
}

type NetworkInterfaces func() ([]gonet.Interface, error)

func NewNetworkInterfaces() NetworkInterfaces {
	return gonet.Interfaces
}

type WindowsNetManager struct {
	runner                        boshsys.CmdRunner
	interfaceConfigurationCreator InterfaceConfigurationCreator
	macAddressDetector            MACAddressDetector
	logTag                        string
	logger                        boshlog.Logger
	clock                         clock.Clock
}

func NewWindowsNetManager(
	runner boshsys.CmdRunner,
	interfaceConfigurationCreator InterfaceConfigurationCreator,
	macAddressDetector MACAddressDetector,
	logger boshlog.Logger,
	clock clock.Clock,
) Manager {
	return WindowsNetManager{
		runner: runner,
		interfaceConfigurationCreator: interfaceConfigurationCreator,
		macAddressDetector:            macAddressDetector,
		logTag:                        "WindowsNetManager",
		logger:                        logger,
		clock:                         clock,
	}
}

const (
	SetDNSTemplate = `
[array]$interfaces = Get-DNSClientServerAddress
$dns = @("%s")
foreach($interface in $interfaces) {
	Set-DnsClientServerAddress -InterfaceAlias $interface.InterfaceAlias -ServerAddresses ($dns -join ",")
}
`

	ResetDNSTemplate = `
[array]$interfaces = Get-DNSClientServerAddress
foreach($interface in $interfaces) {
	Set-DnsClientServerAddress -InterfaceAlias $interface.InterfaceAlias -ResetServerAddresses
}
`

	NicSettingsTemplate = `
$connectionName=(get-wmiobject win32_networkadapter | where-object {$_.MacAddress -eq '%s'}).netconnectionid
netsh interface ip set address $connectionName static %s %s %s
`
)

func (net WindowsNetManager) GetConfiguredNetworkInterfaces() ([]string, error) {
	panic("Not implemented")
}

func (net WindowsNetManager) ComputeNetworkConfig(networks boshsettings.Networks) (
	[]StaticInterfaceConfiguration,
	[]DHCPInterfaceConfiguration,
	[]string,
	error,
) {
	nonVipNetworks := boshsettings.Networks{}
	for networkName, networkSettings := range networks {
		if networkSettings.IsVIP() {
			continue
		}
		nonVipNetworks[networkName] = networkSettings
	}

	staticConfigs, dhcpConfigs, err := net.buildInterfaces(nonVipNetworks)
	if err != nil {
		return nil, nil, nil, err
	}

	dnsNetwork, _ := nonVipNetworks.DefaultNetworkFor("dns")
	dnsServers := dnsNetwork.DNS
	return staticConfigs, dhcpConfigs, dnsServers, nil

}

func (net WindowsNetManager) SetupNetworking(networks boshsettings.Networks, errCh chan error) error {
	nonVipNetworks := boshsettings.Networks{}
	for networkName, networkSettings := range networks {
		if networkSettings.IsVIP() {
			continue
		}
		nonVipNetworks[networkName] = networkSettings
	}
	staticConfigs, _, dnsServers, err := net.ComputeNetworkConfig(networks)
	if err != nil {
		return bosherr.WrapError(err, "Computing network configuration")
	}
	err = net.setupInterfaces(staticConfigs)
	if err != nil {
		return err
	}

	dns := net.setupDNS(dnsServers)
	net.clock.Sleep(5 * time.Second)
	return dns
}

func (net WindowsNetManager) setupInterfaces(staticConfigs []StaticInterfaceConfiguration) error {
	for _, conf := range staticConfigs {
		var gateway string
		if conf.IsDefaultForGateway {
			gateway = conf.Gateway
		}

		content := fmt.Sprintf(NicSettingsTemplate, conf.Mac, conf.Address, conf.Netmask, gateway)

		_, _, _, err := net.runner.RunCommand("-Command", content)
		if err != nil {
			return bosherr.WrapError(err, "Configuring interface")
		}
	}
	return nil
}

func (net WindowsNetManager) buildInterfaces(networks boshsettings.Networks) (
	[]StaticInterfaceConfiguration,
	[]DHCPInterfaceConfiguration,
	error,
) {

	interfacesByMacAddress, err := net.macAddressDetector.MACAddresses()
	if err != nil {
		return nil, nil, bosherr.WrapError(err, "Getting network interfaces")
	}

	staticConfigs, dhcpConfigs, err := net.interfaceConfigurationCreator.CreateInterfaceConfigurations(
		networks, interfacesByMacAddress)
	if err != nil {
		return nil, nil, bosherr.WrapError(err, "Creating interface configurations")
	}

	return staticConfigs, dhcpConfigs, nil
}

func (net WindowsNetManager) setupDNS(dnsServers []string) error {
	var content string
	if len(dnsServers) > 0 {
		content = fmt.Sprintf(SetDNSTemplate, strings.Join(dnsServers, `","`))
	} else {
		content = ResetDNSTemplate
	}

	_, _, _, err := net.runner.RunCommand("-Command", content)
	if err != nil {
		return bosherr.WrapError(err, "Setting DNS servers")
	}
	return nil
}
