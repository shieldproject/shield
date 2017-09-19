package director

import (
	"fmt"
	"net"
	"net/http"
	"net/url"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshhttp "github.com/cloudfoundry/bosh-utils/http"
	boshhttpclient "github.com/cloudfoundry/bosh-utils/httpclient"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"time"
)

type Factory struct {
	logTag string
	logger boshlog.Logger
}

func NewFactory(logger boshlog.Logger) Factory {
	return Factory{
		logTag: "director.Factory",
		logger: logger,
	}
}

func (f Factory) New(config Config, taskReporter TaskReporter, fileReporter FileReporter) (Director, error) {
	err := config.Validate()
	if err != nil {
		return DirectorImpl{}, bosherr.WrapErrorf(
			err, "Validating Director connection config")
	}

	client, err := f.httpClient(config, taskReporter, fileReporter)
	if err != nil {
		return DirectorImpl{}, err
	}

	return DirectorImpl{client: client}, nil
}

func (f Factory) httpClient(config Config, taskReporter TaskReporter, fileReporter FileReporter) (Client, error) {
	certPool, err := config.CACertPool()
	if err != nil {
		return Client{}, err
	}

	if certPool == nil {
		f.logger.Debug(f.logTag, "Using default root CAs")
	} else {
		f.logger.Debug(f.logTag, "Using custom root CAs")
	}

	rawClient := boshhttpclient.CreateDefaultClient(certPool)
	authAdjustment := NewAuthRequestAdjustment(
		config.TokenFunc, config.Client, config.ClientSecret)
	rawClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 10 {
			return bosherr.Error("Too many redirects")
		}

		// Since redirected requests are not retried,
		// forcefully adjust auth token as this is the last chance.
		err := authAdjustment.Adjust(req, true)
		if err != nil {
			return err
		}

		req.URL.Host = net.JoinHostPort(config.Host, fmt.Sprintf("%d", config.Port))

		clearHeaders(req)
		clearBody(req)

		return nil
	}

	retryClient := boshhttp.NewNetworkSafeRetryClient(rawClient, 5, 500*time.Millisecond, f.logger)

	authedClient := NewAdjustableClient(retryClient, authAdjustment)

	httpOpts := boshhttpclient.Opts{NoRedactUrlQuery: true}
	httpClient := boshhttpclient.NewHTTPClientOpts(authedClient, f.logger, httpOpts)

	endpoint := url.URL{
		Scheme: "https",
		Host:   net.JoinHostPort(config.Host, fmt.Sprintf("%d", config.Port)),
	}

	return NewClient(endpoint.String(), httpClient, taskReporter, fileReporter, f.logger), nil
}

func clearBody(req *http.Request) {
	req.Body = nil
}

func clearHeaders(req *http.Request) {
	authValue := req.Header.Get("Authorization")
	req.Header = make(map[string][]string)
	if authValue != "" {
		req.Header.Add("Authorization", authValue)
	}
}
