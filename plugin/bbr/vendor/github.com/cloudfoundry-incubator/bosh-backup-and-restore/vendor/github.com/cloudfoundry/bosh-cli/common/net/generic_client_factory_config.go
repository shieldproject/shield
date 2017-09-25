package net

import (
	"crypto/x509"

	"github.com/cloudfoundry/bosh-utils/crypto"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type ClientFactoryConfig struct {
	Host string
	Port int

	// CA certificate is not required
	CACert string

	Client       string
	ClientSecret string
}

func (c ClientFactoryConfig) Validate() error {
	if len(c.Host) == 0 {
		return bosherr.Error("Missing 'Host'")
	}

	if c.Port == 0 {
		return bosherr.Error("Missing 'Port'")
	}

	if len(c.Client) == 0 {
		return bosherr.Error("Missing 'Client'")
	}

	if _, err := c.CACertPool(); err != nil {
		return err
	}

	return nil
}

func (c ClientFactoryConfig) CACertPool() (*x509.CertPool, error) {
	if len(c.CACert) == 0 {
		return nil, nil
	}

	return crypto.CertPoolFromPEM([]byte(c.CACert))
}
