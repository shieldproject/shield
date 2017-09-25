package integration_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"

	. "github.com/cloudfoundry/bosh-agent/integration"
)

var (
	testEnvironment *TestEnvironment
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)

	BeforeSuite(func() {
		logger := boshlog.NewLogger(boshlog.LevelDebug)
		cmdRunner := boshsys.NewExecCmdRunner(logger)
		testEnvironment = NewTestEnvironment(cmdRunner)

		// Required for reverse-compatibility with older bosh-lite
		// (remove once a new warden stemcell is built).
		err := testEnvironment.ConfigureAgentForGenericInfrastructure()
		Expect(err).ToNot(HaveOccurred())
	})

	RunSpecs(t, "Integration Suite")
}
