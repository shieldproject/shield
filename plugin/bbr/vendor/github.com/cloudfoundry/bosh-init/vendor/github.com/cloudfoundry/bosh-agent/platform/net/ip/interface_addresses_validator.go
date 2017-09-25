package ip

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type InterfaceAddressesValidator interface {
	Validate(desiredInterfaceAddresses []InterfaceAddress) error
}

type interfaceAddressesValidator struct {
	interfaceAddrsProvider InterfaceAddressesProvider
}

func NewInterfaceAddressesValidator(interfaceAddrsProvider InterfaceAddressesProvider) InterfaceAddressesValidator {
	return &interfaceAddressesValidator{
		interfaceAddrsProvider: interfaceAddrsProvider,
	}
}

func (i *interfaceAddressesValidator) Validate(desiredInterfaceAddresses []InterfaceAddress) error {
	systemInterfaceAddresses, err := i.interfaceAddrsProvider.Get()
	if err != nil {
		return bosherr.WrapError(err, "Getting network interface addresses")
	}

	for _, desiredInterfaceAddress := range desiredInterfaceAddresses {
		ifaceName := desiredInterfaceAddress.GetInterfaceName()
		iface, found := i.findInterfaceByName(ifaceName, systemInterfaceAddresses)
		if !found {
			return bosherr.WrapErrorf(err, "Validating network interface '%s' IP addresses, no interface configured with that name", ifaceName)
		}
		desiredIP, _ := desiredInterfaceAddress.GetIP()
		actualIP, _ := iface.GetIP()
		if desiredIP != actualIP {
			return bosherr.WrapErrorf(err, "Validating network interface '%s' IP addresses, expected: '%s', actual: '%s'", ifaceName, desiredIP, actualIP)
		}
	}

	return nil
}

func (i *interfaceAddressesValidator) findInterfaceByName(ifaceName string, ifaces []InterfaceAddress) (InterfaceAddress, bool) {
	for _, iface := range ifaces {
		if iface.GetInterfaceName() == ifaceName {
			return iface, true
		}
	}

	return nil, false
}
