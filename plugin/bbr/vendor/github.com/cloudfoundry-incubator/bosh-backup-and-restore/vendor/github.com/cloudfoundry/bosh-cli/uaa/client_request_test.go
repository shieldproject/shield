package uaa_test

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	boshhttp "github.com/cloudfoundry/bosh-utils/httpclient"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/cloudfoundry/bosh-cli/uaa"
)

var _ = Describe("ClientRequest", func() {
	var (
		server     *ghttp.Server
		resp       []string
		req        ClientRequest
		httpClient boshhttp.HTTPClient
		logger     boshlog.Logger
	)

	BeforeEach(func() {
		_, server = BuildServer()

		httpTransport := &http.Transport{
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			TLSHandshakeTimeout: 10 * time.Second,
		}

		rawClient := &http.Client{Transport: httpTransport}
		logger = boshlog.NewLogger(boshlog.LevelNone)
		httpClient = boshhttp.NewHTTPClient(rawClient, logger)

		resp = nil
		req = NewClientRequest(server.URL(), "", "", httpClient, logger)
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("Get", func() {
		act := func() error { return req.Get("/path", &resp) }

		It("makes request, succeeds and unmarshals response", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/path", ""),
					ghttp.VerifyHeader(http.Header{"Accept": []string{"application/json"}}),
					ghttp.RespondWith(http.StatusOK, `["val"]`),
				),
			)

			err := act()
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).To(Equal([]string{"val"}))
		})

		It("returns error if cannot be unmarshalled", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/path"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unmarshaling UAA response"))
		})

		It("returns error if response in non-successful response code", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/path"),
					ghttp.RespondWith(http.StatusBadRequest, ""),
				),
			)

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("UAA responded with non-successful status code"))
		})

		Context("authorization headers", func() {
			var (
				client       string
				clientSecret string
				headerString string
			)

			BeforeEach(func() {
				client = "zak"
				clientSecret = "is definitely the best"

				req = NewClientRequest(server.URL(), client, clientSecret, httpClient, logger)

				data := []byte(fmt.Sprintf("%s:%s", client, clientSecret))
				encodedBasicAuth := base64.StdEncoding.EncodeToString(data)
				headerString = fmt.Sprintf("Basic %s", encodedBasicAuth)
			})

			It("sends client authorization via headers", func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/path"),
						ghttp.RespondWith(http.StatusBadRequest, ""),
						ghttp.VerifyHeader(http.Header{"Authorization": []string{headerString}}),
					),
				)

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("UAA responded with non-successful status code"))

			})
		})
	})

	Describe("Post", func() {
		act := func() error { return req.Post("/path", []byte("req-body"), &resp) }

		It("makes request, succeeds and unmarshals response", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/path", ""),
					ghttp.VerifyBody([]byte("req-body")),
					ghttp.VerifyHeader(http.Header{"Accept": []string{"application/json"}}),
					ghttp.VerifyHeader(http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}}),
					ghttp.RespondWith(http.StatusOK, `["val"]`),
				),
			)

			err := act()
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).To(Equal([]string{"val"}))
		})

		It("returns error if cannot be unmarshalled", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/path"),
					ghttp.VerifyBody([]byte("req-body")),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unmarshaling UAA response"))
		})

		It("returns error if response in non-successful response code", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/path"),
					ghttp.RespondWith(http.StatusBadRequest, ""),
				),
			)

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("UAA responded with non-successful status code"))
		})

		Context("authorization headers", func() {
			var (
				client       string
				clientSecret string
				headerString string
			)

			BeforeEach(func() {
				client = "zak"
				clientSecret = "is definitely the best"

				req = NewClientRequest(server.URL(), client, clientSecret, httpClient, logger)

				data := []byte(fmt.Sprintf("%s:%s", client, clientSecret))
				encodedBasicAuth := base64.StdEncoding.EncodeToString(data)
				headerString = fmt.Sprintf("Basic %s", encodedBasicAuth)
			})

			It("sends client authorization via headers", func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/path"),
						ghttp.RespondWith(http.StatusBadRequest, ""),
						ghttp.VerifyHeader(http.Header{"Authorization": []string{headerString}}),
					),
				)

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("UAA responded with non-successful status code"))

			})
		})
	})
})
