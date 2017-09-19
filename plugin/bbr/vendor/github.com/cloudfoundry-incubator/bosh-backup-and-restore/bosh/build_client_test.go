package bosh

import (
	"log"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf-experimental/cf-webmock/mockbosh"
	"github.com/pivotal-cf-experimental/cf-webmock/mockuaa"
)

var _ = Describe("BuildClient", func() {
	var logger = boshlog.New(boshlog.LevelDebug, log.New(gbytes.NewBuffer(), "[bosh-package] ", log.Lshortfile), log.New(gbytes.NewBuffer(), "[bosh-package] ", log.Lshortfile))

	var director *mockhttp.Server
	var deploymentName = "my-little-deployment"
	var sslCertPath = "../fixtures/test.crt"

	BeforeEach(func() {
		director = mockbosh.NewTLS()
	})
	AfterEach(func() {
		director.VerifyMocks()
	})

	Context("With Basic Auth", func() {
		It("build the client which makes basic auth against director", func() {
			username := "foo"
			password := "bar"

			director.ExpectedBasicAuth(username, password)
			director.VerifyAndMock(
				mockbosh.Info().WithAuthTypeBasic(),
				mockbosh.Manifest(deploymentName).RespondsWith([]byte("manifest contents")),
			)

			client, err := BuildClient(director.URL, username, password, sslCertPath, logger)

			Expect(err).NotTo(HaveOccurred())
			manifest, err := client.GetManifest(deploymentName)
			Expect(err).NotTo(HaveOccurred())
			Expect(manifest).To(Equal("manifest contents"))
		})
	})

	Context("With UAA", func() {
		var uaaServer *mockuaa.ClientCredentialsServer

		It("build the client which makes basic auth against director", func() {
			username := "foo"
			password := "bar"
			uaaToken := "baz"

			uaaServer = mockuaa.NewClientCredentialsServerTLS(username, password, uaaToken)

			director.ExpectedAuthorizationHeader("bearer " + uaaToken)
			director.VerifyAndMock(
				mockbosh.Info().WithAuthTypeUAA(uaaServer.URL),
				mockbosh.Manifest(deploymentName).RespondsWith([]byte("manifest contents")),
			)

			client, err := BuildClient(director.URL, username, password, sslCertPath, logger)

			Expect(err).NotTo(HaveOccurred())
			manifest, err := client.GetManifest(deploymentName)
			Expect(err).NotTo(HaveOccurred())
			Expect(manifest).To(Equal("manifest contents"))
		})

		It("fails if uaa url is not valid", func() {
			username := "no-relevant"
			password := "no-relevant"

			director.VerifyAndMock(
				mockbosh.Info().WithAuthTypeUAA(""),
			)
			_, err := BuildClient(director.URL, username, password, sslCertPath, logger)

			Expect(err).To(MatchError(ContainSubstring("invalid UAA URL")))

		})
	})

	It("fails if CA-Cert cant be read", func() {
		username := "no-relevant"
		password := "no-relevant"
		caCertPath := "/invalid/path"
		basicAuthDirectorUrl := director.URL

		_, err := BuildClient(basicAuthDirectorUrl, username, password, caCertPath, logger)
		Expect(err).To(MatchError(ContainSubstring("CA-CERT can't be read")))
	})

	It("fails if invalid bosh url", func() {
		username := "no-relevant"
		password := "no-relevant"
		caCertPath := ""
		basicAuthDirectorUrl := ""

		_, err := BuildClient(basicAuthDirectorUrl, username, password, caCertPath, logger)
		Expect(err).To(MatchError(ContainSubstring("invalid bosh URL")))
	})

	It("fails if info cant be retrieved", func() {
		username := "no-relevant"
		password := "no-relevant"

		director.VerifyAndMock(
			mockbosh.Info().Fails("fooo!"),
		)

		_, err := BuildClient(director.URL, username, password, sslCertPath, logger)
		Expect(err).To(MatchError(ContainSubstring("bosh director unreachable or unhealthy")))
	})

})
