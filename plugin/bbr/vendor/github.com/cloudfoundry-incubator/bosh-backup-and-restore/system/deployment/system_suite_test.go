package deployment

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/system"

	"sync"
	"testing"
)

func TestSystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "System Suite")
}

var (
	commandPath string
	err         error
)

var fixturesPath = "../../fixtures/redis-backup/"

var _ = BeforeEach(func() {
	SetDefaultEventuallyTimeout(4 * time.Minute)
	var wg sync.WaitGroup

	wg.Add(6)
	go func() {
		defer GinkgoRecover()
		defer wg.Done()
		By("deploying the Redis test release")
		RedisDeployment.Deploy()
	}()

	go func() {
		defer GinkgoRecover()
		defer wg.Done()
		By("deploying the Redis with metadata")
		RedisWithMetadataDeployment.Deploy()
	}()

	go func() {
		defer GinkgoRecover()
		defer wg.Done()
		By("deploying the Redis with missing backup script")
		RedisWithMissingScriptDeployment.Deploy()
	}()

	go func() {
		defer GinkgoRecover()
		defer wg.Done()
		By("deploying the jump box")
		JumpboxDeployment.Deploy()
	}()

	go func() {
		defer GinkgoRecover()
		defer wg.Done()
		By("deploying the other Redis test release")
		AnotherRedisDeployment.Deploy()
	}()

	go func() {
		defer GinkgoRecover()
		defer wg.Done()
		By("deploying the slow backup Redis test release")
		RedisSlowBackupDeployment.Deploy()
	}()

	wg.Wait()

	By("building bbr")
	commandPath, err = gexec.BuildWithEnvironment("github.com/cloudfoundry-incubator/bosh-backup-and-restore/cmd/bbr", []string{"GOOS=linux", "GOARCH=amd64"})
	Expect(err).NotTo(HaveOccurred())

	By("setting up the jump box")
	Eventually(JumpboxInstance.RunCommand(
		fmt.Sprintf("sudo mkdir %s && sudo chown vcap:vcap %s && sudo chmod 0777 %s", workspaceDir, workspaceDir, workspaceDir))).Should(gexec.Exit(0))

	JumpboxInstance.Copy(commandPath, workspaceDir)
	JumpboxInstance.Copy(MustHaveEnv("BOSH_CERT_PATH"), workspaceDir+"/bosh.crt")
})

var _ = AfterEach(func() {
	var wg sync.WaitGroup

	wg.Add(6)

	go func() {
		defer GinkgoRecover()
		defer wg.Done()
		By("tearing down the redis release")
		RedisDeployment.Delete()
	}()

	go func() {
		defer GinkgoRecover()
		defer wg.Done()
		By("tearing down the other redis release")
		RedisWithMetadataDeployment.Delete()
	}()

	go func() {
		defer GinkgoRecover()
		defer wg.Done()
		By("tearing down the other redis release")
		RedisWithMissingScriptDeployment.Delete()
	}()

	go func() {
		defer GinkgoRecover()
		defer wg.Done()
		By("tearing down the redis with metadata")
		AnotherRedisDeployment.Delete()
	}()

	go func() {
		defer GinkgoRecover()
		defer wg.Done()
		By("tearing down the jump box")
		JumpboxDeployment.Delete()
	}()

	go func() {
		defer GinkgoRecover()
		defer wg.Done()
		By("tearing down the slow backup Redis test release")
		RedisSlowBackupDeployment.Delete()
	}()

	wg.Wait()
})

func runOnInstances(instanceCollection map[string][]string, f func(string, string)) {
	for instanceGroup, instances := range instanceCollection {
		for _, instanceIndex := range instances {
			f(instanceGroup, instanceIndex)
		}
	}
}
