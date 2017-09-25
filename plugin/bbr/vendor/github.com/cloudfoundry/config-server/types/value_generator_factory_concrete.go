package types

import (
	"github.com/cloudfoundry/bosh-utils/errors"
)

type ValueGeneratorConcrete struct {
	loader CertsLoader
}

func NewValueGeneratorConcrete(loader CertsLoader) ValueGeneratorConcrete {
	return ValueGeneratorConcrete{loader}
}

func (vgc ValueGeneratorConcrete) GetGenerator(valueType string) (ValueGenerator, error) {
	switch valueType {
	case "password":
		return NewPasswordGenerator(), nil
	case "ssh":
		return NewSSHKeyGenerator(), nil
	case "rsa":
		return NewRSAKeyGenerator(), nil
	case "certificate":
		return NewCertificateGenerator(vgc.loader), nil
	default:
		return nil, errors.Errorf("Unsupported value type: %s", valueType)
	}
}
