package net

import (
	"bytes"
	"path"
	"strings"
	"text/template"

	bosharp "github.com/cloudfoundry/bosh-agent/platform/net/arp"
	boship "github.com/cloudfoundry/bosh-agent/platform/net/ip"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

const centosNetManagerLogTag = "centosNetManager"

type centosNetManager struct {
	fs                            boshsys.FileSystem
	cmdRunner                     boshsys.CmdRunner
	routesSearcher                RoutesSearcher
	ipResolver                    boship.Resolver
	interfaceConfigurationCreator InterfaceConfigurationCreator
	interfaceAddressesValidator   boship.InterfaceAddressesValidator
	dnsValidator                  DNSValidator
	addressBroadcaster            bosharp.AddressBroadcaster
	logger                        boshlog.Logger
}

func NewCentosNetManager(
	fs boshsys.FileSystem,
	cmdRunner boshsys.CmdRunner,
	ipResolver boship.Resolver,
	interfaceConfigurationCreator InterfaceConfigurationCreator,
	interfaceAddressesValidator boship.InterfaceAddressesValidator,
	dnsValidator DNSValidator,
	addressBroadcaster bosharp.AddressBroadcaster,
	logger boshlog.Logger,
) Manager {
	return centosNetManager{
		fs:                            fs,
		cmdRunner:                     cmdRunner,
		ipResolver:                    ipResolver,
		interfaceConfigurationCreator: interfaceConfigurationCreator,
		interfaceAddressesValidator:   interfaceAddressesValidator,
		dnsValidator:                  dnsValidator,
		addressBroadcaster:            addressBroadcaster,
		logger:                        logger,
	}
}

func (net centosNetManager) SetupNetworking(networks boshsettings.Networks, errCh chan error) error {
	nonVipNetworks := boshsettings.Networks{}
	for networkName, networkSettings := range networks {
		if networkSettings.IsVIP() {
			continue
		}
		nonVipNetworks[networkName] = networkSettings
	}

	staticInterfaceConfigurations, dhcpInterfaceConfigurations, err := net.buildInterfaces(nonVipNetworks)
	if err != nil {
		return err
	}

	dnsNetwork, _ := nonVipNetworks.DefaultNetworkFor("dns")
	dnsServers := dnsNetwork.DNS

	interfacesChanged, err := net.writeNetworkInterfaces(dhcpInterfaceConfigurations, staticInterfaceConfigurations, dnsServers)
	if err != nil {
		return bosherr.WrapError(err, "Writing network configuration")
	}

	dhcpChanged := false
	if len(dhcpInterfaceConfigurations) > 0 {
		dhcpChanged, err = net.writeDHCPConfiguration(dnsServers, dhcpInterfaceConfigurations)
		if err != nil {
			return err
		}
	}

	if interfacesChanged || dhcpChanged {
		net.restartNetworkingInterfaces()
	}

	staticAddresses, dynamicAddresses := net.ifaceAddresses(staticInterfaceConfigurations, dhcpInterfaceConfigurations)

	err = net.interfaceAddressesValidator.Validate(staticAddresses)
	if err != nil {
		return bosherr.WrapError(err, "Validating static network configuration")
	}

	err = net.dnsValidator.Validate(dnsServers)
	if err != nil {
		return bosherr.WrapError(err, "Validating dns configuration")
	}

	net.broadcastIps(append(staticAddresses, dynamicAddresses...), errCh)

	return nil
}

func (net centosNetManager) GetConfiguredNetworkInterfaces() ([]string, error) {
	interfaces := []string{}

	interfacesByMacAddress, err := net.detectMacAddresses()
	if err != nil {
		return interfaces, bosherr.WrapError(err, "Getting network interfaces")
	}

	for _, iface := range interfacesByMacAddress {
		if net.fs.FileExists(ifcfgFilePath(iface)) {
			interfaces = append(interfaces, iface)
		}
	}

	return interfaces, nil
}

const centosDHCPIfcfgTemplate = `DEVICE={{ .Name }}
BOOTPROTO=dhcp
ONBOOT=yes
PEERDNS=yes
`

const centosStaticIfcfgTemplate = `DEVICE={{ .Name }}
BOOTPROTO=static
IPADDR={{ .Address }}
NETMASK={{ .Netmask }}
BROADCAST={{ .Broadcast }}{{if .IsDefaultForGateway}}
GATEWAY={{ .Gateway }}{{end}}
ONBOOT=yes
PEERDNS=no{{ range .DNSServers }}
DNS{{ .Index }}={{ .Address }}{{ end }}
`

type centosStaticIfcfg struct {
	*StaticInterfaceConfiguration
	DNSServers []dnsConfig
}

type dnsConfig struct {
	Index   int
	Address string
}

func newDNSConfigs(dnsServers []string) []dnsConfig {
	dnsConfigs := []dnsConfig{}
	for i := range dnsServers {
		dnsConfigs = append(dnsConfigs, dnsConfig{Index: i + 1, Address: dnsServers[i]})
	}
	return dnsConfigs
}

func ifcfgFilePath(name string) string {
	return path.Join("/etc/sysconfig/network-scripts", "ifcfg-"+name)
}

func (net centosNetManager) writeIfcfgFile(name string, t *template.Template, config interface{}) (bool, error) {
	buffer := bytes.NewBuffer([]byte{})

	err := t.Execute(buffer, config)
	if err != nil {
		return false, bosherr.WrapErrorf(err, "Generating '%s' config from template", name)
	}

	filePath := ifcfgFilePath(name)
	changed, err := net.fs.ConvergeFileContents(filePath, buffer.Bytes())
	if err != nil {
		return false, bosherr.WrapErrorf(err, "Writing config to '%s'", filePath)
	}

	return changed, nil
}

func (net centosNetManager) writeNetworkInterfaces(dhcpInterfaceConfigurations []DHCPInterfaceConfiguration, staticInterfaceConfigurations []StaticInterfaceConfiguration, dnsServers []string) (bool, error) {
	anyInterfaceChanged := false

	staticConfig := centosStaticIfcfg{}
	staticConfig.DNSServers = newDNSConfigs(dnsServers)
	staticTemplate := template.Must(template.New("ifcfg").Parse(centosStaticIfcfgTemplate))

	for i := range staticInterfaceConfigurations {
		staticConfig.StaticInterfaceConfiguration = &staticInterfaceConfigurations[i]

		changed, err := net.writeIfcfgFile(staticConfig.StaticInterfaceConfiguration.Name, staticTemplate, staticConfig)
		if err != nil {
			return false, bosherr.WrapError(err, "Writing static config")
		}

		anyInterfaceChanged = anyInterfaceChanged || changed
	}

	dhcpTemplate := template.Must(template.New("ifcfg").Parse(centosDHCPIfcfgTemplate))

	for i := range dhcpInterfaceConfigurations {
		config := &dhcpInterfaceConfigurations[i]

		changed, err := net.writeIfcfgFile(config.Name, dhcpTemplate, config)
		if err != nil {
			return false, bosherr.WrapError(err, "Writing dhcp config")
		}

		anyInterfaceChanged = anyInterfaceChanged || changed
	}

	return anyInterfaceChanged, nil
}

func (net centosNetManager) buildInterfaces(networks boshsettings.Networks) ([]StaticInterfaceConfiguration, []DHCPInterfaceConfiguration, error) {
	interfacesByMacAddress, err := net.detectMacAddresses()
	if err != nil {
		return nil, nil, bosherr.WrapError(err, "Getting network interfaces")
	}

	staticInterfaceConfigurations, dhcpInterfaceConfigurations, err := net.interfaceConfigurationCreator.CreateInterfaceConfigurations(networks, interfacesByMacAddress)

	if err != nil {
		return nil, nil, bosherr.WrapError(err, "Creating interface configurations")
	}

	return staticInterfaceConfigurations, dhcpInterfaceConfigurations, nil
}

func (net centosNetManager) broadcastIps(addresses []boship.InterfaceAddress, errCh chan error) {
	go func() {
		net.addressBroadcaster.BroadcastMACAddresses(addresses)
		if errCh != nil {
			errCh <- nil
		}
	}()
}

func (net centosNetManager) restartNetworkingInterfaces() {
	net.logger.Debug(centosNetManagerLogTag, "Restarting network interfaces")

	_, _, _, err := net.cmdRunner.RunCommand("service", "network", "restart")
	if err != nil {
		net.logger.Error(centosNetManagerLogTag, "Ignoring network restart failure: %s", err.Error())
	}
}

// DHCP Config file - /etc/dhcp3/dhclient.conf
const centosDHCPConfigTemplate = `# Generated by bosh-agent

option rfc3442-classless-static-routes code 121 = array of unsigned integer 8;

send host-name "<hostname>";

request subnet-mask, broadcast-address, time-offset, routers,
	domain-name, domain-name-servers, domain-search, host-name,
	netbios-name-servers, netbios-scope, interface-mtu,
	rfc3442-classless-static-routes, ntp-servers;
{{ if . }}
prepend domain-name-servers {{ . }};{{ end }}
`

func (net centosNetManager) writeDHCPConfiguration(dnsServers []string, dhcpInterfaceConfigurations []DHCPInterfaceConfiguration) (bool, error) {
	buffer := bytes.NewBuffer([]byte{})
	t := template.Must(template.New("dhcp-config").Parse(centosDHCPConfigTemplate))

	// Keep DNS servers in the order specified by the network
	// because they are added by a *single* DHCP's prepend command
	dnsServersList := strings.Join(dnsServers, ", ")
	err := t.Execute(buffer, dnsServersList)
	if err != nil {
		return false, bosherr.WrapError(err, "Generating config from template")
	}
	dhclientConfigFile := "/etc/dhcp/dhclient.conf"
	changed, err := net.fs.ConvergeFileContents(dhclientConfigFile, buffer.Bytes())

	if err != nil {
		return changed, bosherr.WrapErrorf(err, "Writing to %s", dhclientConfigFile)
	}

	for i := range dhcpInterfaceConfigurations {
		name := dhcpInterfaceConfigurations[i].Name
		interfaceDhclientConfigFile := path.Join("/etc/dhcp/", "dhclient-"+name+".conf")
		err = net.fs.Symlink(dhclientConfigFile, interfaceDhclientConfigFile)
		if err != nil {
			return changed, bosherr.WrapErrorf(err, "Symlinking '%s' to '%s'", interfaceDhclientConfigFile, dhclientConfigFile)
		}
	}

	return changed, nil
}

func (net centosNetManager) detectMacAddresses() (map[string]string, error) {
	addresses := map[string]string{}

	filePaths, err := net.fs.Glob("/sys/class/net/*")
	if err != nil {
		return addresses, bosherr.WrapError(err, "Getting file list from /sys/class/net")
	}

	var macAddress string
	for _, filePath := range filePaths {
		isPhysicalDevice := net.fs.FileExists(path.Join(filePath, "device"))

		if isPhysicalDevice {
			macAddress, err = net.fs.ReadFileString(path.Join(filePath, "address"))
			if err != nil {
				return addresses, bosherr.WrapError(err, "Reading mac address from file")
			}

			macAddress = strings.Trim(macAddress, "\n")

			interfaceName := path.Base(filePath)
			addresses[macAddress] = interfaceName
		}
	}

	return addresses, nil
}

func (net centosNetManager) ifaceAddresses(staticConfigs []StaticInterfaceConfiguration, dhcpConfigs []DHCPInterfaceConfiguration) ([]boship.InterfaceAddress, []boship.InterfaceAddress) {
	staticAddresses := []boship.InterfaceAddress{}
	for _, iface := range staticConfigs {
		staticAddresses = append(staticAddresses, boship.NewSimpleInterfaceAddress(iface.Name, iface.Address))
	}
	dynamicAddresses := []boship.InterfaceAddress{}
	for _, iface := range dhcpConfigs {
		dynamicAddresses = append(dynamicAddresses, boship.NewResolvingInterfaceAddress(iface.Name, net.ipResolver))
	}

	return staticAddresses, dynamicAddresses
}
