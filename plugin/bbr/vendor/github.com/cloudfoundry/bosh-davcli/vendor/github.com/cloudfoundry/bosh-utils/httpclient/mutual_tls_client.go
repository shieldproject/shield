package httpclient

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"time"

	"github.com/pivotal-cf/paraphernalia/secure/tlsconfig"
)

func NewMutualTLSClient(identity tls.Certificate, caCertPool *x509.CertPool, serverName string) *http.Client {
	tlsConfig := tlsconfig.Build(
		tlsconfig.WithIdentity(identity),
		tlsconfig.WithInternalServiceDefaults(),
	)

	clientConfig := tlsConfig.Client(tlsconfig.WithAuthority(caCertPool))
	clientConfig.BuildNameToCertificate()
	clientConfig.ServerName = serverName

	transport := &http.Transport{TLSClientConfig: clientConfig}
	return &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}
}
