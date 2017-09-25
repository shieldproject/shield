package net

import (
	boship "github.com/cloudfoundry/bosh-agent/platform/net/ip"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
)

type dnsConfigArg struct {
	DNSServers []string
}

type customNetwork struct {
	boshsettings.Network
	Interface         string
	NetworkIP         string
	Broadcast         string
	HasDefaultGateway bool
}

type networkInterfaceConfigArg struct {
	DNSServers        []string
	Networks          []customNetwork
	HasDNSNameServers bool
}

func (c customNetwork) ToInterfaceAddress() boship.InterfaceAddress {
	return boship.NewSimpleInterfaceAddress(c.Interface, c.IP)
}

// toInterfaceAddresses bulk converts customNetworks to InterfaceAddresses
func toInterfaceAddresses(networks []customNetwork) (addresses []boship.InterfaceAddress) {
	for _, network := range networks {
		addresses = append(addresses, network.ToInterfaceAddress())
	}
	return
}
