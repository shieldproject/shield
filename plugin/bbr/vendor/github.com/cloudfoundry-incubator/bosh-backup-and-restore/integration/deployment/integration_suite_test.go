package deployment

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/integration"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/testcluster"
)

func TestDeploymentIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deployment Integration Suite")
}

//Default cert for golang ssh
var sslCertPath = "../../../fixtures/test.crt"

var binary integration.Binary

var _ = BeforeSuite(func() {
	commandPath, err := gexec.Build("github.com/cloudfoundry-incubator/bosh-backup-and-restore/cmd/bbr")
	Expect(err).NotTo(HaveOccurred())
	binary = integration.NewBinary(commandPath)
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
	testcluster.WaitForContainersToDie()
})
