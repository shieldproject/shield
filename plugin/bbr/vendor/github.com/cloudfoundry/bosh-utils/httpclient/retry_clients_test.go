package httpclient_test

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cloudfoundry/bosh-utils/httpclient"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("RetryClients", func() {

	Describe("RetryClient", func() {
		Describe("Do", func() {
			var (
				server      *ghttp.Server
				retryClient httpclient.Client
				maxAttempts int
			)

			BeforeEach(func() {
				server = ghttp.NewServer()
				logger := boshlog.NewLogger(boshlog.LevelNone)
				maxAttempts = 7

				retryClient = httpclient.NewRetryClient(httpclient.DefaultClient, uint(maxAttempts), 0, logger)
			})

			It("returns response from retryable request", func() {
				server.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/"),
					ghttp.RespondWith(201, "fake-response-body"),
				))

				req, err := http.NewRequest("GET", server.URL(), nil)
				Expect(err).NotTo(HaveOccurred())

				resp, err := retryClient.Do(req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(201))
				Expect(readString(resp.Body)).To(Equal("fake-response-body"))
			})

			It("attemps once if request is successful", func() {
				server.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/"),
					ghttp.RespondWith(200, "fake-response-body"),
				))

				req, err := http.NewRequest("GET", server.URL(), nil)
				Expect(err).NotTo(HaveOccurred())

				resp, err := retryClient.Do(req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(200))
				Expect(server.ReceivedRequests()).To(HaveLen(1))
			})

			It("retries for maxAttempts if request is failing", func() {
				server.RouteToHandler("GET", "/", ghttp.RespondWith(404, "fake-response-body"))

				req, err := http.NewRequest("GET", server.URL(), nil)
				Expect(err).NotTo(HaveOccurred())

				resp, err := retryClient.Do(req)
				Expect(err).To(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(404))
				Expect(server.ReceivedRequests()).To(HaveLen(maxAttempts))
			})
		})
	})

	Describe("NetworkSafeClient", func() {
		Describe("Do", func() {
			var (
				server      *ghttp.Server
				retryClient httpclient.Client
				maxAttempts int
			)

			BeforeEach(func() {
				server = ghttp.NewServer()
				logger := boshlog.NewLogger(boshlog.LevelNone)
				maxAttempts = 7

				retryClient = httpclient.NewNetworkSafeRetryClient(http.DefaultClient, uint(maxAttempts), 0, logger)
			})

			It("returns response from retryable request", func() {
				server.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/"),
					ghttp.RespondWith(201, "fake-response-body"),
				))

				req, err := http.NewRequest("GET", server.URL(), nil)
				Expect(err).NotTo(HaveOccurred())

				resp, err := retryClient.Do(req)
				Expect(err).ToNot(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(201))
			})

			directorErrorCodes := []int{400, 401, 403, 404, 500}
			for _, code := range directorErrorCodes {
				code := code
				It(fmt.Sprintf("attemps once if request is %d", code), func() {
					server.AppendHandlers(ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/"),
						ghttp.RespondWith(code, "fake-response-body"),
					))

					req, err := http.NewRequest("GET", server.URL(), nil)
					Expect(err).NotTo(HaveOccurred())

					resp, err := retryClient.Do(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(resp.StatusCode).To(Equal(code))

					Expect(server.ReceivedRequests()).To(HaveLen(1))
				})
			}

			for code := 200; code < 400; code++ {
				code := code
				if code >= 301 && code <= 303 {
					continue
				}
				It(fmt.Sprintf("attempts once if request is %d", code), func() {
					server.AppendHandlers(ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/"),
						ghttp.RespondWith(code, "fake-response-body"),
					))

					req, err := http.NewRequest("GET", server.URL(), nil)
					Expect(err).NotTo(HaveOccurred())

					resp, err := retryClient.Do(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(resp.StatusCode).To(Equal(code))

					Expect(server.ReceivedRequests()).To(HaveLen(1))
				})
			}

			for code := 301; code <= 303; code++ {
				code := code
				It(fmt.Sprintf("follows redirects if response is %d", code), func() {
					server.AppendHandlers(ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/"),
						ghttp.RespondWith(code, "fake-response-body", http.Header{"Location": []string{"redirected"}}),
					), ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/redirected"),
						ghttp.RespondWith(200, "fake-response-body"),
					))

					req, err := http.NewRequest("GET", server.URL(), nil)
					Expect(err).NotTo(HaveOccurred())

					resp, err := retryClient.Do(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(resp.StatusCode).To(Equal(200))

					Expect(server.ReceivedRequests()).To(HaveLen(2))
				})
			}

			Context("underlying connection errors should not be influenced by request method", func() {
				for _, method := range []string{"GET", "HEAD", "POST", "DELETE"} {
					method := method
					It(fmt.Sprintf("retries for maxAttempts with a %s request", method), func() {
						server.RouteToHandler(method, "/", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							hijacker, _, err := w.(http.Hijacker).Hijack()
							Expect(err).NotTo(HaveOccurred())
							hijacker.Close()
						}))

						req, err := http.NewRequest(method, server.URL(), nil)
						Expect(err).NotTo(HaveOccurred())

						resp, err := retryClient.Do(req)
						Expect(err).To(HaveOccurred())

						Expect(err).To(MatchError(ContainSubstring("EOF")))
						Expect(resp).To(BeNil())

						Expect(server.ReceivedRequests()).To(HaveLen(maxAttempts))
					})
				}
			})

			timeoutCodes := []int{
				http.StatusGatewayTimeout,
				http.StatusServiceUnavailable,
			}
			for _, code := range timeoutCodes {
				code := code
				for _, method := range []string{"GET", "HEAD"} {
					method := method
					Context(fmt.Sprintf("timeout http status code '%d' with %s request", code, method), func() {
						It("retries for maxAttempts", func() {
							server.RouteToHandler(method, "/", ghttp.RespondWith(code, "fake-response-body"))

							req, err := http.NewRequest(method, server.URL(), nil)
							Expect(err).NotTo(HaveOccurred())

							resp, err := retryClient.Do(req)
							Expect(err).To(HaveOccurred())

							Expect(resp.StatusCode).To(Equal(code))

							Eventually(server.ReceivedRequests, 5*time.Second).Should(HaveLen(maxAttempts))
						})
					})
				}

				for _, method := range []string{"POST", "DELETE"} {
					method := method
					Context(fmt.Sprintf("timeout http status code '%d' with %s request", code, method), func() {
						It("does not retry", func() {
							server.AppendHandlers(ghttp.CombineHandlers(
								ghttp.VerifyRequest(method, "/"),
								ghttp.RespondWith(code, "fake-response-body"),
							))

							req, err := http.NewRequest(method, server.URL(), nil)
							Expect(err).NotTo(HaveOccurred())

							resp, err := retryClient.Do(req)
							Expect(err).ToNot(HaveOccurred())

							Expect(resp.StatusCode).To(Equal(code))

							Expect(server.ReceivedRequests()).To(HaveLen(1))
						})
					})
				}
			}
		})
	})
})
