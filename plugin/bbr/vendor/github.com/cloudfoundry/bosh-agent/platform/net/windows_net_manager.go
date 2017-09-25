package net

import (
	"fmt"
	gonet "net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"code.cloudfoundry.org/clock"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshdirs "github.com/cloudfoundry/bosh-agent/settings/directories"
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

type WindowsNetManager struct {
	runner                        boshsys.CmdRunner
	interfaceConfigurationCreator InterfaceConfigurationCreator
	macAddressDetector            MACAddressDetector
	logTag                        string
	logger                        boshlog.Logger
	clock                         clock.Clock
	fs                            boshsys.FileSystem
	dirProvider                   boshdirs.Provider
}

func NewWindowsNetManager(
	runner boshsys.CmdRunner,
	interfaceConfigurationCreator InterfaceConfigurationCreator,
	macAddressDetector MACAddressDetector,
	logger boshlog.Logger,
	clock clock.Clock,
	fs boshsys.FileSystem,
	dirProvider boshdirs.Provider,
) Manager {
	return WindowsNetManager{
		runner: runner,
		interfaceConfigurationCreator: interfaceConfigurationCreator,
		macAddressDetector:            macAddressDetector,
		logTag:                        "WindowsNetManager",
		logger:                        logger,
		clock:                         clock,
		fs:                            fs,
		dirProvider:                   dirProvider,
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

func (net WindowsNetManager) configuredInterfacesFile() string {
	return filepath.Join(net.dirProvider.BoshDir(), "configured_interfaces.txt")
}

// GetConfiguredNetworkInterfaces returns all of the network interfaces if a
// previous call to SetupNetworking succeeded as indicated by the presence of
// a file ("configured_interfaces.txt").
//
// A file is used because there is no good way to determine if network
// interfaces are configured on Windows and SetupNetworking may be called
// during bootstrap so it is possible the agent will have restarted since
// it the last call.
//
// We return all of the network interfaces as we configure DNS for all of the
// network interfaces.  Apart from DNS, the returned network interfaces may
// not have been configured.
//
func (net WindowsNetManager) GetConfiguredNetworkInterfaces() ([]string, error) {
	// TODO (CEV): This function is only used by the ensureMinimalNetworkSetup
	// of HTTPMetadataService to determine if networks have been configured by
	// asserting the length of the returned slice is not zero.  On Linux, this
	// might be okay, but since this function is not accurate on Windows there
	// is a danger that another function will treat its output as a canonical
	// list of configured interfaces.  A better solution might be to change
	// the Platform.GetConfiguredNetworkInterfaces interface to something that
	// simply reports whether the interfaces have been configured and let each
	// OS handle determine that their own way.

	net.logger.Info(net.logTag, "Getting Configured Network Interfaces...")

	path := net.configuredInterfacesFile()
	if _, err := net.fs.Stat(path); err != nil {
		if os.IsNotExist(err) {
			net.logger.Info(net.logTag, "No network interfaces file")
			return []string{}, nil
		}
		return nil, bosherr.WrapErrorf(err, "Statting dns configuration file: %s", path)
	}

	net.logger.Info(net.logTag, "Found network interfaces file")

	ifs, err := gonet.Interfaces()
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Getting network interfaces: %s", err)
	}
	names := make([]string, 0, len(ifs))
	for _, f := range ifs {
		names = append(names, f.Name)
	}
	return names, nil
}

func (net WindowsNetManager) createConfiguredInterfacesFile() error {
	net.logger.Info(net.logTag, "Creating Configured Network Interfaces file...")

	path := net.configuredInterfacesFile()
	if _, err := net.fs.Stat(path); os.IsNotExist(err) {
		f, err := net.fs.OpenFile(path, os.O_CREATE, 0644)
		if err != nil {
			return bosherr.WrapErrorf(err, "Creating configured interfaces file: %s", err)
		}
		f.Close()
	}
	return nil
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

func (net WindowsNetManager) SetupIPv6(_ boshsettings.IPv6, _ <-chan struct{}) error { return nil }

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

	if err := net.setupInterfaces(staticConfigs); err != nil {
		return err
	}

	if err := net.setupDNS(dnsServers); err != nil {
		return err
	}
	if err := net.createConfiguredInterfacesFile(); err != nil {
		return bosherr.WrapError(err, "Writing configured network interfaces")
	}
	net.clock.Sleep(5 * time.Second)
	return nil
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
	net.logger.Info(net.logTag, "Setting up DNS...")

	var content string
	if len(dnsServers) > 0 {
		net.logger.Info(net.logTag, "Setting DNS servers: %v", dnsServers)
		content = fmt.Sprintf(SetDNSTemplate, strings.Join(dnsServers, `","`))
	} else {
		net.logger.Info(net.logTag, "Resetting DNS servers")
		content = ResetDNSTemplate
	}

	_, _, _, err := net.runner.RunCommand("-Command", content)
	if err != nil {
		return bosherr.WrapError(err, "Setting DNS servers")
	}
	return nil
}
