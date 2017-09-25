package client_test

import (
	"io/ioutil"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/cloudfoundry/bosh-davcli/client"
	davconf "github.com/cloudfoundry/bosh-davcli/config"
	"github.com/cloudfoundry/bosh-utils/httpclient"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("Client", func() {
	var (
		server *ghttp.Server
		config davconf.Config
		client Client
		logger boshlog.Logger
	)

	BeforeEach(func() {
		server = ghttp.NewServer()
		config.Endpoint = server.URL()
		config.User = "some_user"
		config.Password = "some password"
		logger = boshlog.NewLogger(boshlog.LevelNone)
		client = NewClient(config, httpclient.DefaultClient, logger)
	})

	disconnectingRequestHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		conn, _, err := w.(http.Hijacker).Hijack()
		Expect(err).NotTo(HaveOccurred())

		conn.Close()
	})

	Describe("Exists", func() {
		It("does not return an error if file exists", func() {
			server.AppendHandlers(ghttp.RespondWith(200, ""))
			err := client.Exists("/somefile")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("the file does not exist", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.RespondWith(404, ""),
					ghttp.RespondWith(404, ""),
					ghttp.RespondWith(404, ""),
				)
			})

			It("returns an error saying blob was not found", func() {
				err := client.Exists("/somefile")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("Checking if dav blob /somefile exists: /somefile not found")))
			})
		})

		Context("unexpected http status code returned", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.RespondWith(601, ""),
					ghttp.RespondWith(601, ""),
					ghttp.RespondWith(601, ""),
				)
			})

			It("returns an error saying an unexpected error occurred", func() {
				err := client.Exists("/somefile")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("Checking if dav blob /somefile exists:")))
			})
		})
	})

	Describe("Delete", func() {
		Context("when the file does not exist", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.RespondWith(404, ""),
					ghttp.RespondWith(404, ""),
					ghttp.RespondWith(404, ""),
				)
			})

			It("does not return an error if file does not exists", func() {
				err := client.Delete("/somefile")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the file exists", func() {
			BeforeEach(func() {
				server.AppendHandlers(ghttp.RespondWith(204, ""))
			})

			It("does not return an error", func() {
				err := client.Delete("/somefile")
				Expect(err).ToNot(HaveOccurred())
				Expect(server.ReceivedRequests()).To(HaveLen(1))
				request := server.ReceivedRequests()[0]
				Expect(request.URL.Path).To(Equal("/19/somefile"))
				Expect(request.Method).To(Equal("DELETE"))
				Expect(request.Header["Authorization"]).To(Equal([]string{"Basic c29tZV91c2VyOnNvbWUgcGFzc3dvcmQ="}))
				Expect(request.Host).To(Equal(server.Addr()))
			})
		})

		Context("unexpected http status code returned", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.RespondWith(601, ""),
					ghttp.RespondWith(601, ""),
					ghttp.RespondWith(601, ""),
				)
			})

			It("returns an error saying an unexpected error occurred", func() {
				err := client.Delete("/somefile")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(Equal("Deleting blob '/somefile': Request failed, response: Response{ StatusCode: 601, Status: '601 status code 601' }")))
			})
		})
	})

	Describe("Get", func() {
		It("returns the response body from the given path", func() {
			server.AppendHandlers(ghttp.RespondWith(200, "response"))

			responseBody, err := client.Get("/")
			Expect(err).NotTo(HaveOccurred())
			buf := make([]byte, 1024)
			n, _ := responseBody.Read(buf)
			Expect(string(buf[0:n])).To(Equal("response"))
		})

		Context("when the http request fails", func() {
			BeforeEach(func() {
				server.Close()
			})

			It("returns err", func() {
				responseBody, err := client.Get("/")
				Expect(responseBody).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Getting dav blob /"))
			})
		})

		Context("when the http response code is not 200", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.RespondWith(300, "response"),
					ghttp.RespondWith(300, "response"),
					ghttp.RespondWith(300, "response"),
				)
			})

			It("returns err", func() {
				responseBody, err := client.Get("/")
				Expect(responseBody).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("Getting dav blob /: Request failed, response: Response{ StatusCode: 300, Status: '300 Multiple Choices' }")))
				Expect(server.ReceivedRequests()).To(HaveLen(3))
			})
		})
	})

	Describe("Put", func() {
		Context("When the put request succeeds", func() {
			itUploadsABlob := func() {
				body := ioutil.NopCloser(strings.NewReader("content"))
				err := client.Put("/", body, int64(7))
				Expect(err).NotTo(HaveOccurred())

				Expect(server.ReceivedRequests()).To(HaveLen(1))
				req := server.ReceivedRequests()[0]
				Expect(req.ContentLength).To(Equal(int64(7)))
			}

			It("uploads the given content if the blob does not exist", func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.RespondWith(201, ""),
						ghttp.VerifyBody([]byte("content")),
					),
				)
				itUploadsABlob()
			})

			It("uploads the given content if the blob exists", func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.RespondWith(204, ""),
						ghttp.VerifyBody([]byte("content")),
					),
				)
				itUploadsABlob()
			})
		})

		Context("when the http request fails", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					disconnectingRequestHandler,
					disconnectingRequestHandler,
					disconnectingRequestHandler,
				)
			})

			It("returns err", func() {
				body := ioutil.NopCloser(strings.NewReader("content"))
				err := client.Put("/", body, int64(7))
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("Putting dav blob /: Put %s/42: EOF", server.URL())))
				Expect(server.ReceivedRequests()).To(HaveLen(3))
			})
		})

		Context("when the http response code is not 201 or 204", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.RespondWith(300, "response"),
					ghttp.RespondWith(300, "response"),
					ghttp.RespondWith(300, "response"),
				)
			})

			It("returns err", func() {
				body := ioutil.NopCloser(strings.NewReader("content"))
				err := client.Put("/", body, int64(7))
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("Putting dav blob /: Request failed, response: Response{ StatusCode: 300, Status: '300 Multiple Choices' }")))
			})
		})
	})

	Describe("retryable count is configurable", func() {
		BeforeEach(func() {
			server.AppendHandlers(
				disconnectingRequestHandler,
				disconnectingRequestHandler,
				disconnectingRequestHandler,
				disconnectingRequestHandler,
				disconnectingRequestHandler,
				disconnectingRequestHandler,
				disconnectingRequestHandler,
			)
			config = davconf.Config{RetryAttempts: 7, Endpoint: server.URL()}
			client = NewClient(config, httpclient.DefaultClient, logger)
		})

		It("tries the specified number of times", func() {
			body := ioutil.NopCloser(strings.NewReader("content"))
			err := client.Put("/", body, int64(7))
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("Putting dav blob /: Put %s/42: EOF", server.URL())))
			Expect(server.ReceivedRequests()).To(HaveLen(7))
		})
	})
})
