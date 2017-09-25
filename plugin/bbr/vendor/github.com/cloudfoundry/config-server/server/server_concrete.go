package server

import (
	"github.com/cloudfoundry/config-server/config"
	"github.com/cloudfoundry/config-server/store"
	"github.com/cloudfoundry/config-server/types"
	"net/http"
	"strconv"

	"github.com/cloudfoundry/bosh-utils/errors"
)

type configServer struct {
	config config.ServerConfig
}

func NewConfigServer(config config.ServerConfig) ConfigServer {
	return configServer{config: config}
}

func (cs configServer) Start() error {
	if err := cs.configureHandler(); err != nil {
		return err
	}

	return http.ListenAndServeTLS(":"+strconv.Itoa(cs.config.Port),
		cs.config.CertificateFilePath,
		cs.config.PrivateKeyFilePath, nil)
}

func (cs configServer) configureHandler() error {
	jwtTokenValidator, err := NewJwtTokenValidator(cs.config.JwtVerificationKeyPath)
	if err != nil {
		return errors.WrapError(err, "Failed to create JWT token validator")
	}

	store, err := store.CreateStore(cs.config)
	if err != nil {
		return errors.WrapError(err, "Failed to create data store")
	}

	x509Loader := NewX509Loader(store)
	requestHandler, err := NewRequestHandler(store, types.NewValueGeneratorConcrete(x509Loader))
	if err != nil {
		return errors.WrapError(err, "Failed to create Request Handler")
	}
	authenticationHandler := NewAuthenticationHandler(jwtTokenValidator, requestHandler)

	http.Handle("/v1/data", authenticationHandler)
	http.Handle("/v1/data/", authenticationHandler)

	return nil
}
