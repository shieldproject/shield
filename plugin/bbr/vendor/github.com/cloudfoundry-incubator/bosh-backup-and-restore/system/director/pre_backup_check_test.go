package director

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/system"
)

var _ = Describe("PreBackupCheck", func() {

	It("checks if the director is backupable", func() {
		By("running the pre-backup-check command")
		preBackupCheckCommand := JumpboxInstance.RunCommandAs("vcap",
			fmt.Sprintf(
				`cd %s; ./bbr director --username vcap --private-key-path ./key.pem --host %s pre-backup-check`,
				workspaceDir,
				MustHaveEnv("HOST_TO_BACKUP")),
		)
		Eventually(preBackupCheckCommand).Should(gexec.Exit(0))
		Expect(preBackupCheckCommand.Out.Contents()).To(ContainSubstring("Director can be backed up"))
	})
})
