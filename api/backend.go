package api

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/starkandwayne/shield/cmd/shield/log"
)

var (
	curBackend      *Backend
	backendCertPool *x509.CertPool
	curClient       *http.Client
)

//Backend is all the information about a backend. It's split into
// different maps in the config, so this is all of that information
// reconstructed in one place for input and output.
type Backend struct {
	Name              string `json:"name"`
	Address           string `json:"uri"`
	Token             string `json:"-"`
	CACert            string `json:"ca_cert"`
	SkipSSLValidation bool   `json:"skip_ssl_validation"`
	APIVersion        int    `json:"-"`

	resolvedAddr string
}

//SetBackend makes all of the API calls target the given backend
func SetBackend(b *Backend) error {
	curBackend = b
	if curBackend.CACert != "" {
		backendCertPool = x509.NewCertPool()
		ok := backendCertPool.AppendCertsFromPEM([]byte(curBackend.CACert))
		if !ok {
			return fmt.Errorf("ca cert could not be added to cert pool")
		}
	}

	skipSSL := os.Getenv("SHIELD_SKIP_SSL_VERIFY") != "" || curBackend.SkipSSLValidation

	curClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: skipSSL,
				RootCAs:            backendCertPool,
			},
			Proxy:             http.ProxyFromEnvironment,
			DisableKeepAlives: true,
		},
		Timeout: 30 * time.Second,
	}

	log.DEBUG("Setting API backend: %+v", *b)

	return nil
}

//Canonize formats the backend data such that differently formatted backend datas
// that reference the same endpoint will have the same address string
func (b *Backend) Canonize() {
	b.Address = CanonizeURI(b.Address)
}

//CanonizeURI takes an input URI and normalizes it for use in API functions
func CanonizeURI(uri string) string {
	return strings.TrimSuffix(uri, "/")
}

//SecureBackendURI Hits the /v1/ping endpoint to trigger any HTTP -> HTTPS
//redirection and then returns the ultimate URL base (minus the '/v1/ping')
func (b *Backend) SecureBackendURI() (string, error) {
	if b.resolvedAddr != "" {
		return b.resolvedAddr, nil
	}

	skipSSL := os.Getenv("SHIELD_SKIP_SSL_VERIFY") != "" || b.SkipSSLValidation

	final := b.Address
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: skipSSL,
				RootCAs:            backendCertPool,
			},
			Proxy:             http.ProxyFromEnvironment,
			DisableKeepAlives: true,
		},
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			final = fmt.Sprintf("%s://%s", req.URL.Scheme, req.URL.Host)
			if len(via) > 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}
	res, err := client.Get(fmt.Sprintf("%s/v1/ping", final))
	if err != nil {
		b.resolvedAddr = final
		return final, err
	}
	defer res.Body.Close()
	io.Copy(ioutil.Discard, res.Body)
	return final, err
}
