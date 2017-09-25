package net

import (
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type StaticInterfaceConfiguration struct {
	Name                string
	Address             string
	Netmask             string
	Network             string
	Broadcast           string
	IsDefaultForGateway bool
	Mac                 string
	Gateway             string
}

type StaticInterfaceConfigurations []StaticInterfaceConfiguration

func (configs StaticInterfaceConfigurations) Len() int {
	return len(configs)
}

func (configs StaticInterfaceConfigurations) Less(i, j int) bool {
	return configs[i].Name < configs[j].Name
}

func (configs StaticInterfaceConfigurations) Swap(i, j int) {
	configs[i], configs[j] = configs[j], configs[i]
}

type DHCPInterfaceConfiguration struct {
	Name string
}

type DHCPInterfaceConfigurations []DHCPInterfaceConfiguration

func (configs DHCPInterfaceConfigurations) Len() int {
	return len(configs)
}

func (configs DHCPInterfaceConfigurations) Less(i, j int) bool {
	return configs[i].Name < configs[j].Name
}

func (configs DHCPInterfaceConfigurations) Swap(i, j int) {
	configs[i], configs[j] = configs[j], configs[i]
}

type InterfaceConfigurationCreator interface {
	CreateInterfaceConfigurations(boshsettings.Networks, map[string]string) ([]StaticInterfaceConfiguration, []DHCPInterfaceConfiguration, error)
}

type interfaceConfigurationCreator struct {
	logger boshlog.Logger
	logTag string
}

func NewInterfaceConfigurationCreator(logger boshlog.Logger) InterfaceConfigurationCreator {
	return interfaceConfigurationCreator{
		logger: logger,
		logTag: "interfaceConfigurationCreator",
	}
}

func (creator interfaceConfigurationCreator) createInterfaceConfiguration(staticConfigs []StaticInterfaceConfiguration, dhcpConfigs []DHCPInterfaceConfiguration, ifaceName string, networkSettings boshsettings.Network) ([]StaticInterfaceConfiguration, []DHCPInterfaceConfiguration, error) {
	creator.logger.Debug(creator.logTag, "Creating network configuration with settings: %s", networkSettings)

	if networkSettings.IsDHCP() || networkSettings.Mac == "" {
		creator.logger.Debug(creator.logTag, "Using dhcp networking")
		dhcpConfigs = append(dhcpConfigs, DHCPInterfaceConfiguration{
			Name: ifaceName,
		})
	} else {
		creator.logger.Debug(creator.logTag, "Using static networking")
		networkAddress, broadcastAddress, err := boshsys.CalculateNetworkAndBroadcast(networkSettings.IP, networkSettings.Netmask)
		if err != nil {
			return nil, nil, bosherr.WrapError(err, "Calculating Network and Broadcast")
		}

		staticConfigs = append(staticConfigs, StaticInterfaceConfiguration{
			Name:                ifaceName,
			Address:             networkSettings.IP,
			Netmask:             networkSettings.Netmask,
			Network:             networkAddress,
			IsDefaultForGateway: networkSettings.IsDefaultFor("gateway"),
			Broadcast:           broadcastAddress,
			Mac:                 networkSettings.Mac,
			Gateway:             networkSettings.Gateway,
		})
	}
	return staticConfigs, dhcpConfigs, nil
}

func (creator interfaceConfigurationCreator) CreateInterfaceConfigurations(networks boshsettings.Networks, interfacesByMAC map[string]string) ([]StaticInterfaceConfiguration, []DHCPInterfaceConfiguration, error) {
	// In cases where we only have one network and it has no MAC address (either because the IAAS doesn't give us one or
	// it's an old CPI), if we only have one interface, we should map them
	if len(networks) == 1 && len(interfacesByMAC) == 1 {
		networkSettings := creator.getFirstNetwork(networks)
		if networkSettings.Mac == "" {
			var ifaceName string
			networkSettings.Mac, ifaceName = creator.getFirstInterface(interfacesByMAC)
			return creator.createInterfaceConfiguration([]StaticInterfaceConfiguration{}, []DHCPInterfaceConfiguration{}, ifaceName, networkSettings)
		}
	}

	return creator.createMultipleInterfaceConfigurations(networks, interfacesByMAC)
}

func (creator interfaceConfigurationCreator) createMultipleInterfaceConfigurations(networks boshsettings.Networks, interfacesByMAC map[string]string) ([]StaticInterfaceConfiguration, []DHCPInterfaceConfiguration, error) {
	if len(interfacesByMAC) < len(networks) {
		return nil, nil, bosherr.Errorf("Number of network settings '%d' is greater than the number of network devices '%d'", len(networks), len(interfacesByMAC))
	}

	for name := range networks {
		if mac := networks[name].Mac; mac != "" {
			if _, ok := interfacesByMAC[mac]; !ok {
				return nil, nil, bosherr.Errorf("No device found for network '%s' with MAC address '%s'", name, mac)
			}
		}
	}

	// Configure interfaces with network settings matching MAC address.
	// If we cannot find a network setting with a matching MAC address, configure that interface as DHCP
	var networkSettings boshsettings.Network
	var err error
	staticConfigs := []StaticInterfaceConfiguration{}
	dhcpConfigs := []DHCPInterfaceConfiguration{}

	for mac, ifaceName := range interfacesByMAC {
		networkSettings, _ = networks.NetworkForMac(mac)
		staticConfigs, dhcpConfigs, err = creator.createInterfaceConfiguration(staticConfigs, dhcpConfigs, ifaceName, networkSettings)
		if err != nil {
			return nil, nil, bosherr.WrapError(err, "Creating interface configuration")
		}
	}

	return staticConfigs, dhcpConfigs, nil
}

func (creator interfaceConfigurationCreator) getFirstNetwork(networks boshsettings.Networks) boshsettings.Network {
	for networkName := range networks {
		return networks[networkName]
	}
	return boshsettings.Network{}
}

func (creator interfaceConfigurationCreator) getFirstInterface(interfacesByMAC map[string]string) (string, string) {
	for mac := range interfacesByMAC {
		return mac, interfacesByMAC[mac]
	}
	return "", ""
}
