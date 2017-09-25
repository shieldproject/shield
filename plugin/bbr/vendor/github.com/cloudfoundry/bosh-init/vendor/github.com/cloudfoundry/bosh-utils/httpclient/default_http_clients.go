package httpclient

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"time"
)

var DefaultClient = CreateDefaultClientInsecureSkipVerify()

type Client interface {
	Do(*http.Request) (*http.Response, error)
}

func CreateDefaultClient(certPool *x509.CertPool) *http.Client {
	insecureSkipVerify := false
	return createDefaultClient(insecureSkipVerify, certPool)
}

func CreateDefaultClientInsecureSkipVerify() *http.Client {
	insecureSkipVerify := true
	return createDefaultClient(insecureSkipVerify, nil)
}

func createDefaultClient(insecureSkipVerify bool, certPool *x509.CertPool) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:            certPool,
				InsecureSkipVerify: insecureSkipVerify,
			},

			Proxy: http.ProxyFromEnvironment,

			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 0,
			}).Dial,

			TLSHandshakeTimeout: 30 * time.Second,
			DisableKeepAlives:   true,
		},
	}
}
