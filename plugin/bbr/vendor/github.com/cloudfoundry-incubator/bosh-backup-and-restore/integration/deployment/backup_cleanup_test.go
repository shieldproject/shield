package deployment

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf-experimental/cf-webmock/mockbosh"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/testcluster"
)

var _ = Describe("Cleanup", func() {
	var cleanupWorkspace string
	var director *mockhttp.Server

	Context("when deployment has a single instance", func() {
		var session *gexec.Session
		var instance1 *testcluster.Instance
		var deploymentName string
		var err error

		BeforeEach(func() {
			cleanupWorkspace, err = ioutil.TempDir(".", "cleanup-workspace-")

			instance1 = testcluster.NewInstance()

			deploymentName = "my-new-deployment"
			director = mockbosh.NewTLS()
			director.ExpectedBasicAuth("admin", "admin")
			director.VerifyAndMock(AppendBuilders(
				InfoWithBasicAuth(),
				VmsForDeployment(deploymentName, []mockbosh.VMsOutput{
					{
						IPs:     []string{"10.0.0.1"},
						JobName: "redis-dedicated-node",
						JobID:   "fake-uuid",
					}}),
				SetupSSH(deploymentName, "redis-dedicated-node", "fake-uuid", 0, instance1),
				CleanupSSH(deploymentName, "redis-dedicated-node"),
			)...)

			instance1.CreateScript("/var/vcap/jobs/redis/bin/bbr/backup", ``)
			instance1.CreateDir("/var/vcap/store/bbr-backup")
		})

		JustBeforeEach(func() {
			session = binary.Run(
				cleanupWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
				"deployment",
				"--ca-cert", sslCertPath,
				"--username", "admin",
				"--debug",
				"--target", director.URL,
				"--deployment", deploymentName,
				"backup-cleanup",
			)
		})

		AfterEach(func() {
			instance1.DieInBackground()
			Expect(os.RemoveAll(cleanupWorkspace)).To(Succeed())
		})

		It("runs the restore script successfully and cleans up", func() {
			By("succeeding", func() {
				Eventually(session.ExitCode()).Should(Equal(0))
			})

			By("cleaning up the archive file on the remote", func() {
				Expect(instance1.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
			})
		})
	})
})
