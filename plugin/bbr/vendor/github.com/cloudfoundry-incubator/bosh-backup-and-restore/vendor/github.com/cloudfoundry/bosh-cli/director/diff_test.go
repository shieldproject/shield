package director_test

import (
	"net/http"

	. "github.com/cloudfoundry/bosh-cli/director"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Deployment", func() {
	Describe("Diff", func() {

		var (
			deployment Deployment
			server     *ghttp.Server
		)

		BeforeEach(func() {
			var director Director
			director, server = BuildServer()

			var err error
			deployment, err = director.FindDeployment("dep1")
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			server.Close()
		})

		Context("when diffing manifest with 'no redact' option", func() {
			var expectedDiffResponse DeploymentDiffResponse

			expectedDiffResponse = DeploymentDiffResponse{
				Context: map[string]interface{}{
					"cloud_config_id":   2,
					"runtime_config_id": nil,
				},
				Diff: [][]interface{}{
					[]interface{}{"name: simple manifest", nil},
					[]interface{}{"properties:", nil},
					[]interface{}{"  - property1", "removed"},
					[]interface{}{"  - property2", "added"},
				},
			}

			It("returns non-redacted diff result if redact is false", func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/deployments/dep1/diff"),
						ghttp.VerifyFormKV("redact", "false"),
						ghttp.VerifyBasicAuth("username", "password"),
						ghttp.VerifyHeader(http.Header{
							"Content-Type": []string{"text/yaml"},
						}),
						ghttp.VerifyBody([]byte("manifest")),
						ghttp.RespondWithJSONEncoded(http.StatusOK, expectedDiffResponse),
					),
				)

				diff, err := deployment.Diff([]byte("manifest"), true)
				Expect(err).ToNot(HaveOccurred())
				Expect(diff.Diff).To(Equal(expectedDiffResponse.Diff))
			})

			It("returns redacted diff if redact is true", func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/deployments/dep1/diff"),
						ghttp.VerifyFormKV("redact", "true"),
						ghttp.VerifyBasicAuth("username", "password"),
						ghttp.VerifyHeader(http.Header{
							"Content-Type": []string{"text/yaml"},
						}),
						ghttp.VerifyBody([]byte("manifest")),
						ghttp.RespondWithJSONEncoded(http.StatusOK, expectedDiffResponse),
					),
				)

				diff, err := deployment.Diff([]byte("manifest"), false)
				Expect(err).ToNot(HaveOccurred())
				Expect(diff.Diff).To(Equal(expectedDiffResponse.Diff))
			})

			It("returns error if response is non-200", func() {
				AppendBadRequest(ghttp.VerifyRequest("POST", "/deployments/dep1/diff"), server)

				_, err := deployment.Diff([]byte(""), false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Fetching diff result: Director responded with non-successful status code '400'"))
			})

			It("returns error if task result cannot be unmarshalled", func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/deployments/dep1/diff"),
						ghttp.RespondWith(http.StatusOK, ""),
					),
				)
				_, err := deployment.Diff([]byte(""), false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Fetching diff result: Unmarshaling Director response"))
			})

		})
	})
})
