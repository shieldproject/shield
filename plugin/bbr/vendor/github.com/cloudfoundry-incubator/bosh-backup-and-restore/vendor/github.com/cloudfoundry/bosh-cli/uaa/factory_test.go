package uaa_test

import (
	"crypto/tls"
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/cloudfoundry/bosh-cli/uaa"
	"github.com/cloudfoundry/bosh-utils/logger/loggerfakes"
)

var _ = Describe("Factory", func() {
	Describe("New", func() {
		var logger *loggerfakes.FakeLogger

		BeforeEach(func() {
			logger = &loggerfakes.FakeLogger{}
		})

		It("returns error if config is invalid", func() {
			_, err := NewFactory(logger).New(Config{})
			Expect(err).To(HaveOccurred())
		})

		It("UAA returns error if TLS cannot be verified", func() {
			server := ghttp.NewTLSServer()
			defer server.Close()

			config, err := NewConfigFromURL(server.URL())
			Expect(err).ToNot(HaveOccurred())

			config.Client = "client"
			config.ClientSecret = "fake-client-secret"

			uaa, err := NewFactory(logger).New(config)
			Expect(err).ToNot(HaveOccurred())

			_, err = uaa.ClientCredentialsGrant()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("x509: certificate signed by unknown authority"))
		})

		It("UAA succeeds making a request with client creds if TLS can be verified", func() {
			server := ghttp.NewUnstartedServer()

			cert, err := tls.X509KeyPair(validCert, validKey)
			Expect(err).ToNot(HaveOccurred())

			server.HTTPTestServer.TLS = &tls.Config{
				Certificates: []tls.Certificate{cert},
			}

			server.HTTPTestServer.StartTLS()

			config, err := NewConfigFromURL(server.URL())
			Expect(err).ToNot(HaveOccurred())

			config.Client = "client"
			config.ClientSecret = "fake-client-secret"
			config.CACert = validCACert

			uaa, err := NewFactory(logger).New(config)
			Expect(err).ToNot(HaveOccurred())

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.VerifyBody([]byte("grant_type=client_credentials")),
					ghttp.VerifyBasicAuth("client", "fake-client-secret"),
					ghttp.VerifyHeader(http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}}),
					ghttp.RespondWith(http.StatusOK, `{}`),
				),
			)

			_, err = uaa.ClientCredentialsGrant()
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when the server url has a context path", func() {
			It("properly follows that path", func() {
				server := ghttp.NewUnstartedServer()

				cert, err := tls.X509KeyPair(validCert, validKey)
				Expect(err).ToNot(HaveOccurred())

				server.HTTPTestServer.TLS = &tls.Config{
					Certificates: []tls.Certificate{cert},
				}

				server.HTTPTestServer.StartTLS()

				config, err := NewConfigFromURL(fmt.Sprintf("%s/test_path", server.URL()))
				Expect(err).ToNot(HaveOccurred())

				config.Client = "client"
				config.ClientSecret = "fake-client-secret"
				config.CACert = validCACert

				uaa, err := NewFactory(logger).New(config)
				Expect(err).ToNot(HaveOccurred())

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/test_path/oauth/token"),
						ghttp.VerifyBody([]byte("grant_type=client_credentials")),
						ghttp.VerifyBasicAuth("client", "fake-client-secret"),
						ghttp.VerifyHeader(http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}}),
						ghttp.RespondWith(http.StatusOK, `{}`),
					),
				)

				_, err = uaa.ClientCredentialsGrant()
				Expect(err).ToNot(HaveOccurred())

			})
		})

		It("retries request 3 times if a StatusGatewayTimeout returned", func() {
			server := ghttp.NewUnstartedServer()

			cert, err := tls.X509KeyPair(validCert, validKey)
			Expect(err).ToNot(HaveOccurred())

			server.HTTPTestServer.TLS = &tls.Config{
				Certificates: []tls.Certificate{cert},
			}

			server.HTTPTestServer.StartTLS()

			config, err := NewConfigFromURL(server.URL())
			Expect(err).ToNot(HaveOccurred())

			config.Client = "client"
			config.ClientSecret = "fake-client-secret"
			config.CACert = validCACert

			uaa, err := NewFactory(logger).New(config)
			Expect(err).ToNot(HaveOccurred())

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.VerifyBody([]byte("grant_type=client_credentials")),
					ghttp.VerifyBasicAuth("client", "fake-client-secret"),
					ghttp.VerifyHeader(http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}}),
					ghttp.RespondWith(http.StatusGatewayTimeout, nil),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.VerifyBody([]byte("grant_type=client_credentials")),
					ghttp.VerifyBasicAuth("client", "fake-client-secret"),
					ghttp.VerifyHeader(http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}}),
					ghttp.RespondWith(http.StatusGatewayTimeout, nil),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.VerifyBody([]byte("grant_type=client_credentials")),
					ghttp.VerifyBasicAuth("client", "fake-client-secret"),
					ghttp.VerifyHeader(http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}}),
					ghttp.RespondWith(http.StatusOK, `{}`),
				),
			)

			_, err = uaa.ClientCredentialsGrant()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(server.ReceivedRequests())).To(Equal(3))
		})

		It("does not retry on non-successful http status codes", func() {
			server := ghttp.NewUnstartedServer()

			cert, err := tls.X509KeyPair(validCert, validKey)
			Expect(err).ToNot(HaveOccurred())

			server.HTTPTestServer.TLS = &tls.Config{
				Certificates: []tls.Certificate{cert},
			}

			server.HTTPTestServer.StartTLS()

			config, err := NewConfigFromURL(server.URL())
			Expect(err).ToNot(HaveOccurred())

			config.Client = "client"
			config.ClientSecret = "fake-client-secret"
			config.CACert = validCACert

			uaa, err := NewFactory(logger).New(config)
			Expect(err).ToNot(HaveOccurred())

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.VerifyBody([]byte("grant_type=client_credentials")),
					ghttp.VerifyBasicAuth("client", "fake-client-secret"),
					ghttp.VerifyHeader(http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}}),
					ghttp.RespondWith(http.StatusGatewayTimeout, nil),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.VerifyBody([]byte("grant_type=client_credentials")),
					ghttp.VerifyBasicAuth("client", "fake-client-secret"),
					ghttp.VerifyHeader(http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}}),
					ghttp.RespondWith(http.StatusUnauthorized, `{"no"=>"access"}`),
				),
			)

			_, err = uaa.ClientCredentialsGrant()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`Requesting token via client credentials grant: UAA responded with non-successful status code '401' response '{"no"=>"access"}'`))
			Expect(len(server.ReceivedRequests())).To(Equal(2))
		})

	})
})
