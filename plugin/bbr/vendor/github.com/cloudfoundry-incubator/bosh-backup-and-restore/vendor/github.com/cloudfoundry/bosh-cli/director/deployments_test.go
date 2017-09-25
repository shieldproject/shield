package director_test

import (
	"net/http"

	semver "github.com/cppforlife/go-semi-semantic/version"
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

	Describe("Deployments", func() {
		It("returns deployments", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[
  {
    "name": "dep1-name",
    "stemcells": [
      {"name": "stem1-name", "version": "3147"}
    ],
    "releases": [
      {"name": "rel1-name", "version": "3"},
      {"name": "rel2-name", "version": "243"}
    ],
    "teams": ["team1", "team2"],
    "cloud_config": "none"
  }
]`),
				),
			)

			deps, err := director.Deployments()
			Expect(err).ToNot(HaveOccurred())
			Expect(deps).To(HaveLen(1))

			Expect(deps[0].Name()).To(Equal("dep1-name"))

			rels, err := deps[0].Releases()
			Expect(err).ToNot(HaveOccurred())
			Expect(rels[0].Name()).To(Equal("rel1-name"))
			Expect(rels[0].Version()).To(Equal(semver.MustNewVersionFromString("3")))
			Expect(rels[1].Name()).To(Equal("rel2-name"))
			Expect(rels[1].Version()).To(Equal(semver.MustNewVersionFromString("243")))

			stems, err := deps[0].Stemcells()
			Expect(err).ToNot(HaveOccurred())
			Expect(stems[0].Name()).To(Equal("stem1-name"))
			Expect(stems[0].Version()).To(Equal(semver.MustNewVersionFromString("3147")))

			teams, err := deps[0].Teams()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(teams)).To(Equal(2))
			Expect(teams[0]).To(Equal("team1"))
			Expect(teams[1]).To(Equal("team2"))

			Expect(deps[0].CloudConfig()).To(Equal("none"))
		})

		It("returns empty deployment with no teams", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[
  {
    "name": "dep-no-team-name",
    "stemcells": [],
    "releases": [],
    "teams": [],
    "cloud_config": "none"
  }
]`),
				),
			)

			deps, err := director.Deployments()
			Expect(err).ToNot(HaveOccurred())
			Expect(deps).To(HaveLen(1))

			Expect(deps[0].Name()).To(Equal("dep-no-team-name"))

			rels, err := deps[0].Releases()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(rels)).To(Equal(0))

			stems, err := deps[0].Stemcells()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(stems)).To(Equal(0))

			teams, err := deps[0].Teams()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(teams)).To(Equal(0))

			Expect(deps[0].CloudConfig()).To(Equal("none"))
		})

		It("returns an error for invalid release versions", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[{"releases":[{"name":"name","version":"-"}]}]`),
				),
			)

			deps, err := director.Deployments()
			Expect(err).ToNot(HaveOccurred())

			_, err = deps[0].Releases()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Parsing version for release"))
		})

		It("returns an error for invalid stemcell versions", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[{"stemcells":[{"name":"name","version":"-"}]}]`),
				),
			)

			deps, err := director.Deployments()
			Expect(err).ToNot(HaveOccurred())

			_, err = deps[0].Stemcells()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Parsing version for stemcell"))
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("GET", "/deployments"), server)

			_, err := director.Deployments()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Finding deployments: Director responded with non-successful status code"))
		})

		It("returns error if response cannot be unmarshalled", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/deployments"),
					ghttp.RespondWith(http.StatusOK, ``),
				),
			)

			_, err := director.Deployments()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Finding deployments: Unmarshaling Director response"))
		})
	})

	Describe("FindDeployment", func() {
		It("returns an error if name is empty", func() {
			_, err := director.FindDeployment("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Expected non-empty deployment name"))
		})
	})
})
