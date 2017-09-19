package instance_test

import (
	"log"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh/fakes"
)

var _ = Describe("Jobs", func() {
	var jobs instance.Jobs
	var scripts instance.BackupAndRestoreScripts
	var artifactNames map[string]instance.Metadata
	var sshConnection *fakes.FakeSSHConnection
	var logger boshlog.Logger

	BeforeEach(func() {
		artifactNames = map[string]instance.Metadata{}
		sshConnection = new(fakes.FakeSSHConnection)

		combinedLog := log.New(GinkgoWriter, "[instance-test] ", log.Lshortfile)
		logger = boshlog.New(boshlog.LevelDebug, combinedLog, combinedLog)
	})

	JustBeforeEach(func() {
		jobs = instance.NewJobs(sshConnection, "identifier", logger, scripts, artifactNames)

	})

	Describe("NewJobs", func() {
		Context("when there are two jobs each with a backup script", func() {
			BeforeEach(func() {
				scripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/foo/bin/bbr/backup",
					"/var/vcap/jobs/bar/bin/bbr/backup",
				}
			})
			It("groups scripts to create jobs", func() {
				Expect(jobs).To(ConsistOf(
					instance.NewJob(sshConnection, "identifier", logger,
						instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/bbr/backup"}, instance.Metadata{}),
					instance.NewJob(sshConnection, "identifier", logger,
						instance.BackupAndRestoreScripts{"/var/vcap/jobs/bar/bin/bbr/backup"}, instance.Metadata{}),
				))
			})
		})

		Context("when there is one job with a backup script", func() {
			BeforeEach(func() {
				scripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/foo/bin/bbr/backup",
				}
			})
			It("groups scripts to create jobs", func() {
				Expect(jobs).To(ConsistOf(
					instance.NewJob(sshConnection, "identifier", logger,
						instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/bbr/backup"}, instance.Metadata{}),
				))
			})
		})

		Context("when there is one job with a backup script and an artifact name", func() {
			BeforeEach(func() {
				scripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/foo/bin/bbr/backup",
				}
				artifactNames = map[string]instance.Metadata{
					"foo": {
						BackupName: "a-bosh-backup",
					},
				}
			})

			It("creates a job with the correct artifact name", func() {
				Expect(jobs).To(ConsistOf(
					instance.NewJob(sshConnection, "identifier", logger,
						instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/bbr/backup"},
						instance.Metadata{
							BackupName: "a-bosh-backup",
						},
					),
				))
			})
		})

		Context("when there are two jobs, both with backup scripts and unique metadata names", func() {
			BeforeEach(func() {
				scripts = instance.BackupAndRestoreScripts{
					"/var/vcap/jobs/foo/bin/bbr/backup",
					"/var/vcap/jobs/bar/bin/bbr/backup",
				}
				artifactNames = map[string]instance.Metadata{
					"foo": {
						BackupName: "a-bosh-backup",
					},
					"bar": {
						BackupName: "another-backup",
					},
				}
			})

			It("creates two jobs with the correct artifact names", func() {
				Expect(jobs).To(ConsistOf(
					instance.NewJob(sshConnection, "identifier", logger,
						instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/bbr/backup"},
						instance.Metadata{
							BackupName: "a-bosh-backup",
						},
					),
					instance.NewJob(sshConnection, "identifier", logger,
						instance.BackupAndRestoreScripts{"/var/vcap/jobs/bar/bin/bbr/backup"},
						instance.Metadata{
							BackupName: "another-backup",
						},
					),
				))
			})
		})

	})

	Context("contains jobs with backup script", func() {
		BeforeEach(func() {
			scripts = instance.BackupAndRestoreScripts{
				"/var/vcap/jobs/foo/bin/bbr/backup",
				"/var/vcap/jobs/bar/bin/bbr/restore",
			}
		})

		Describe("Backupable", func() {
			It("returns the backupable job", func() {
				Expect(jobs.Backupable()).To(ConsistOf(
					instance.NewJob(sshConnection, "identifier", logger,
						instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/bbr/backup"}, instance.Metadata{}),
				))
			})
		})

		Describe("AnyAreBackupable", func() {
			It("returns true", func() {
				Expect(jobs.AnyAreBackupable()).To(BeTrue())
			})
		})
	})

	Context("contains no jobs with backup script", func() {
		BeforeEach(func() {
			scripts = instance.BackupAndRestoreScripts{
				"/var/vcap/jobs/bar/bin/bbr/restore",
			}
		})

		Describe("Backupable", func() {
			It("returns empty", func() {
				Expect(jobs.Backupable()).To(BeEmpty())
			})
		})

		Describe("AnyAreBackupable", func() {
			It("returns false", func() {
				Expect(jobs.AnyAreBackupable()).To(BeFalse())
			})
		})
	})

	Context("contains jobs with post-backup-lock scripts", func() {

		BeforeEach(func() {
			scripts = instance.BackupAndRestoreScripts{
				"/var/vcap/jobs/foo/bin/bbr/backup",
				"/var/vcap/jobs/foo/bin/bbr/post-backup-unlock",
				"/var/vcap/jobs/bar/bin/bbr/restore",
			}
		})
	})

	Context("contains jobs with restore scripts", func() {
		BeforeEach(func() {
			scripts = instance.BackupAndRestoreScripts{
				"/var/vcap/jobs/foo/bin/bbr/backup",
				"/var/vcap/jobs/foo/bin/bbr/post-backup-unlock",
				"/var/vcap/jobs/bar/bin/bbr/restore",
			}
		})

		Describe("Restorable", func() {
			It("returns the unlockable job", func() {
				Expect(jobs.Restorable()).To(ConsistOf(instance.NewJob(sshConnection, "identifier", logger,
					instance.BackupAndRestoreScripts{"/var/vcap/jobs/bar/bin/bbr/restore"}, instance.Metadata{}),
				))
			})
		})

		Describe("AnyAreRestorable", func() {
			It("returns true", func() {
				Expect(jobs.AnyAreRestorable()).To(BeTrue())
			})
		})

		Describe("AnyNeedDefaultArtifactsForRestore", func() {
			It("returns true, as all of the jobs need a default artifact for restore", func() {
				Expect(jobs.AnyNeedDefaultArtifactsForRestore()).To(BeTrue())
			})
		})
	})

	Context("contains no jobs with backup script", func() {
		BeforeEach(func() {
			scripts = instance.BackupAndRestoreScripts{
				"/var/vcap/jobs/bar/bin/bbr/backup",
			}
		})

		It("returns empty", func() {
			Expect(jobs.Restorable()).To(BeEmpty())
		})
	})

	Context("contains no jobs with named backup artifacts", func() {
		Describe("CustomBackupArtifactNames", func() {
			It("returns empty", func() {
				Expect(jobs.CustomBackupArtifactNames()).To(BeEmpty())
			})
		})
	})

	Context("contains jobs with a named backup artifact", func() {
		BeforeEach(func() {
			artifactNames = map[string]instance.Metadata{
				"bar": {
					BackupName: "my-cool-artifact",
				},
			}
			scripts = instance.BackupAndRestoreScripts{
				"/var/vcap/jobs/bar/bin/bbr/backup",
				"/var/vcap/jobs/bar/bin/bbr/restore",
				"/var/vcap/jobs/foo/bin/bbr/backup",
				"/var/vcap/jobs/baz/bin/bbr/restore",
			}
		})

		Describe("AnyNeedDefaultArtifactsForBackup", func() {
			It("returns true", func() {
				Expect(jobs.AnyNeedDefaultArtifactsForBackup()).To(BeTrue())
			})
		})
	})

	Context("contains jobs with a named restore artifact", func() {
		BeforeEach(func() {
			artifactNames = map[string]instance.Metadata{
				"bar": {
					RestoreName: "my-cool-restore",
				},
			}
			scripts = instance.BackupAndRestoreScripts{
				"/var/vcap/jobs/bar/bin/bbr/backup",
				"/var/vcap/jobs/bar/bin/bbr/restore",
				"/var/vcap/jobs/foo/bin/bbr/backup",
				"/var/vcap/jobs/baz/bin/bbr/restore",
			}
		})

		Describe("CustomRestoreArtifactNames", func() {
			It("returns a list of artifact names", func() {
				Expect(jobs.CustomRestoreArtifactNames()).To(ConsistOf("my-cool-restore"))
			})
		})

		Describe("AnyNeedDefaultArtifactsForRestore", func() {
			It("returns true, as job 'baz' needs a default artifact for restore", func() {
				Expect(jobs.AnyNeedDefaultArtifactsForRestore()).To(BeTrue())
			})
		})
	})

	Context("contains jobs with multiple named artifacts", func() {
		BeforeEach(func() {
			scripts = instance.BackupAndRestoreScripts{
				"/var/vcap/jobs/foo/bin/bbr/backup",
				"/var/vcap/jobs/bar/bin/bbr/backup",
			}
			artifactNames = map[string]instance.Metadata{
				"foo": {
					BackupName: "a-bosh-backup",
				},
				"bar": {
					BackupName: "another-backup",
				},
			}
		})

		Describe("CustomBackupArtifactNames", func() {
			It("returns a list of artifact names", func() {
				Expect(jobs.CustomBackupArtifactNames()).To(ConsistOf("a-bosh-backup", "another-backup"))
			})
		})

		Describe("AnyNeedDefaultArtifactsForRestore", func() {
			It("returns false, as none of the jobs need a default artifact for restore", func() {
				Expect(jobs.AnyNeedDefaultArtifactsForRestore()).To(BeFalse())
			})
		})

		Describe("AnyNeedDefaultArtifactsForBackup", func() {
			It("returns false", func() {
				Expect(jobs.AnyNeedDefaultArtifactsForBackup()).To(BeFalse())
			})
		})
	})
})
