package uaa

import (
	boshhttp "github.com/cloudfoundry/bosh-utils/httpclient"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type Client struct {
	clientRequest ClientRequest
}

func NewClient(endpoint string, client string, clientSecret string, httpClient boshhttp.HTTPClient, logger boshlog.Logger) Client {
	return Client{NewClientRequest(endpoint, client, clientSecret, httpClient, logger)}
}
