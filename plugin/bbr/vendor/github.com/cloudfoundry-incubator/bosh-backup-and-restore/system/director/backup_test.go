package director

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/system"
)

var _ = Describe("Backup", func() {
	AfterEach(func() {
		By("removing the backup")
		Eventually(JumpboxInstance.RunCommandAs("vcap",
			fmt.Sprintf(
				`sudo rm -rf %s/%s`,
				workspaceDir,
				MustHaveEnv("HOST_TO_BACKUP"),
			))).Should(gexec.Exit(0))
	})

	It("backs up the director", func() {
		directorIP := MustHaveEnv("HOST_TO_BACKUP")

		By("running the backup command")
		backupCommand := JumpboxInstance.RunCommandAs("vcap",
			fmt.Sprintf(
				`cd %s; ./bbr director --username vcap --private-key-path ./key.pem --host %s backup`,
				workspaceDir,
				directorIP),
		)
		Eventually(backupCommand).Should(gexec.Exit(0))

		JumpboxInstance.AssertFilesExist([]string{
			fmt.Sprintf("%s/%s/bosh-0-test-backup-and-restore.tar", workspaceDir, BackupDirWithTimestamp(directorIP)),
			fmt.Sprintf("%s/%s/bosh-0-remarkable-backup-and-restore.tar", workspaceDir, BackupDirWithTimestamp(directorIP)),
			fmt.Sprintf("%s/%s/bosh-0-amazing-backup-and-restore.tar", workspaceDir, BackupDirWithTimestamp(directorIP)),
		})
	})
})
