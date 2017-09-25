package factory

import (
	settings "github.com/cloudfoundry/bosh-agent/settings"
)

type Network struct {
	Type settings.NetworkType

	IP      string
	Netmask string
	Gateway string

	Default []string
	DNS     *[]string

	Mac string
}

func (n Network) Build() settings.Network {
	realNetwork := settings.Network{
		Type:    n.Type,
		IP:      defaultString(n.IP, "10.10.0.3"),
		Netmask: defaultString(n.Netmask, "255.255.254.0"),
		Gateway: defaultString(n.Gateway, "10.10.0.1"),
		Default: n.Default,
		Mac:     n.Mac,
	}

	if n.DNS == nil {
		realNetwork.DNS = []string{"10.10.0.1"}
	} else {
		realNetwork.DNS = *n.DNS
	}

	return realNetwork
}

func defaultString(s string, defaultValue string) string {
	if s == "" {
		return defaultValue
	}

	return s
}
