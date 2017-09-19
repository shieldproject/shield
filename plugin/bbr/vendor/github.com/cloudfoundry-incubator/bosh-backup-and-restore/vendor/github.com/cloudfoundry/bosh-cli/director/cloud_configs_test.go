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

	Describe("LatestCloudConfig", func() {
		It("returns latest cloud config if there is at least one", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/cloud_configs", "limit=1"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[
	{"properties": "first"},
	{"properties": "second"}
]`),
				),
			)

			cc, err := director.LatestCloudConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(cc).To(Equal(CloudConfig{Properties: "first"}))
		})

		It("returns error if there is no cloud config", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/cloud_configs", "limit=1"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[]`),
				),
			)

			_, err := director.LatestCloudConfig()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("No cloud config"))
		})

		It("returns error if info response in non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("GET", "/cloud_configs"), server)

			_, err := director.LatestCloudConfig()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Finding cloud configs: Director responded with non-successful status code"))
		})

		It("returns error if info cannot be unmarshalled", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/cloud_configs"),
					ghttp.RespondWith(http.StatusOK, ``),
				),
			)

			_, err := director.LatestCloudConfig()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Finding cloud configs: Unmarshaling Director response"))
		})
	})

	Describe("UpdateCloudConfig", func() {
		It("updates cloud config", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/cloud_configs"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"text/yaml"},
					}),
					ghttp.RespondWith(http.StatusOK, `{}`),
				),
			)

			err := director.UpdateCloudConfig([]byte("config"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if info response in non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("POST", "/cloud_configs"), server)

			err := director.UpdateCloudConfig(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Updating cloud config: Director responded with non-successful status code"))
		})
	})

	Describe("DiffCloudConfig", func() {
		var expectedDiffResponse CloudConfigDiff

		expectedDiffResponse = CloudConfigDiff{
			Diff: [][]interface{}{
				[]interface{}{"azs:", nil},
				[]interface{}{"- name: az2", "removed"},
				[]interface{}{"  cloud_properties: {}", "removed"},
			},
		}

		It("diffs cloud config", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/cloud_configs/diff"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"text/yaml"},
					}),
					ghttp.RespondWith(http.StatusOK, `{"diff":[["azs:",null],["- name: az2","removed"],["  cloud_properties: {}","removed"]]}`),
				),
			)

			diff, err := director.DiffCloudConfig([]byte("config"))
			Expect(err).ToNot(HaveOccurred())
			Expect(diff).To(Equal(expectedDiffResponse))
		})

		It("returns error if info response in non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("POST", "/cloud_configs/diff"), server)

			_, err := director.DiffCloudConfig(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Fetching diff result: Director responded with non-successful status code"))
		})

		It("is backwards compatible with directors without the `/diff` endpoint", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/cloud_configs/diff"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"text/yaml"},
					}),
					ghttp.RespondWith(http.StatusNotFound, ""),
				),
			)

			diff, err := director.DiffCloudConfig([]byte("config"))
			Expect(err).ToNot(HaveOccurred())
			Expect(diff).To(Equal(CloudConfigDiff{}))
		})
	})
})
