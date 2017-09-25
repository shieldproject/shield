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

	Describe("LatestCPIConfig", func() {
		It("returns latest cpi config if there is at least one", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/cpi_configs", "limit=1"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[
	{"properties": "first"},
	{"properties": "second"}
]`),
				),
			)

			cc, err := director.LatestCPIConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(cc).To(Equal(CPIConfig{Properties: "first"}))
		})

		It("returns error if there is no cpi config", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/cpi_configs", "limit=1"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[]`),
				),
			)

			_, err := director.LatestCPIConfig()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("No CPI config"))
		})

		It("returns error if info response in non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("GET", "/cpi_configs"), server)

			_, err := director.LatestCPIConfig()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Finding CPI configs: Director responded with non-successful status code"))
		})

		It("returns error if info cannot be unmarshalled", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/cpi_configs"),
					ghttp.RespondWith(http.StatusOK, ``),
				),
			)

			_, err := director.LatestCPIConfig()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Finding CPI configs: Unmarshaling Director response"))
		})
	})

	Describe("UpdateCPIConfig", func() {
		It("updates cpi config", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/cpi_configs"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"text/yaml"},
					}),
					ghttp.RespondWith(http.StatusOK, `{}`),
				),
			)

			err := director.UpdateCPIConfig([]byte("config"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if info response in non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("POST", "/cpi_configs"), server)

			err := director.UpdateCPIConfig(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Updating CPI config: Director responded with non-successful status code"))
		})
	})
})
