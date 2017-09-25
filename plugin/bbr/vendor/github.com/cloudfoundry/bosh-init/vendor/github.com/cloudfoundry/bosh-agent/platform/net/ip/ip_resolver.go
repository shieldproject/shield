package ip

import (
	gonet "net"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type InterfaceToAddrsFunc func(string) ([]gonet.Addr, error)

func NetworkInterfaceToAddrsFunc(interfaceName string) ([]gonet.Addr, error) {
	iface, err := gonet.InterfaceByName(interfaceName)
	if err != nil {
		return []gonet.Addr{}, bosherr.WrapErrorf(err, "Searching for '%s' interface", interfaceName)
	}

	return iface.Addrs()
}

type Resolver interface {
	// GetPrimaryIPv4 always returns error unless IPNet is found for given interface
	GetPrimaryIPv4(interfaceName string) (*gonet.IPNet, error)
}

type ipResolver struct {
	ifaceToAddrsFunc InterfaceToAddrsFunc
}

func NewResolver(ifaceToAddrsFunc InterfaceToAddrsFunc) Resolver {
	return ipResolver{ifaceToAddrsFunc: ifaceToAddrsFunc}
}

func (r ipResolver) GetPrimaryIPv4(interfaceName string) (*gonet.IPNet, error) {
	addrs, err := r.ifaceToAddrsFunc(interfaceName)
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Looking up addresses for interface '%s'", interfaceName)
	}

	if len(addrs) == 0 {
		return nil, bosherr.Errorf("No addresses found for interface '%s'", interfaceName)
	}

	for _, addr := range addrs {
		ip, ok := addr.(*gonet.IPNet)
		if !ok {
			continue
		}

		// ignore ipv6
		if ip.IP.To4() == nil {
			continue
		}

		return ip, nil
	}

	return nil, bosherr.Errorf("Failed to find primary IPv4 address for interface '%s'", interfaceName)
}
