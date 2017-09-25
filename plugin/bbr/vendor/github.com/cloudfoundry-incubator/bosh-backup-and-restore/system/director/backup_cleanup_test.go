package director

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/system"
)

var _ = Describe("Director backup cleanup", func() {
	var directorIP = MustHaveEnv("HOST_TO_BACKUP")

	BeforeEach(func() {
		By("starting a backup and aborting mid-way")
		backupSession := JumpboxInstance.RunCommandAs("vcap",
			fmt.Sprintf(
				`cd %s; \
				 ./bbr director \
				   --username vcap \
				   --private-key-path ./key.pem \
				   --host %s backup`,
				workspaceDir,
				directorIP),
		)

		Eventually(backupSession.Out).Should(gbytes.Say("Backing up test-backup-and-restore on bosh"))
		Eventually(JumpboxInstance.RunCommandAs("vcap", "killall bbr")).Should(gexec.Exit(0))
	})

	AfterEach(func() {
		By("cleaning up the director")
		Eventually(JumpboxInstance.RunCommandAs("vcap",
			fmt.Sprintf(
				`cd %s; \
				ssh %s vcap@%s \
					-i key.pem \
					"sudo rm -rf /var/vcap/store/bbr-backup"`,
				workspaceDir,
				skipSSHFingerprintCheckOpts,
				directorIP,
			))).Should(gexec.Exit(0))

		By("removing the backup")
		Eventually(JumpboxInstance.RunCommandAs("vcap",
			fmt.Sprintf(
				`sudo rm -rf %s/%s*`,
				workspaceDir,
				directorIP,
			))).Should(gexec.Exit(0))
	})

	Context("When we run cleanup", func() {
		It("succeeds", func() {
			By("cleaning up the director artifact", func() {
				cleanupCommand := JumpboxInstance.RunCommandAs("vcap",
					fmt.Sprintf(
						`cd %s; \
					 ./bbr director \
						 --username vcap \
						 --debug \
						 --private-key-path ./key.pem \
						 --host %s backup-cleanup`,
						workspaceDir,
						directorIP),
				)

				Eventually(cleanupCommand).Should(gexec.Exit(0))
				Eventually(cleanupCommand).Should(gbytes.Say("'%s' cleaned up", directorIP))

				Eventually(JumpboxInstance.RunCommandAs("vcap",
					fmt.Sprintf(
						`cd %s; \
						ssh %s vcap@%s \
						-i key.pem \
						"ls -l /var/vcap/store/bbr-backup"`,
						workspaceDir,
						skipSSHFingerprintCheckOpts,
						directorIP,
					))).Should(gbytes.Say("ls: cannot access /var/vcap/store/bbr-backup: No such file or directory"))
			})

			By("allowing subsequent backups to complete successfully", func() {
				backupCommand := JumpboxInstance.RunCommandAs("vcap",
					fmt.Sprintf(
						`cd %s; \
					 ./bbr director \
						 --debug \
						 --username vcap \
						 --private-key-path ./key.pem \
						 --host %s backup`,
						workspaceDir,
						directorIP),
				)

				Eventually(backupCommand).Should(gexec.Exit(0))
			})
		})
	})

	Context("when we don't run a cleanup", func() {
		It("is in a state where subsequent backups fail", func() {
			backupCommand := JumpboxInstance.RunCommandAs("vcap",
				fmt.Sprintf(
					`cd %s; \
					 ./bbr director \
						 --username vcap \
						 --private-key-path ./key.pem \
						 --host %s backup`,
					workspaceDir,
					directorIP),
			)

			Eventually(backupCommand).Should(gexec.Exit(1))
			Expect(backupCommand.Out.Contents()).To(ContainSubstring("Directory /var/vcap/store/bbr-backup already exists"))
		})
	})
})
