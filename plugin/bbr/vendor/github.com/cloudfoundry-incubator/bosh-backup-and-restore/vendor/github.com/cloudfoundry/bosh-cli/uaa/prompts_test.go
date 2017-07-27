package uaa_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/cloudfoundry/bosh-cli/uaa"
)

var _ = Describe("UAA", func() {
	var (
		uaa    UAA
		server *ghttp.Server
	)

	BeforeEach(func() {
		uaa, server = BuildServer()
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("Prompts", func() {
		It("returns list of prompts sorted with passwords showing up last", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/login"),
					ghttp.VerifyBasicAuth("client", "client-secret"),
					ghttp.VerifyHeader(http.Header{
						"Accept": []string{"application/json"},
					}),
					ghttp.RespondWith(http.StatusOK, `{
	                 	"prompts": {
	                 		"key1": ["password", "lbl"],
	                 		"key2": ["text", "lbl2"],
	                 		"key3": ["password", "lbl"]
	                 	}
	                }`),
				),
			)

			prompts, err := uaa.Prompts()
			Expect(err).ToNot(HaveOccurred())

			types := []string{prompts[0].Type, prompts[1].Type, prompts[2].Type}
			Expect(types).To(Equal([]string{"text", "password", "password"}))

			Expect(prompts).To(ConsistOf(
				Prompt{Key: "key1", Type: "password", Label: "lbl"},
				Prompt{Key: "key2", Type: "text", Label: "lbl2"},
				Prompt{Key: "key3", Type: "password", Label: "lbl"},
			))
		})

		It("returns error if prompts response in non-200", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/login"),
					ghttp.RespondWith(http.StatusBadRequest, ``),
				),
			)

			_, err := uaa.Prompts()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("UAA responded with non-successful status code"))
		})

		It("returns error if prompts cannot be unmarshalled", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/login"),
					ghttp.RespondWith(http.StatusOK, ``),
				),
			)

			_, err := uaa.Prompts()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unmarshaling UAA response"))
		})
	})
})

var _ = Describe("Prompt", func() {
	Describe("IsPassword", func() {
		It("returns true if type is 'password'", func() {
			Expect(Prompt{Type: "password"}.IsPassword()).To(BeTrue())
		})

		It("returns false if type is not 'password'", func() {
			Expect(Prompt{}.IsPassword()).To(BeFalse())
			Expect(Prompt{Type: "passwordz"}.IsPassword()).To(BeFalse())
		})
	})
})
