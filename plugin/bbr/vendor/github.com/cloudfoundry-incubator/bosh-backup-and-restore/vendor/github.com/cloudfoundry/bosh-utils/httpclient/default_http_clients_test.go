package httpclient_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-utils/httpclient"
	"github.com/cloudfoundry/bosh-utils/httpclient/fakes"
)

var _ HTTPClient = &fakes.FakeHTTPClient{}

var _ = Describe("Default HTTP clients", func() {
	Describe("DefaultClient", func() {
		It("is a singleton http client", func() {
			client := DefaultClient
			Expect(client).ToNot(BeNil())
			Expect(client).To(Equal(DefaultClient))
		})

		It("disables HTTP Transport keep-alive (disables HTTP/1.[01] connection reuse)", func() {
			var client *http.Client
			client = DefaultClient

			Expect(client.Transport.(*http.Transport).DisableKeepAlives).To(Equal(true))
		})

		It("disables HTTP/2", func() {
			var client *http.Client
			client = DefaultClient

			Expect(client.Transport.(*http.Transport).TLSNextProto).ToNot(BeNil())
			Expect(client.Transport.(*http.Transport).TLSNextProto).To(BeEmpty())
		})
	})

	Describe("CreateDefaultClient", func() {
		It("enforces ssl verification", func() {
			client := CreateDefaultClient(nil)
			Expect(client.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify).To(Equal(false))
		})

		It("disables HTTP Transport keep-alive (disables HTTP/1.[01] connection reuse)", func() {
			client := CreateDefaultClient(nil)
			Expect(client.Transport.(*http.Transport).DisableKeepAlives).To(Equal(true))
		})
	})

	Describe("CreateDefaultClientInsecureSkipVerify", func() {
		It("skips ssl verification", func() {
			client := CreateDefaultClientInsecureSkipVerify()
			Expect(client.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify).To(Equal(true))
		})

		It("disables HTTP Transport keep-alive (disables HTTP/1.[01] connection reuse)", func() {
			client := CreateDefaultClientInsecureSkipVerify()
			Expect(client.Transport.(*http.Transport).DisableKeepAlives).To(Equal(true))
		})
	})
})
