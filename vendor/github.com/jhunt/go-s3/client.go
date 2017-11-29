package s3

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"golang.org/x/net/proxy"
	"net/http"
)

type Client struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Bucket          string
	Domain          string
	Protocol        string
	SOCKS5Proxy     string

	SignatureVersion int

	CACertificates     []string
	SkipSystemCAs      bool
	InsecureSkipVerify bool

	ua *http.Client
}

func NewClient(c *Client) (*Client, error) {
	var (
		roots *x509.CertPool
		err   error
	)

	if c.SignatureVersion == 0 {
		c.SignatureVersion = 4
	}

	if !c.SkipSystemCAs {
		roots, err = x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve system root certificate authorities: %s", err)
		}
	} else {
		roots = x509.NewCertPool()
	}

	for _, ca := range c.CACertificates {
		if ok := roots.AppendCertsFromPEM([]byte(ca)); !ok {
			return nil, fmt.Errorf("unable to append CA certificate")
		}
	}

	dial := http.DefaultTransport.(*http.Transport).Dial
	if c.SOCKS5Proxy != "" {
		dialer, err := proxy.SOCKS5("tcp", c.SOCKS5Proxy, nil, proxy.Direct)
		if err != nil {
			return nil, err
		}
		dial = dialer.Dial
	}

	c.ua = &http.Client{
		Transport: &http.Transport{
			Dial:  dial,
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				RootCAs:            roots,
				InsecureSkipVerify: c.InsecureSkipVerify,
			},
		},
	}

	return c, nil
}

func (c *Client) domain() string {
	if c.Domain == "" {
		return "s3.amazonaws.com"
	}
	return c.Domain
}

func (c *Client) bucket() string {
	if c.Bucket == "" {
		panic("no bucket specified.")
	}
	return c.Bucket
}
