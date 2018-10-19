package vault

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	URL   string
	Token string
	HTTP  *http.Client
}

type Credentials struct {
	SealKey   string `json:"seal_key"`
	RootToken string `json:"root_token"`
}

func Connect(url, cacert string) (*Client, error) {
	pool := x509.NewCertPool()
	if cacert != "" {
		if ok := pool.AppendCertsFromPEM([]byte(cacert)); !ok {
			return nil, fmt.Errorf("Invalid or malformed CA Certificate")
		}
	}

	return &Client{
		URL:   url,
		Token: "",
		HTTP: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: pool,
				},
				DisableKeepAlives: true,
			},
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				req.Header.Add("X-Vault-Token", "")
				return nil
			},
		},
	}, nil
}
