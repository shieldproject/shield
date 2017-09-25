package instance_test

import (
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"

	"log"

	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh/fakes"
)

var _ = Describe("Job", func() {
	var job instance.Job
	var jobScripts instance.BackupAndRestoreScripts
	var metadata instance.Metadata
	var sshConnection *fakes.FakeSSHConnection
	var logger boshlog.Logger

	BeforeEach(func() {
		jobScripts = instance.BackupAndRestoreScripts{
			"/var/vcap/jobs/foo/bin/bbr/restore",
			"/var/vcap/jobs/foo/bin/bbr/backup",
			"/var/vcap/jobs/foo/bin/bbr/pre-backup-lock",
			"/var/vcap/jobs/foo/bin/bbr/post-backup-unlock",
		}
		metadata = instance.Metadata{}
		sshConnection = new(fakes.FakeSSHConnection)

		combinedLog := log.New(GinkgoWriter, "[instance-test] ", log.Lshortfile)
		logger = boshlog.New(boshlog.LevelDebug, combinedLog, combinedLog)

	})

	JustBeforeEach(func() {
		job = instance.NewJob(sshConnection, "", logger, jobScripts, metadata)
	})

	Describe("BackupArtifactDirectory", func() {
		It("calculates the artifact directory based on the name", func() {
			Expect(job.BackupArtifactDirectory()).To(Equal("/var/vcap/store/bbr-backup/foo"))
		})

		Context("when an artifact name is provided", func() {
			var jobWithName instance.Job

			JustBeforeEach(func() {
				jobWithName = instance.NewJob(sshConnection, "", logger,
					jobScripts, instance.Metadata{
						BackupName: "a-bosh-backup",
					})
			})

			It("calculates the artifact directory based on the artifact name", func() {
				Expect(jobWithName.BackupArtifactDirectory()).To(Equal("/var/vcap/store/bbr-backup/a-bosh-backup"))
			})
		})
	})

	Describe("RestoreArtifactDirectory", func() {
		It("calculates the artifact directory based on the name", func() {
			Expect(job.BackupArtifactDirectory()).To(Equal("/var/vcap/store/bbr-backup/foo"))
		})

		Context("when an artifact name is provided", func() {
			var jobWithName instance.Job

			JustBeforeEach(func() {
				jobWithName = instance.NewJob(sshConnection, "", logger,
					jobScripts, instance.Metadata{
						RestoreName: "a-bosh-backup",
					})
			})

			It("calculates the artifact directory based on the artifact name", func() {
				Expect(jobWithName.RestoreArtifactDirectory()).To(Equal("/var/vcap/store/bbr-backup/a-bosh-backup"))
			})
		})
	})

	//TODO test Backup

	Describe("BackupArtifactName", func() {
		Context("the job has a custom backup artifact name", func() {
			BeforeEach(func() {
				metadata = instance.Metadata{
					BackupName: "fool",
				}
			})

			It("returns the job's custom backup artifact name", func() {
				Expect(job.BackupArtifactName()).To(Equal("fool"))
			})
		})

		Context("the job does not have a custom backup artifact name", func() {
			It("returns empty string", func() {
				Expect(job.BackupArtifactName()).To(Equal(""))
			})
		})
	})

	Describe("RestoreArtifactName", func() {
		Context("the job has a custom backup artifact name", func() {
			BeforeEach(func() {
				metadata = instance.Metadata{
					RestoreName: "bard",
				}
			})

			It("returns the job's custom backup artifact name", func() {
				Expect(job.RestoreArtifactName()).To(Equal("bard"))
			})
		})

		Context("the job does not have a custom backup artifact name", func() {
			It("returns empty string", func() {
				Expect(job.RestoreArtifactName()).To(Equal(""))
			})
		})
	})

	Describe("HasBackup", func() {
		It("returns true", func() {
			Expect(job.HasBackup()).To(BeTrue())
		})

		Context("no backup scripts exist", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/bbr/restore"}
			})
			It("returns false", func() {
				Expect(job.HasBackup()).To(BeFalse())
			})
		})
	})

	Describe("RestoreScript", func() {
		It("returns the post-backup script", func() {
			Expect(job.RestoreScript()).To(Equal(instance.Script("/var/vcap/jobs/foo/bin/bbr/restore")))
		})
		Context("no post-backup scripts exist", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/bbr/backup"}
			})
			It("returns nil", func() {
				Expect(job.RestoreScript()).To(BeEmpty())
			})
		})
	})

	Describe("HasRestore", func() {
		It("returns true", func() {
			Expect(job.HasRestore()).To(BeTrue())
		})

		Context("no post-backup scripts exist", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/bbr/backup"}
			})
			It("returns false", func() {
				Expect(job.HasRestore()).To(BeFalse())
			})
		})
	})

	Describe("HasNamedBackupArtifact", func() {
		It("returns false", func() {
			Expect(job.HasNamedBackupArtifact()).To(BeFalse())
		})

		Context("when the job has a named backup artifact", func() {
			BeforeEach(func() {
				metadata = instance.Metadata{
					BackupName: "foo",
				}
			})

			It("returns true", func() {
				Expect(job.HasNamedBackupArtifact()).To(BeTrue())
			})
		})

		Context("when the job has a named restore artifact", func() {
			BeforeEach(func() {
				metadata = instance.Metadata{
					RestoreName: "foo",
				}
			})

			It("returns false", func() {
				Expect(job.HasNamedBackupArtifact()).To(BeFalse())
			})
		})
	})

	Describe("HasNamedRestoreArtifact", func() {
		It("returns false", func() {
			Expect(job.HasNamedRestoreArtifact()).To(BeFalse())
		})

		Context("when the job has a named restore artifact", func() {
			BeforeEach(func() {
				metadata = instance.Metadata{
					RestoreName: "foo",
				}
			})

			It("returns true", func() {
				Expect(job.HasNamedRestoreArtifact()).To(BeTrue())
			})
		})

		Context("when the job has a named backup artifact", func() {
			BeforeEach(func() {
				metadata = instance.Metadata{
					BackupName: "foo",
				}
			})

			It("returns false", func() {
				Expect(job.HasNamedRestoreArtifact()).To(BeFalse())
			})
		})
	})

	Describe("Backup", func() {
		var backupError error
		JustBeforeEach(func() {
			backupError = job.Backup()
		})
		Context("job has no backup script", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/foo/bin/bbr/pre-backup-lock",
				}
			})
			It("should not run anything on the ssh connection", func() {
				Expect(sshConnection.RunCallCount()).To(BeZero())
			})
		})
		Context("job has a backup script", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/foo/bin/bbr/backup",
				}
			})

			It("uses the ssh connection to run the script", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal(
					"sudo mkdir -p /var/vcap/store/bbr-backup/foo && " +
						"sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/foo/ " +
						"ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/foo/ /var/vcap/jobs/foo/bin/bbr/backup"))
			})

			Context("backup script runs successfully", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("stdout"), []byte("stderr"), 0, nil)
				})
				It("succeeds", func() {
					Expect(backupError).NotTo(HaveOccurred())
				})

			})
			Context("backup script run has a connection error", func() {
				var connectionError = fmt.Errorf("wierd error")
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("stdout"), []byte("stderr"), 0, connectionError)
				})
				It("fails", func() {
					Expect(backupError).To(MatchError(ContainSubstring(connectionError.Error())))
				})
			})
			Context("backup script exits with a non zero exit code", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("stdout"), []byte("stderr-script-errorred"), 1, nil)
				})
				It("fails", func() {
					Expect(backupError).To(MatchError(ContainSubstring("stderr-script-errorred")))
				})
			})
		})
	})

	Describe("Restore", func() {
		var restoreError error
		JustBeforeEach(func() {
			restoreError = job.Restore()
		})
		Context("job has no restore script", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/foo/bin/bbr/pre-backup-lock",
				}
			})
			It("should not run anything on the ssh connection", func() {
				Expect(sshConnection.RunCallCount()).To(BeZero())
			})
		})
		Context("job has a restore script", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/foo/bin/bbr/restore",
				}
			})

			It("uses the ssh connection to run the script", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal(
					"sudo BBR_ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/foo/ " +
						"ARTIFACT_DIRECTORY=/var/vcap/store/bbr-backup/foo/ /var/vcap/jobs/foo/bin/bbr/restore"))
			})

			Context("restore script runs successfully", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("stdout"), []byte("stderr"), 0, nil)
				})
				It("succeeds", func() {
					Expect(restoreError).NotTo(HaveOccurred())
				})

			})
			Context("restore script run has a connection error", func() {
				var connectionError = fmt.Errorf("wierd error")
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("stdout"), []byte("stderr"), 0, connectionError)
				})
				It("fails", func() {
					Expect(restoreError).To(MatchError(ContainSubstring(connectionError.Error())))
				})
			})
			Context("restore script exits with a non zero exit code", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("stdout"), []byte("stderr-script-errorred"), 1, nil)
				})
				It("fails", func() {
					Expect(restoreError).To(MatchError(ContainSubstring("stderr-script-errorred")))
				})
			})
		})
	})

	Describe("PreBackupLock", func() {
		var preBackupLockError error
		JustBeforeEach(func() {
			preBackupLockError = job.PreBackupLock()
		})
		Context("job has no pre-backup-lock script", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/foo/bin/bbr/restore",
				}
			})
			It("should not run anything on the ssh connection", func() {
				Expect(sshConnection.RunCallCount()).To(BeZero())
			})
		})
		Context("job has a pre-backup-lock script", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/foo/bin/bbr/pre-backup-lock",
				}
			})

			It("uses the ssh connection to run the script", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal(
					"sudo /var/vcap/jobs/foo/bin/bbr/pre-backup-lock"))
			})

			Context("pre-backup-lock script runs successfully", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("stdout"), []byte("stderr"), 0, nil)
				})
				It("succeeds", func() {
					Expect(preBackupLockError).NotTo(HaveOccurred())
				})

			})
			Context("pre-backup-lock script run has a connection error", func() {
				var connectionError = fmt.Errorf("wierd error")
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("stdout"), []byte("stderr"), 0, connectionError)
				})
				It("fails", func() {
					Expect(preBackupLockError).To(MatchError(ContainSubstring(connectionError.Error())))
				})
			})
			Context("pre-backup-lock script exits with a non zero exit code", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("stdout"), []byte("stderr-script-errorred"), 1, nil)
				})
				It("fails", func() {
					Expect(preBackupLockError).To(MatchError(ContainSubstring("stderr-script-errorred")))
				})
			})
		})
	})



	Describe("PostBackupUnlock", func() {
		var postBackupUnlockError error
		JustBeforeEach(func() {
			postBackupUnlockError = job.PostBackupUnlock()
		})
		Context("job has no post-backup-unlock script", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/foo/bin/bbr/restore",
				}
			})
			It("should not run anything on the ssh connection", func() {
				Expect(sshConnection.RunCallCount()).To(BeZero())
			})
		})
		Context("job has a post-backup-unlock script", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/foo/bin/bbr/post-backup-unlock",
				}
			})

			It("uses the ssh connection to run the script", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal(
					"sudo /var/vcap/jobs/foo/bin/bbr/post-backup-unlock"))
			})

			Context("post-backup-unlock script runs successfully", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("stdout"), []byte("stderr"), 0, nil)
				})
				It("succeeds", func() {
					Expect(postBackupUnlockError).NotTo(HaveOccurred())
				})

			})
			Context("post-backup-unlock script run has a connection error", func() {
				var connectionError = fmt.Errorf("wierd error")
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("stdout"), []byte("stderr"), 0, connectionError)
				})
				It("fails", func() {
					Expect(postBackupUnlockError).To(MatchError(ContainSubstring(connectionError.Error())))
				})
			})
			Context("post-backup-unlock script exits with a non zero exit code", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("stdout"), []byte("stderr-script-errorred"), 1, nil)
				})
				It("fails", func() {
					Expect(postBackupUnlockError).To(MatchError(ContainSubstring("stderr-script-errorred")))
				})
			})
		})
	})


	Describe("PostRestoreUnlock", func() {
		var postRestoreUnlockError error
		JustBeforeEach(func() {
			postRestoreUnlockError = job.PostRestoreUnlock()
		})
		Context("job has no post-restore-unlock script", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/foo/bin/bbr/restore",
				}
			})
			It("should not run anything on the ssh connection", func() {
				Expect(sshConnection.RunCallCount()).To(BeZero())
			})
		})
		Context("job has a post-restore-unlock script", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/foo/bin/bbr/post-restore-unlock",
				}
			})

			It("uses the ssh connection to run the script", func() {
				Expect(sshConnection.RunCallCount()).To(Equal(1))
				Expect(sshConnection.RunArgsForCall(0)).To(Equal(
					"sudo /var/vcap/jobs/foo/bin/bbr/post-restore-unlock"))
			})

			Context("post-restore-unlock script runs successfully", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("stdout"), []byte("stderr"), 0, nil)
				})
				It("succeeds", func() {
					Expect(postRestoreUnlockError).NotTo(HaveOccurred())
				})

			})
			Context("post-restore-unlock script run has a connection error", func() {
				var connectionError = fmt.Errorf("wierd error")
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("stdout"), []byte("stderr"), 0, connectionError)
				})
				It("fails", func() {
					Expect(postRestoreUnlockError).To(MatchError(ContainSubstring(connectionError.Error())))
				})
			})
			Context("post-restore-unlock script exits with a non zero exit code", func() {
				BeforeEach(func() {
					sshConnection.RunReturns([]byte("stdout"), []byte("stderr-script-errorred"), 1, nil)
				})
				It("fails", func() {
					Expect(postRestoreUnlockError).To(MatchError(ContainSubstring("stderr-script-errorred")))
				})
			})
		})
	})
})
