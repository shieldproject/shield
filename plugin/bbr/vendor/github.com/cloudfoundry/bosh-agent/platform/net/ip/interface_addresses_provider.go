package ip

import (
	"net"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type InterfaceAddressesProvider interface {
	Get() ([]InterfaceAddress, error)
}

type systemInterfaceAddrs struct{}

func NewSystemInterfaceAddressesProvider() InterfaceAddressesProvider {
	return &systemInterfaceAddrs{}
}

func (s *systemInterfaceAddrs) Get() ([]InterfaceAddress, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return []InterfaceAddress{}, bosherr.WrapError(err, "Getting network interfaces")
	}

	interfaceAddrs := []InterfaceAddress{}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return []InterfaceAddress{}, bosherr.WrapErrorf(err, "Getting addresses of interface '%s'", iface.Name)
		}

		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				return []InterfaceAddress{}, bosherr.WrapErrorf(err, "Parsing addresses of interface '%s'", iface.Name)
			}

			if ipv4 := ip.To4(); ipv4 != nil {
				interfaceAddrs = append(interfaceAddrs, NewSimpleInterfaceAddress(iface.Name, ipv4.String()))
			}
		}

	}

	return interfaceAddrs, nil
}
