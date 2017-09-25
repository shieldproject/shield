package cmd_test

import (
	"fmt"
	"net/http"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	fakecmd "github.com/cloudfoundry/bosh-cli/cmd/cmdfakes"
	cmdconf "github.com/cloudfoundry/bosh-cli/cmd/config"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("SessionImpl", func() {
	var (
		context          *fakecmd.FakeSessionContext
		ui               *fakeui.FakeUI
		printEnvironment bool
		printDeployment  bool
		logger           boshlog.Logger
		sess             *SessionImpl
	)

	BeforeEach(func() {
		context = &fakecmd.FakeSessionContext{}
		ui = &fakeui.FakeUI{}
		printEnvironment = false
		printDeployment = false
		logger = boshlog.NewLogger(boshlog.LevelNone)
		sess = NewSessionImpl(context, ui, printEnvironment, printDeployment, logger)
	})

	Describe("UAA", func() {
		It("returns UAA access with client and client secret", func() {
			server, caCert := BuildSSLServer()
			defer server.Close()

			context.EnvironmentReturns(server.URL())
			context.CACertReturns(caCert)
			context.CredentialsReturns(cmdconf.Creds{Client: "client", ClientSecret: "client-secret"})

			server.AppendHandlers(
				// Anon info request to Director
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/info"),
					func(_ http.ResponseWriter, req *http.Request) {
						auth := req.Header.Get("Authorization")
						Expect(auth).To(BeEmpty(), "Authorization header must empty")
					},
					ghttp.RespondWith(http.StatusOK, fmt.Sprintf(
						`{"user_authentication":{"type":"uaa","options":{"url":"%s"}}}`, server.URL())),
				),
				// Token request to UAA
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.VerifyBody([]byte("grant_type=client_credentials")),
					ghttp.VerifyBasicAuth("client", "client-secret"),
					ghttp.VerifyHeader(http.Header{"Accept": []string{"application/json"}}),
					ghttp.VerifyHeader(http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}}),
					ghttp.RespondWith(http.StatusOK, `{"token_type":"bearer","access_token":"access-token"}`),
				),
			)

			uaa, err := sess.UAA()
			Expect(err).ToNot(HaveOccurred())

			_, err = uaa.ClientCredentialsGrant()
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns UAA access with default client (bosh_cli)", func() {
			server, caCert := BuildSSLServer()
			defer server.Close()

			context.EnvironmentReturns(server.URL())
			context.CACertReturns(caCert)

			server.AppendHandlers(
				// Anon info request to Director
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/info"),
					func(_ http.ResponseWriter, req *http.Request) {
						auth := req.Header.Get("Authorization")
						Expect(auth).To(BeEmpty(), "Authorization header must empty")
					},
					ghttp.RespondWith(http.StatusOK, fmt.Sprintf(
						`{"user_authentication":{"type":"uaa","options":{"url":"%s"}}}`, server.URL())),
				),
				// Token request to UAA
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.VerifyBody([]byte("grant_type=client_credentials")),
					ghttp.VerifyBasicAuth("bosh_cli", ""),
					ghttp.VerifyHeader(http.Header{"Accept": []string{"application/json"}}),
					ghttp.VerifyHeader(http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}}),
					ghttp.RespondWith(http.StatusOK, `{"token_type":"bearer","access_token":"access-token"}`),
				),
			)

			uaa, err := sess.UAA()
			Expect(err).ToNot(HaveOccurred())

			// Use a different request than Info
			_, err = uaa.ClientCredentialsGrant()
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if Director configuration fails", func() {
			context.EnvironmentReturns("")

			_, err := sess.UAA()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Expected non-empty Director URL"))
		})
	})

	Describe("Director", func() {
		var server *ghttp.Server
		var caCert string

		Context("director is configured for basic auth", func() {
			It("returns basic authed access to the Director", func() {
				server, caCert = BuildSSLServer()
				defer server.Close()

				context.EnvironmentReturns(server.URL())
				context.CACertReturns(caCert)
				context.CredentialsReturns(cmdconf.Creds{Client: "username", ClientSecret: "password"})

				server.AppendHandlers(
					// Anon info request to Director
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/info"),
						func(_ http.ResponseWriter, req *http.Request) {
							auth := req.Header.Get("Authorization")
							Expect(auth).To(BeEmpty(), "Authorization header must empty")
						},
						ghttp.RespondWith(http.StatusOK, fmt.Sprintf(
							`{"user_authentication":{"type":"basic","options":{}}}`)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/locks"),
						ghttp.VerifyBasicAuth("username", "password"),
						ghttp.RespondWith(http.StatusOK, "[]"),
					),
				)

				dir, err := sess.Director()
				Expect(err).ToNot(HaveOccurred())

				_, err = dir.Locks()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("director is configured for UAA client/secret login", func() {
			Context("with client and client secret", func() {
				It("returns UAA authed access to the Director", func() {
					server, caCert = BuildSSLServer()
					defer server.Close()

					context.EnvironmentReturns(server.URL())
					context.CACertReturns(caCert)
					context.CredentialsReturns(cmdconf.Creds{Client: "client", ClientSecret: "client-secret"})

					server.AppendHandlers(
						// Anon info request to Director
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/info"),
							func(_ http.ResponseWriter, req *http.Request) {
								auth := req.Header.Get("Authorization")
								Expect(auth).To(BeEmpty(), "Authorization header must empty")
							},
							ghttp.RespondWith(http.StatusOK, fmt.Sprintf(
								`{"user_authentication":{"type":"uaa","options":{"url":"%s"}}}`, server.URL())),
						),
						// Token request to UAA
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("POST", "/oauth/token"),
							ghttp.VerifyBody([]byte("grant_type=client_credentials")),
							ghttp.VerifyBasicAuth("client", "client-secret"),
							ghttp.VerifyHeader(http.Header{"Accept": []string{"application/json"}}),
							ghttp.VerifyHeader(http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}}),
							ghttp.RespondWith(http.StatusOK, `{"token_type":"bearer","access_token":"access-token"}`),
						),
						// Authed info request to Director
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/locks"),
							ghttp.VerifyHeader(http.Header{"Authorization": []string{"bearer access-token"}}),
							ghttp.RespondWith(http.StatusOK, "[]"),
						),
					)

					dir, err := sess.Director()
					Expect(err).ToNot(HaveOccurred())

					_, err = dir.Locks()
					Expect(err).ToNot(HaveOccurred())
				})
			})

			Context("with refresh token", func() {
				It("returns UAA authed access to the Director with default client (bosh_cli)", func() {
					server, caCert = BuildSSLServer()
					defer server.Close()

					context.EnvironmentReturns(server.URL())
					context.CACertReturns(caCert)
					context.CredentialsReturns(cmdconf.Creds{RefreshToken: "bearer rt-val"})

					server.AppendHandlers(
						// Anon info request to Director
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/info"),
							func(_ http.ResponseWriter, req *http.Request) {
								auth := req.Header.Get("Authorization")
								Expect(auth).To(BeEmpty(), "Authorization header must empty")
							},
							ghttp.RespondWith(http.StatusOK, fmt.Sprintf(
								`{"user_authentication":{"type":"uaa","options":{"url":"%s"}}}`, server.URL())),
						),
						// Token request to UAA
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("POST", "/oauth/token"),
							ghttp.VerifyBody([]byte("grant_type=refresh_token&refresh_token=bearer+rt-val")),
							ghttp.VerifyBasicAuth("bosh_cli", ""),
							ghttp.VerifyHeader(http.Header{"Accept": []string{"application/json"}}),
							ghttp.VerifyHeader(http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}}),
							ghttp.RespondWith(http.StatusOK, `{"token_type":"bearer","access_token":"access-token"}`),
						),
						// Authed info request to Director
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/locks"),
							ghttp.VerifyHeader(http.Header{"Authorization": []string{"bearer access-token"}}),
							ghttp.RespondWith(http.StatusOK, "[]"),
						),
					)

					dir, err := sess.Director()
					Expect(err).ToNot(HaveOccurred())

					// Use a different request than Info
					_, err = dir.Locks()
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

		It("returns error if Director configuration fails", func() {
			context.EnvironmentReturns("")

			_, err := sess.Director()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Expected non-empty Director URL"))
		})
	})

	Describe("AnonymousDirector", func() {
		It("returns Director that does not use authentication", func() {
			server, caCert := BuildSSLServer()
			defer server.Close()

			context.EnvironmentReturns(server.URL())
			context.CACertReturns(caCert)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/info"),
					func(_ http.ResponseWriter, req *http.Request) {
						auth := req.Header.Get("Authorization")
						Expect(auth).To(BeEmpty(), "Authorization header must empty")
					},
					ghttp.RespondWith(http.StatusOK, "{}"),
				),
			)

			dir, err := sess.AnonymousDirector()
			Expect(err).ToNot(HaveOccurred())

			authed, err := dir.IsAuthenticated()
			Expect(err).ToNot(HaveOccurred())

			Expect(authed).To(BeFalse())
		})

		It("returns error if Director configuration fails", func() {
			context.EnvironmentReturns("")

			_, err := sess.AnonymousDirector()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Expected non-empty Director URL"))
		})
	})

	Describe("Deployment", func() {
		It("returns deployment", func() {
			server, caCert := BuildSSLServer()
			defer server.Close()

			context.EnvironmentReturns(server.URL())
			context.CACertReturns(caCert)
			context.CredentialsReturns(cmdconf.Creds{Client: "username", ClientSecret: "password"})
			context.DeploymentReturns("config-dep")

			server.AppendHandlers(
				// Anon info request to Director
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/info"),
					func(_ http.ResponseWriter, req *http.Request) {
						auth := req.Header.Get("Authorization")
						Expect(auth).To(BeEmpty(), "Authorization header must empty")
					},
					ghttp.RespondWith(http.StatusOK, `{"user_authentication":{"type":"basic","options":{}}}`),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments/config-dep"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `{"manifest":"content"}`),
				),
			)

			dep, err := sess.Deployment()
			Expect(err).ToNot(HaveOccurred())

			_, err = dep.Manifest()
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if Director configuration fails", func() {
			context.EnvironmentReturns("")
			context.DeploymentReturns("config-dep")

			_, err := sess.Deployment()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Expected non-empty Director URL"))
		})

		It("returns error if deployment fails", func() {
			server, caCert := BuildSSLServer()
			defer server.Close()

			context.EnvironmentReturns(server.URL())
			context.CACertReturns(caCert)
			context.CredentialsReturns(cmdconf.Creds{Client: "username", ClientSecret: "password"})
			context.DeploymentReturns("")

			server.AppendHandlers(
				// Anon info request to Director
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/info"),
					func(_ http.ResponseWriter, req *http.Request) {
						auth := req.Header.Get("Authorization")
						Expect(auth).To(BeEmpty(), "Authorization header must empty")
					},
					ghttp.RespondWith(http.StatusOK, `{"user_authentication":{"type":"basic","options":{}}}`),
				),
			)

			_, err := sess.Deployment()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Expected non-empty deployment name"))
		})
	})
})
