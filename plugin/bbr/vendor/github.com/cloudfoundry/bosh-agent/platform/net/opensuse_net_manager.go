package net

import (
	"bytes"
	"path"
	"regexp"
	"strings"
	"text/template"

	bosharp "github.com/cloudfoundry/bosh-agent/platform/net/arp"
	boship "github.com/cloudfoundry/bosh-agent/platform/net/ip"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

const opensuseNetManagerLogTag = "opensuseNetManager"

type opensuseNetManager struct {
	cmdRunner                     boshsys.CmdRunner
	fs                            boshsys.FileSystem
	routesSearcher                RoutesSearcher
	ipResolver                    boship.Resolver
	interfaceConfigurationCreator InterfaceConfigurationCreator
	interfaceAddressesValidator   boship.InterfaceAddressesValidator
	dnsValidator                  DNSValidator
	addressBroadcaster            bosharp.AddressBroadcaster
	logger                        boshlog.Logger
}

type OpenSuseDHCPInterfaceConfiguration struct {
	Name              string
	SetupDefaultRoute bool
}

func NewOpensuseNetManager(
	fs boshsys.FileSystem,
	cmdRunner boshsys.CmdRunner,
	ipResolver boship.Resolver,
	interfaceConfigurationCreator InterfaceConfigurationCreator,
	interfaceAddressesValidator boship.InterfaceAddressesValidator,
	dnsValidator DNSValidator,
	addressBroadcaster bosharp.AddressBroadcaster,
	logger boshlog.Logger,
) Manager {
	return opensuseNetManager{
		cmdRunner:                     cmdRunner,
		fs:                            fs,
		ipResolver:                    ipResolver,
		interfaceConfigurationCreator: interfaceConfigurationCreator,
		interfaceAddressesValidator:   interfaceAddressesValidator,
		dnsValidator:                  dnsValidator,
		addressBroadcaster:            addressBroadcaster,
		logger:                        logger,
	}
}

func (net opensuseNetManager) SetupIPv6(_ boshsettings.IPv6, _ <-chan struct{}) error { return nil }

func (net opensuseNetManager) SetupNetworking(networks boshsettings.Networks, errCh chan error) error {
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

func (net opensuseNetManager) GetConfiguredNetworkInterfaces() ([]string, error) {
	interfaces := []string{}

	interfacesByMacAddress, err := net.detectMacAddresses()
	if err != nil {
		return interfaces, bosherr.WrapError(err, "Getting network interfaces")
	}

	for _, iface := range interfacesByMacAddress {
		if net.fs.FileExists(net.ifcfgFilePath(iface)) {
			interfaces = append(interfaces, iface)
		}
	}

	return interfaces, nil
}

const opensuseDHCPIfcfgTemplate = `DEVICE={{ .Name }}
BOOTPROTO=dhcp
STARTMODE='auto'
DHCLIENT_SET_DEFAULT_ROUTE={{ if .SetupDefaultRoute }}yes{{ else }}no{{ end }}
`

const opensuseStaticIfcfgTemplate = `DEVICE={{ .Name }}
BOOTPROTO=static
STARTMODE='auto'
IPADDR={{ .Address }}
NETMASK={{ .Netmask }}
BROADCAST={{ .Broadcast }}
GATEWAY={{ .Gateway }}{{ range .DNSServers }}
DNS{{ .Index }}={{ .Address }}{{ end }}
`

type opensuseStaticIfcfg struct {
	*StaticInterfaceConfiguration
	DNSServers []dnsConfig
}

func (net opensuseNetManager) ifcfgFilePath(name string) string {
	return path.Join("/etc/sysconfig/network", "ifcfg-"+name)
}

func (net opensuseNetManager) writeIfcfgFile(name string, t *template.Template, config interface{}) (bool, error) {
	buffer := bytes.NewBuffer([]byte{})

	err := t.Execute(buffer, config)
	if err != nil {
		return false, bosherr.WrapErrorf(err, "Generating '%s' config from template", name)
	}

	filePath := net.ifcfgFilePath(name)
	changed, err := net.fs.ConvergeFileContents(filePath, buffer.Bytes())
	if err != nil {
		return false, bosherr.WrapErrorf(err, "Writing config to '%s'", filePath)
	}

	return changed, nil
}

func (net opensuseNetManager) writeNetworkInterfaces(dhcpInterfaceConfigurations []DHCPInterfaceConfiguration, staticInterfaceConfigurations []StaticInterfaceConfiguration, dnsServers []string) (bool, error) {
	anyInterfaceChanged := false

	staticConfig := opensuseStaticIfcfg{}
	staticConfig.DNSServers = newDNSConfigs(dnsServers)
	staticTemplate := template.Must(template.New("ifcfg").Parse(opensuseStaticIfcfgTemplate))

	for i := range staticInterfaceConfigurations {
		staticConfig.StaticInterfaceConfiguration = &staticInterfaceConfigurations[i]

		changed, err := net.writeIfcfgFile(staticConfig.StaticInterfaceConfiguration.Name, staticTemplate, staticConfig)
		if err != nil {
			return false, bosherr.WrapError(err, "Writing static config")
		}

		anyInterfaceChanged = anyInterfaceChanged || changed
	}

	dhcpTemplate := template.Must(template.New("ifcfg").Parse(opensuseDHCPIfcfgTemplate))

	setupDefaultRoute := true
	for i := range dhcpInterfaceConfigurations {
		config := OpenSuseDHCPInterfaceConfiguration{
			Name:              dhcpInterfaceConfigurations[i].Name,
			SetupDefaultRoute: setupDefaultRoute,
		}

		setupDefaultRoute = false

		changed, err := net.writeIfcfgFile(config.Name, dhcpTemplate, config)
		if err != nil {
			return false, bosherr.WrapError(err, "Writing dhcp config")
		}

		anyInterfaceChanged = anyInterfaceChanged || changed
	}

	return anyInterfaceChanged, nil
}

func (net opensuseNetManager) buildInterfaces(networks boshsettings.Networks) ([]StaticInterfaceConfiguration, []DHCPInterfaceConfiguration, error) {
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

func (net opensuseNetManager) broadcastIps(addresses []boship.InterfaceAddress, errCh chan error) {
	go func() {
		net.addressBroadcaster.BroadcastMACAddresses(addresses)
		if errCh != nil {
			errCh <- nil
		}
	}()
}

func (net opensuseNetManager) restartNetworkingInterfaces() {
	net.logger.Debug(opensuseNetManagerLogTag, "Restarting network interfaces")

	_, _, _, err := net.cmdRunner.RunCommand("service", "network", "restart")
	if err != nil {
		net.logger.Error(opensuseNetManagerLogTag, "Ignoring network restart failure: %s", err.Error())
	}
}

// DHCP Config file - /etc/dhcp3/dhclient.conf
const opensuseDHCPConfigTemplate = `# Generated by bosh-agent
WICKED_DEBUG=""
WICKED_LOG_LEVEL=""
CHECK_DUPLICATE_IP="yes"
SEND_GRATUITOUS_ARP="auto"
DEBUG="no"
WAIT_FOR_INTERFACES="30"
FIREWALL="yes"
NM_ONLINE_TIMEOUT="30"
NETCONFIG_MODULES_ORDER="dns-resolver dns-bind dns-dnsmasq nis ntp-runtime"
NETCONFIG_VERBOSE="no"
NETCONFIG_FORCE_REPLACE="no"
NETCONFIG_DNS_POLICY="auto"
NETCONFIG_DNS_FORWARDER="resolver"
NETCONFIG_DNS_FORWARDER_FALLBACK="yes"
NETCONFIG_DNS_STATIC_SEARCHLIST=""{{ if . }}
NETCONFIG_DNS_STATIC_SERVERS="{{ . }}"{{ end }}
NETCONFIG_DNS_RANKING="auto"
NETCONFIG_DNS_RESOLVER_OPTIONS=""
NETCONFIG_DNS_RESOLVER_SORTLIST=""
NETCONFIG_NTP_POLICY="auto"
NETCONFIG_NTP_STATIC_SERVERS=""
NETCONFIG_NIS_POLICY="auto"
NETCONFIG_NIS_SETDOMAINNAME="yes"
NETCONFIG_NIS_STATIC_DOMAIN=""
NETCONFIG_NIS_STATIC_SERVERS=""
WIRELESS_REGULATORY_DOMAIN=''
`

func (net opensuseNetManager) writeDHCPConfiguration(newServers []string, dhcpInterfaceConfigurations []DHCPInterfaceConfiguration) (bool, error) {
	changed := false
	dnsServers := []string{}
	buffer := bytes.NewBuffer([]byte{})
	t := template.Must(template.New("dhcp-config").Parse(opensuseDHCPConfigTemplate))

	dhclientConfigFile := "/etc/sysconfig/network/config"

	if net.fs.FileExists(dhclientConfigFile) {
		config, err := net.fs.ReadFileString(dhclientConfigFile)
		allServers := []string{}

		if err != nil {
			return changed, bosherr.WrapErrorf(err, "Reading %s", dhclientConfigFile)
		}

		re := regexp.MustCompile(`NETCONFIG_DNS_STATIC_SERVERS="(.*)"`)
		existingMatch := re.FindStringSubmatch(config)
		if len(existingMatch) == 2 && len(existingMatch[1]) > 0 {
			existingServers := strings.Split(existingMatch[1], " ")
			for i, server := range existingServers {
				existingServers[i] = strings.TrimSpace(server)
			}
			allServers = append(allServers, existingServers...)
		}

		allServers = append(allServers, newServers...)

		m := map[string]bool{}
		for _, server := range allServers {
			if _, seen := m[server]; !seen {
				dnsServers = append(dnsServers, server)
				m[server] = true
			}
		}
	} else {
		dnsServers = newServers
	}

	// Keep DNS servers in the order specified by the network
	// because they are added by a *single* DHCP's prepend command
	dnsServersList := strings.Join(dnsServers, " ")
	err := t.Execute(buffer, dnsServersList)
	if err != nil {
		return false, bosherr.WrapError(err, "Generating config from template")
	}

	changed, err = net.fs.ConvergeFileContents(dhclientConfigFile, buffer.Bytes())
	if err != nil {
		return changed, bosherr.WrapErrorf(err, "Writing to %s", dhclientConfigFile)
	}

	if changed {
		_, _, _, err := net.cmdRunner.RunCommand("rm", "/etc/resolv.conf")
		if err != nil {
			net.logger.Error(opensuseNetManagerLogTag, "Failed to remove /etc/resolv.conf: %s", err.Error())
		}
	}

	return changed, nil
}

func (net opensuseNetManager) detectMacAddresses() (map[string]string, error) {
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

func (net opensuseNetManager) ifaceAddresses(staticConfigs []StaticInterfaceConfiguration, dhcpConfigs []DHCPInterfaceConfiguration) ([]boship.InterfaceAddress, []boship.InterfaceAddress) {
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
