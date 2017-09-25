package mockbosh

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"

	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"
)

func New() *mockhttp.Server {
	return mockhttp.StartServer("mock-bosh", httptest.NewServer)
}

func NewTLS() *mockhttp.Server {
	return mockhttp.StartServer("mock-bosh", httptest.NewTLSServer)
}

func NewTLSWithCert(cert tls.Certificate) *mockhttp.Server {
	return mockhttp.StartServer("mock-bosh", func(handler http.Handler) *httptest.Server {
		ts := httptest.NewUnstartedServer(handler)

		ts.TLS = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		ts.StartTLS()
		return ts
	})
}
