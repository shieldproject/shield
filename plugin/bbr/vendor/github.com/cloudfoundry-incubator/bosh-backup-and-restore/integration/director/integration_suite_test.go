package director

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"

	"io/ioutil"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/integration"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/testcluster"
)

func TestDirectorIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Director Integration Suite")
}

var pathToPrivateKeyFile = "../../../fixtures/test_rsa"
var pathToPublicKeyFile = "../../fixtures/test_rsa.pub"

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

func readFile(fileName string) string {
	contents, err := ioutil.ReadFile(fileName)
	Expect(err).NotTo(HaveOccurred())
	return string(contents)
}
