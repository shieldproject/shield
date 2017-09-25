package integration_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/cloudfoundry/bosh-s3cli/integration"

	"testing"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var s3CLIPath string
var largeContent string

var _ = BeforeSuite(func() {
	// Running the IAM tests within an AWS Lambda environment
	// require a pre-compiled binary
	s3CLIPath = os.Getenv("S3_CLI_PATH")
	largeContent = integration.GenerateRandomString(1024 * 1024 * 6)

	if len(s3CLIPath) == 0 {
		var err error
		s3CLIPath, err = gexec.Build("github.com/cloudfoundry/bosh-s3cli")
		Expect(err).ShouldNot(HaveOccurred())
	}
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
