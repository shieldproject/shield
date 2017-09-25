package server

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"github.com/cloudfoundry/config-server/store"

	"encoding/pem"
	"github.com/cloudfoundry/bosh-utils/errors"
	"github.com/cloudfoundry/config-server/types"
)

type x509Loader struct {
	store store.Store
}

func NewX509Loader(store store.Store) types.CertsLoader {
	return x509Loader{store}
}

func (l x509Loader) LoadCerts(name string) (*x509.Certificate, *rsa.PrivateKey, error) {

	configurations, err := l.store.GetByName(name)
	if err != nil {
		return nil, nil, err
	}

	if len(configurations) == 0 {
		return nil, nil, errors.Error("No certificate found")
	}

	configuration := configurations[0]

	var certContainer struct {
		CertResponse types.CertResponse `json:"value"`
	}

	err = json.Unmarshal([]byte(configuration.Value), &certContainer)
	if err != nil {
		return nil, nil, errors.WrapError(err, "Failed to parse certificate value")
	}
	certValue := certContainer.CertResponse

	if certValue.Certificate == "" || certValue.PrivateKey == "" {
		return nil, nil, errors.Errorf("Certificate %s doesn't contain expected attributes\n", name)
	}

	cpb, _ := pem.Decode([]byte(certValue.Certificate))
	rootCrt, err := x509.ParseCertificate(cpb.Bytes)
	if err != nil {
		return nil, nil, errors.WrapError(err, "Failed to parse root certificate")
	}

	kpb, _ := pem.Decode([]byte(certValue.PrivateKey))
	rootKey, err := x509.ParsePKCS1PrivateKey(kpb.Bytes)
	if err != nil {
		return nil, nil, errors.WrapError(err, "Failed to parse root private key")
	}

	return rootCrt, rootKey, nil
}
