package director

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/testcluster"
)

var _ = Describe("Cleanup", func() {
	var cleanupWorkspace string

	Context("when director has a backup artifact", func() {
		var session *gexec.Session
		var directorInstance *testcluster.Instance
		var directorAddress string
		var err error

		BeforeEach(func() {
			cleanupWorkspace, err = ioutil.TempDir(".", "cleanup-workspace-")

			directorInstance = testcluster.NewInstance()
			directorInstance.CreateUser("foobar", readFile(pathToPublicKeyFile))
			directorAddress = directorInstance.Address()

			directorInstance.CreateScript("/var/vcap/jobs/redis/bin/bbr/backup", ``)
			directorInstance.CreateDir("/var/vcap/store/bbr-backup")
		})

		JustBeforeEach(func() {
			session = binary.Run(
				cleanupWorkspace,
				[]string{"BOSH_CLIENT_SECRET=admin"},
				"director",
				"--host", directorAddress,
				"--username", "foobar",
				"--private-key-path", pathToPrivateKeyFile,
				"--debug",
				"backup-cleanup",
			)
		})

		AfterEach(func() {
			directorInstance.DieInBackground()
			Expect(os.RemoveAll(cleanupWorkspace)).To(Succeed())
		})

		It("runs the restore script successfully and cleans up", func() {
			By("succeeding", func() {
				Eventually(session.ExitCode()).Should(Equal(0))
			})

			By("cleaning up the archive file on the remote", func() {
				Expect(directorInstance.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
			})
		})
	})
})
