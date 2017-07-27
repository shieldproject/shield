package director_test

import (
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

	Describe("IsAuthenticated", func() {
		It("returns true if user is included in info response", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/info"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `{"user": "user"}`),
				),
			)

			authed, err := director.IsAuthenticated()
			Expect(err).ToNot(HaveOccurred())
			Expect(authed).To(BeTrue())
		})

		It("returns false if user is empty in info response", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/info"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `{"user": null}`),
				),
			)

			authed, err := director.IsAuthenticated()
			Expect(err).ToNot(HaveOccurred())
			Expect(authed).To(BeFalse())
		})

		It("returns error if info response in non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("GET", "/info"), server)

			_, err := director.IsAuthenticated()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Director responded with non-successful status code"))
		})

		It("returns error if info cannot be unmarshalled", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/info"),
					ghttp.RespondWith(http.StatusOK, ``),
				),
			)

			_, err := director.IsAuthenticated()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unmarshaling Director response"))
		})
	})

	Describe("Info", func() {
		It("returns Director info", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/info"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `{
  "name": "name",
  "uuid": "uuid",
  "version": "version",
  
  "cpi": "cpi",
  "user": "user",

  "features": {
    "snapshots": {
      "status": false
    },
    "compiled_package_cache": {
      "extras": { "provider": null },
      "status": true
    },
    "dns": {
      "extras": { "domain_name": "bosh" },
      "status": false
    }
  },

  "user_authentication": {
    "options": { "url": "https://uaa" },
    "type": "uaa"
  }
}`),
				),
			)

			info, err := director.Info()
			Expect(err).ToNot(HaveOccurred())
			Expect(info).To(Equal(Info{
				Name:    "name",
				UUID:    "uuid",
				Version: "version",

				User: "user",

				Auth: UserAuthentication{
					Type:    "uaa",
					Options: map[string]interface{}{"url": "https://uaa"},
				},

				Features: map[string]bool{
					"snapshots":              false,
					"compiled_package_cache": true,
					"dns": false,
				},

				CPI: "cpi",
			}))
		})

		It("returns error if info response in non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("GET", "/info"), server)

			_, err := director.Info()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Director responded with non-successful status code"))
		})

		It("returns error if info cannot be unmarshalled", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/info"),
					ghttp.RespondWith(http.StatusOK, ``),
				),
			)

			_, err := director.Info()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unmarshaling Director response"))
		})
	})
})
