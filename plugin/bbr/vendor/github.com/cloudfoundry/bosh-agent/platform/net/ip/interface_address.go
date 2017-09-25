package ip

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type InterfaceAddress interface {
	GetInterfaceName() string
	// GetIP gets the exposed internet protocol address of the above interface
	GetIP() (string, error)
}

type simpleInterfaceAddress struct {
	interfaceName string
	ip            string
}

func NewSimpleInterfaceAddress(interfaceName string, ip string) InterfaceAddress {
	return simpleInterfaceAddress{interfaceName: interfaceName, ip: ip}
}

func (s simpleInterfaceAddress) GetInterfaceName() string { return s.interfaceName }
func (s simpleInterfaceAddress) GetIP() (string, error)   { return s.ip, nil }

type resolvingInterfaceAddress struct {
	interfaceName string
	ipResolver    Resolver
	ip            string
}

func NewResolvingInterfaceAddress(
	interfaceName string,
	ipResolver Resolver,
) InterfaceAddress {
	return &resolvingInterfaceAddress{
		interfaceName: interfaceName,
		ipResolver:    ipResolver,
	}
}

func (s resolvingInterfaceAddress) GetInterfaceName() string { return s.interfaceName }

func (s *resolvingInterfaceAddress) GetIP() (string, error) {
	if s.ip != "" {
		return s.ip, nil
	}

	ip, err := s.ipResolver.GetPrimaryIPv4(s.interfaceName)
	if err != nil {
		return "", bosherr.WrapError(err, "Getting primary IPv4")
	}

	s.ip = ip.IP.String()

	return s.ip, nil
}
