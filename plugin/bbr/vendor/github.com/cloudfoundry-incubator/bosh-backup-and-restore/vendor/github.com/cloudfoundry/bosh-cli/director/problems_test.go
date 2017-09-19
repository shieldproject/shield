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
		director   Director
		deployment Deployment
		server     *ghttp.Server
	)

	BeforeEach(func() {
		director, server = BuildServer()

		var err error

		deployment, err = director.FindDeployment("dep1")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("ScanForProblems", func() {
		It("returns problems", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/deployments/dep1/scans"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"application/json"},
					}),
				),
				"",
				server,
			)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments/dep1/problems"),
					ghttp.RespondWith(http.StatusOK, `[
	{
		"id": 4,
		"type": "unresponsive_agent",
		"description": "desc1",
		"resolutions": [
			{"name": "Skip for now", "plan": "ignore"},
			{"name": "Reboot VM", "plan": "reboot_vm"}
		]
	},
	{
		"id": 5,
		"type": "unresponsive_agent",
		"description": "desc2",
		"resolutions": [
			{"name": "Skip for now", "plan": "ignore"}
		]
	}
]`),
				),
			)

			problems, err := deployment.ScanForProblems()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(problems)).To(Equal(2))

			problem0 := problems[0]

			Expect(problem0.ID).To(Equal(4))
			Expect(problem0.Type).To(Equal("unresponsive_agent"))
			Expect(problem0.Description).To(Equal("desc1"))
			problem0Resolutions := problem0.Resolutions
			Expect(len(problem0Resolutions)).To(Equal(2))
			Expect(*problem0Resolutions[0].Name).To(Equal("Skip for now"))
			Expect(problem0Resolutions[0].Plan).To(Equal("ignore"))
			Expect(*problem0Resolutions[1].Name).To(Equal("Reboot VM"))
			Expect(problem0Resolutions[1].Plan).To(Equal("reboot_vm"))

			problem1 := problems[1]
			Expect(problem1.ID).To(Equal(5))
			Expect(problem1.Type).To(Equal("unresponsive_agent"))
			Expect(problem1.Description).To(Equal("desc2"))
			problem1Resolutions := problem1.Resolutions
			Expect(len(problem1Resolutions)).To(Equal(1))
			Expect(*problem1Resolutions[0].Name).To(Equal("Skip for now"))
			Expect(problem1Resolutions[0].Plan).To(Equal("ignore"))
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("POST", "/deployments/dep1/scans"), server)

			_, err := deployment.ScanForProblems()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Performing a scan on deployment 'dep1': Director responded with non-successful status code"))
		})

		It("returns error if listing problems response is non-200", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/deployments/dep1/scans"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"application/json"},
					}),
				),
				"",
				server,
			)

			AppendBadRequest(ghttp.VerifyRequest("GET", "/deployments/dep1/problems"), server)

			_, err := deployment.ScanForProblems()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Listing problems for deployment 'dep1': Director responded with non-successful status code"))
		})

		It("returns error if response cannot be unmarshalled", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/deployments/dep1/scans"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"application/json"},
					}),
				),
				"",
				server,
			)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments/dep1/problems"),
					ghttp.RespondWith(http.StatusOK, ``),
				),
			)

			_, err := deployment.ScanForProblems()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Listing problems for deployment 'dep1': Unmarshaling Director response"))
		})
	})

	Describe("ProblemResolutionDefault", func() {
		It("provides default resolution", func() {
			Expect(ProblemResolutionDefault).To(Equal(ProblemResolution{
				Name: nil,
			}))
		})
	})

	Describe("ResolveProblems", func() {
		It("resolves problems with provided answers", func() {
			ConfigureTaskResult(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/deployments/dep1/problems"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyHeader(http.Header{
						"Content-Type": []string{"application/json"},
					}),
					ghttp.VerifyBody([]byte(`{"resolutions":{"4":"res1-name","5":"res2-name"}}`)),
				),
				"",
				server,
			)

			resolutionName1 := "res1-name"
			resolutionName2 := "res2-name"
			answers := []ProblemAnswer{
				{ProblemID: 4, Resolution: ProblemResolution{Name: &resolutionName1}},
				{ProblemID: 5, Resolution: ProblemResolution{Name: &resolutionName2}},
			}

			err := deployment.ResolveProblems(answers)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("PUT", "/deployments/dep1/problems"), server)

			err := deployment.ResolveProblems(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Resolving problems for deployment 'dep1': Director responded with non-successful status code"))
		})
	})
})
