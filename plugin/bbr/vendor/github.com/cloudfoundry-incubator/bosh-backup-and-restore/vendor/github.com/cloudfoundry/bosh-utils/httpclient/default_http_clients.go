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
	return factory{}.New(insecureSkipVerify, certPool)
}

func CreateDefaultClientInsecureSkipVerify() *http.Client {
	insecureSkipVerify := true
	return factory{}.New(insecureSkipVerify, nil)
}

type factory struct{}

func (f factory) New(insecureSkipVerify bool, certPool *x509.CertPool) *http.Client {
	defaultDialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSNextProto: map[string]func(authority string, c *tls.Conn) http.RoundTripper{},
			TLSClientConfig: &tls.Config{
				RootCAs:            certPool,
				InsecureSkipVerify: insecureSkipVerify,
			},

			Proxy: http.ProxyFromEnvironment,
			Dial:  SOCKS5DialFuncFromEnvironment(defaultDialer.Dial),

			TLSHandshakeTimeout: 30 * time.Second,
			DisableKeepAlives:   true,
		},
	}

	return client
}
