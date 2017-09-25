package manifest

import (
	"encoding/hex"
	"fmt"
	"net"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
)

type NetworkType string

func (n NetworkType) String() string {
	return string(n)
}

const (
	Dynamic NetworkType = "dynamic"
	Manual  NetworkType = "manual"
	VIP     NetworkType = "vip"
)

type Network struct {
	Name            string
	Type            NetworkType
	CloudProperties biproperty.Map
	DNS             []string
	Subnets         []Subnet
}

type Subnet struct {
	Range           string
	Gateway         string
	DNS             []string
	CloudProperties biproperty.Map
}

// Interface returns a property map representing a generic network interface.
// Expected Keys: ip, type, cloud properties.
// Optional Keys: netmask, gateway, dns
func (n Network) Interface(staticIPs []string, networkDefaults []NetworkDefault) (biproperty.Map, error) {
	networkInterface := biproperty.Map{
		"type": n.Type.String(),
	}

	if n.Type == Manual {
		networkInterface["gateway"] = n.Subnets[0].Gateway
		if len(n.Subnets[0].DNS) > 0 {
			networkInterface["dns"] = n.Subnets[0].DNS
		}

		_, ipNet, err := net.ParseCIDR(n.Subnets[0].Range)
		if err != nil {
			return biproperty.Map{}, bosherr.WrapError(err, "Failed to parse subnet range")
		}
		ipParts, err := hex.DecodeString(ipNet.Mask.String())
		if err != nil {
			return biproperty.Map{}, bosherr.WrapError(err, "Failed to convert subnet range to IP string")
		}
		networkInterface["netmask"] = fmt.Sprintf("%v.%v.%v.%v", ipParts[0], ipParts[1], ipParts[2], ipParts[3])

		networkInterface["cloud_properties"] = n.Subnets[0].CloudProperties
	} else {
		networkInterface["cloud_properties"] = n.CloudProperties
	}

	if n.Type == Dynamic && len(n.DNS) > 0 {
		networkInterface["dns"] = n.DNS
	}

	if len(staticIPs) > 0 {
		networkInterface["ip"] = staticIPs[0]
	}

	if len(networkDefaults) > 0 {
		networkInterface["default"] = networkDefaults
	}

	return networkInterface, nil
}
