package director_test

import (
	"bytes"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/cloudfoundry/bosh-cli/director"
)

var _ = Describe("Director", func() {
	var (
		director Director
		server   *ghttp.Server
	)

	BeforeEach(func() {
		director, server = BuildServer()
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("EnableResurrection", func() {
		It("enables resurrection for all instances and returns without an error", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/resurrection"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"application/json"},
					}),
					ghttp.VerifyBody([]byte(`{"resurrection_paused":false}`)),
				),
			)

			err := director.EnableResurrection(true)
			Expect(err).ToNot(HaveOccurred())
		})

		It("disables resurrection for all instances and returns without an error", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/resurrection"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"application/json"},
					}),
					ghttp.VerifyBody([]byte(`{"resurrection_paused":true}`)),
				),
			)

			err := director.EnableResurrection(false)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("PUT", "/resurrection"), server)

			err := director.EnableResurrection(true)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Changing VM resurrection state"))
		})
	})

	Describe("CleanUp", func() {
		It("cleans up all resources and returns without an error", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/cleanup"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"application/json"},
					}),
					ghttp.VerifyBody([]byte(`{"config":{"remove_all":true}}`)),
				),
				"",
				server,
			)

			err := director.CleanUp(true)
			Expect(err).ToNot(HaveOccurred())
		})

		It("cleans up some resources and returns without an error", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/cleanup"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"application/json"},
					}),
					ghttp.VerifyBody([]byte(`{"config":{"remove_all":false}}`)),
				),
				"",
				server,
			)

			err := director.CleanUp(false)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("POST", "/cleanup"), server)

			err := director.CleanUp(true)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Cleaning up resources"))
		})
	})

	Describe("DownloadResourceUnchecked", func() {
		var (
			buf *bytes.Buffer
		)

		BeforeEach(func() {
			buf = bytes.NewBufferString("")
		})

		It("writes to the writer downloaded contents and returns without an error", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/resources/blob-id"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, "result"),
				),
			)

			err := director.DownloadResourceUnchecked("blob-id", buf)
			Expect(err).ToNot(HaveOccurred())

			Expect(buf.String()).To(Equal("result"))
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("GET", "/resources/blob-id"), server)

			err := director.DownloadResourceUnchecked("blob-id", buf)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Downloading resource 'blob-id'"))
		})
	})

	Describe("With Context", func() {
		It("Adds the context id to requests", func() {
			buf := bytes.NewBufferString("")
			contextId := "example-context-id"
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/resources/blob-id"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeaderKV("X-Bosh-Context-Id", contextId),
					ghttp.RespondWith(http.StatusOK, contextId),
				),
			)

			director = director.WithContext(contextId)
			err := director.DownloadResourceUnchecked("blob-id", buf)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
