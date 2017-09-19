package director

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/testcluster"

	"github.com/onsi/gomega/gexec"

	"os/exec"

	"path"

	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Restore", func() {
	var restoreWorkspace string
	var session *gexec.Session
	var directorAddress, directorIP string
	var artifactName string

	BeforeEach(func() {
		var err error
		restoreWorkspace, err = ioutil.TempDir(".", "restore-workspace-")
		Expect(err).NotTo(HaveOccurred())
		artifactName = "director-backup-integration"

		command := exec.Command("cp", "-r", "../../fixtures/director-backup-integration", path.Join(restoreWorkspace, artifactName))
		cpFiles, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		Eventually(cpFiles).Should(gexec.Exit())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(restoreWorkspace)).To(Succeed())
	})

	JustBeforeEach(func() {
		session = binary.Run(
			restoreWorkspace,
			[]string{"BOSH_CLIENT_SECRET=admin"},
			"director",
			"--host", directorAddress,
			"--username", "foobar",
			"--private-key-path", pathToPrivateKeyFile,
			"--debug",
			"restore",
			"--artifact-path", artifactName,
		)
	})

	Context("When there is a director instance", func() {
		var directorInstance *testcluster.Instance

		BeforeEach(func() {
			directorInstance = testcluster.NewInstance()
			directorInstance.CreateUser("foobar", readFile(pathToPublicKeyFile))
			directorAddress = directorInstance.Address()
			directorIP = directorInstance.IP()
		})

		AfterEach(func() {
			directorInstance.DieInBackground()
		})

		Context("and there are restore scripts", func() {
			BeforeEach(func() {
				directorInstance.CreateFiles("/var/vcap/jobs/bosh/bin/bbr/restore")
				directorInstance.CreateFiles("/var/vcap/jobs/bosh/bin/bbr/post-restore-unlock")
			})

			Context("and the restore script succeeds", func() {
				BeforeEach(func() {
					directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/restore", `#!/usr/bin/env sh
set -u

mkdir -p /var/vcap/store/bosh/
cat $BBR_ARTIFACT_DIRECTORY/backup > /var/vcap/store/bosh/restored_file
`)
				})

				It("successfully restores to the director", func() {
					By("exiting zero", func() {
						Expect(session.ExitCode()).To(BeZero())
					})

					By("running the restore script successfully", func() {
						Expect(directorInstance.FileExists("/var/vcap/store/bosh/restored_file")).To(BeTrue())
						Expect(directorInstance.GetFileContents("/var/vcap/store/bosh/restored_file")).To(ContainSubstring(`this is a backup`))
					})

					By("running the restore script successfully", func() {
						Expect(directorInstance.FileExists("/var/vcap/store/bosh/restored_file")).To(BeTrue())
						Expect(directorInstance.GetFileContents("/var/vcap/store/bosh/restored_file")).To(ContainSubstring(`this is a backup`))
					})

					By("cleaning up backup artifacts from the remote", func() {
						Expect(directorInstance.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
					})
				})

				Context("there is a post-restore-unlock script which succeeds", func() {
					BeforeEach(func() {
						directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/post-restore-unlock", `#!/usr/bin/env sh
touch /tmp/post-restore-unlock-script-was-run
`)
					})
					It("runs the post-restore-unlock script successfully", func() {
						By("exiting zero", func() {
							Expect(session.ExitCode()).To(BeZero())
						})

						By("running the restore script successfully", func() {
							Expect(directorInstance.FileExists("/var/vcap/store/bosh/restored_file")).To(BeTrue())
							Expect(directorInstance.GetFileContents("/var/vcap/store/bosh/restored_file")).To(ContainSubstring(`this is a backup`))
						})

						By("running the post-restore-unlock script successfully", func() {
							Expect(directorInstance.FileExists("/tmp/post-restore-unlock-script-was-run")).To(BeTrue())
						})

						By("cleaning up backup artifacts from the remote", func() {
							Expect(directorInstance.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
						})
					})
				})

				Context("and there is a post-restore-unlock script which fails", func() {
					BeforeEach(func() {
						directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/post-restore-unlock", `#!/usr/bin/env sh
echo "post-restore-unlock errored!"
exit 1
`)
					})
					It("fails the command", func() {
						By("exiting non-zero", func() {
							Expect(session.ExitCode()).NotTo(BeZero())
						})

						By("running the restore script successfully", func() {
							Expect(directorInstance.FileExists("/var/vcap/store/bosh/restored_file")).To(BeTrue())
							Expect(directorInstance.GetFileContents("/var/vcap/store/bosh/restored_file")).To(ContainSubstring(`this is a backup`))
						})

						By("cleaning up backup artifacts from the remote", func() {
							Expect(directorInstance.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
						})

						By("error is displayed", func() {
							Expect(session.Out.Contents()).To(ContainSubstring("post-restore-unlock errored"))
						})
					})

				})
			})

			Context("but the restore script fails", func() {
				BeforeEach(func() {
					directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/restore", "echo 'NOPE!'; exit 1")
				})

				It("fails to restore the director", func() {
					By("returning exit code 1", func() {
						Expect(session.ExitCode()).To(Equal(1))
						Expect(session.Out.Contents()).To(ContainSubstring("NOPE!"))
					})
				})

				Context("there is a post-restore-unlock script which succeeds", func() {
					BeforeEach(func() {
						directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/post-restore-unlock", `#!/usr/bin/env sh
touch /tmp/post-restore-unlock-script-was-run
`)
					})
					It("runs the post-restore-unlock script successfully", func() {
						Expect(directorInstance.FileExists("/tmp/post-restore-unlock-script-was-run")).To(BeTrue())
					})
				})

				Context("and there is a post-restore-unlock script which fails", func() {
					BeforeEach(func() {
						directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/post-restore-unlock", `#!/usr/bin/env sh
echo "post-restore-unlock errored!"
exit 1
`)
					})
					It("fails the command", func() {
						By("exiting non-zero", func() {
							Expect(session.ExitCode()).NotTo(BeZero())
						})

						By("error is displayed", func() {
							Expect(session.Out.Contents()).To(ContainSubstring("post-restore-unlock errored"))
							Expect(session.Out.Contents()).To(ContainSubstring("NOPE!"))
						})
					})

				})
			})

			Context("but the artifact directory already exists", func() {
				BeforeEach(func() {
					directorInstance.CreateDir("/var/vcap/store/bbr-backup")
				})

				It("fails to restore the director", func() {
					By("exiting non-zero", func() {
						Expect(session.ExitCode()).NotTo(BeZero())
					})

					By("printing a log message saying the director instance cannot be backed up", func() {
						Expect(string(session.Err.Contents())).To(ContainSubstring("Directory /var/vcap/store/bbr-backup already exists on instance bosh/0"))
					})

					By("not deleting the existing artifact directory", func() {
						Expect(directorInstance.FileExists("/var/vcap/store/bbr-backup")).To(BeTrue())
					})
				})
			})
		})

		Context("but there are no restore scripts", func() {
			BeforeEach(func() {
				directorInstance.CreateFiles("/var/vcap/jobs/bosh/bin/bbr/backup")
				directorInstance.CreateFiles("/var/vcap/jobs/bosh/bin/bbr/not-a-restore-script")
			})

			It("fails to restore the director", func() {
				By("returning exit code 1", func() {
					Expect(session.ExitCode()).To(Equal(1))
				})

				By("printing an error", func() {
					Expect(string(session.Err.Contents())).To(ContainSubstring(fmt.Sprintf("Deployment '%s' has no restore scripts", directorIP)))
				})

				By("saving the stack trace into a file", func() {
					files, err := filepath.Glob(filepath.Join(restoreWorkspace, "bbr-*.err.log"))
					Expect(err).NotTo(HaveOccurred())
					logFilePath := files[0]
					_, err = os.Stat(logFilePath)
					Expect(os.IsNotExist(err)).To(BeFalse())
					stackTrace, err := ioutil.ReadFile(logFilePath)
					Expect(err).ToNot(HaveOccurred())
					Expect(gbytes.BufferWithBytes(stackTrace)).To(gbytes.Say("main.go"))
				})
			})
		})
	})

	Context("When the director does not resolve", func() {
		BeforeEach(func() {
			directorAddress = "does-not-resolve"
		})

		It("fails to restore the director", func() {
			By("returning exit code 1", func() {
				Expect(session.ExitCode()).To(Equal(1))
			})

			By("printing an error", func() {
				Expect(string(session.Err.Contents())).To(ContainSubstring("no such host"))
			})
		})
	})
})
