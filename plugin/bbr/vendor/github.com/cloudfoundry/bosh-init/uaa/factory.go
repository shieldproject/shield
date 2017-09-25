package uaa

import (
	"fmt"
	"net/url"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshhttp "github.com/cloudfoundry/bosh-utils/httpclient"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type Factory struct {
	logTag string
	logger boshlog.Logger
}

func NewFactory(logger boshlog.Logger) Factory {
	return Factory{
		logTag: "uaa.Factory",
		logger: logger,
	}
}

func (f Factory) New(config Config) (UAA, error) {
	err := config.Validate()
	if err != nil {
		return UAAImpl{}, bosherr.WrapErrorf(
			err, "Validating UAA connection config")
	}

	client, err := f.httpClient(config)
	if err != nil {
		return UAAImpl{}, err
	}

	return UAAImpl{client: client}, nil
}

func (f Factory) httpClient(config Config) (Client, error) {
	certPool, err := config.CACertPool()
	if err != nil {
		return Client{}, err
	}

	if certPool == nil {
		f.logger.Debug(f.logTag, "Using default root CAs")
	} else {
		f.logger.Debug(f.logTag, "Using custom root CAs")
	}

	rawClient := boshhttp.CreateDefaultClient(certPool)

	httpClient := boshhttp.NewHTTPClient(rawClient, f.logger)

	endpoint := url.URL{
		Scheme: "https",
		Host:   fmt.Sprintf("%s:%d", config.Host, config.Port),
		User:   url.UserPassword(config.Client, config.ClientSecret),
	}

	return NewClient(endpoint.String(), httpClient, f.logger), nil
}
