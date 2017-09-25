package instance_test

import (
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Backup and Restore Scripts", func() {
	Describe("NewBackupAndRestoreScripts", func() {
		Context("Backup", func() {
			It("returns the matching script when it has only one backup script", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/bbr/backup",
					"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/bbr/backup"}))
			})

			It("returns empty when backup scripts is in a subfolder", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/foo/backup",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{}))
			})

			It("returns empty when backup scripts in bin with a subfolder", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/foo/bin/bbr/backup",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{}))
			})
			It("returns the matching scripts when there are multiple backup scripts", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/bbr/backup",
					"/var/vcap/jobs/consul_agent/bin/bbr/backup",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{
					"/var/vcap/jobs/cloud_controller_clock/bin/bbr/backup",
					"/var/vcap/jobs/consul_agent/bin/bbr/backup",
				}))
			})

			It("returns empty when there are backup scripts", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{}))
			})
		})

		Context("Restore", func() {
			It("returns the matching script when it has only one restore script", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/bbr/restore",
					"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/bbr/restore"}))
			})

			It("returns empty when restore scripts is in a subfolder", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/foo/restore",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{}))
			})

			It("returns empty when restore scripts in bin with a subfolder", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/foo/bin/bbr/restore",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{}))
			})
			It("returns the matching scripts when there are multiple restore scripts", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/bbr/restore",
					"/var/vcap/jobs/consul_agent/bin/bbr/restore",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{
					"/var/vcap/jobs/cloud_controller_clock/bin/bbr/restore",
					"/var/vcap/jobs/consul_agent/bin/bbr/restore",
				}))
			})

			It("returns empty when there are no restore scripts", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{}))
			})
		})

		Context("PreBackupLock", func() {
			It("returns the matching scripts", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/bbr/pre-backup-lock",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{
					"/var/vcap/jobs/cloud_controller_clock/bin/bbr/pre-backup-lock",
				}))
			})
		})

		Context("PostBackupUnlock", func() {
			It("returns the matching scripts", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/bbr/post-backup-unlock",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{
					"/var/vcap/jobs/cloud_controller_clock/bin/bbr/post-backup-unlock",
				}))
			})
		})

		Context("Metadata", func() {
			It("returns the matching scripts", func() {
				var allScripts = []string{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
					"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
					"/var/vcap/jobs/cloud_controller_clock/bin/bbr/metadata",
					"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
				Expect(NewBackupAndRestoreScripts(allScripts)).To(Equal(BackupAndRestoreScripts{
					"/var/vcap/jobs/cloud_controller_clock/bin/bbr/metadata",
				}))
			})
		})
	})

	Describe("BackupOnly", func() {
		It("returns the backup scripts when it only has one", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
				"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
				"/var/vcap/jobs/cloud_controller_clock/bin/bbr/backup",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.BackupOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/bbr/backup"}))
		})

		It("returns empty when it has none", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
				"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.BackupOnly()).To(Equal(BackupAndRestoreScripts{}))
		})

		It("returns all backup scripts when there are several", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/bbr/backup",
				"/var/vcap/jobs/cloud_controller/bin/bbr/backup",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.BackupOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/bbr/backup",
				"/var/vcap/jobs/cloud_controller/bin/bbr/backup",
			}))
		})
	})

	Describe("RestoreOnly", func() {
		It("returns the backup scripts when it only has one", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
				"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
				"/var/vcap/jobs/cloud_controller_clock/bin/bbr/restore",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.RestoreOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/bbr/restore"}))
		})

		It("returns empty when it has none", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
				"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.RestoreOnly()).To(Equal(BackupAndRestoreScripts{}))
		})

		It("returns all backup scripts when there are several", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/bbr/restore",
				"/var/vcap/jobs/cloud_controller/bin/bbr/restore",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.RestoreOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/bbr/restore",
				"/var/vcap/jobs/cloud_controller/bin/bbr/restore",
			}))
		})
	})

	Describe("MetadataOnly", func() {
		It("returns the backup scripts when it only has one", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
				"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
				"/var/vcap/jobs/cloud_controller_clock/bin/bbr/metadata",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.MetadataOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/bbr/metadata"}))
		})

		It("returns empty when it has none", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
				"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.MetadataOnly()).To(Equal(BackupAndRestoreScripts{}))
		})

		It("returns all backup scripts when there are several", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/bbr/metadata",
				"/var/vcap/jobs/cloud_controller/bin/bbr/metadata",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.MetadataOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/bbr/metadata",
				"/var/vcap/jobs/cloud_controller/bin/bbr/metadata",
			}))
		})
	})

	Describe("PreBackupLockOnly", func() {
		It("returns the pre-backup-lock scripts when it only has one", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
				"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
				"/var/vcap/jobs/cloud_controller_clock/bin/bbr/backup",
				"/var/vcap/jobs/cloud_controller_clock/bin/bbr/pre-backup-lock",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.PreBackupLockOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/bbr/pre-backup-lock"}))
		})

		It("returns empty when it has none", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
				"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.PreBackupLockOnly()).To(Equal(BackupAndRestoreScripts{}))
		})

		It("returns all pre-backup-lock scripts when there are several", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/bbr/pre-backup-lock",
				"/var/vcap/jobs/cloud_controller/bin/bbr/pre-backup-lock",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.PreBackupLockOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/bbr/pre-backup-lock",
				"/var/vcap/jobs/cloud_controller/bin/bbr/pre-backup-lock",
			}))
		})
	})

	Describe("SinglePostRestoreUnlockScript", func() {
		It("returns exactly one post-restore-unlock script", func() {
			s := BackupAndRestoreScripts{
				"/var/vcap/jobs/job1/bin/bbr/post-restore-unlock",
				"/var/vcap/jobs/job2/bin/bbr/post-restore-unlock",
				"/var/vcap/jobs/job1/bin/bbr/other-script",
			}
			Expect(s.SinglePostRestoreUnlockScript()).To(Equal(Script("/var/vcap/jobs/job1/bin/bbr/post-restore-unlock")))
		})

		It("returns empty string when it has none", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
				"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.SinglePostRestoreUnlockScript()).To(Equal(Script("")))
		})
	})

	Describe("PostBackupUnlockOnly", func() {
		It("returns the post-backup-unlock scripts when it only has one", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
				"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
				"/var/vcap/jobs/cloud_controller_clock/bin/bbr/backup",
				"/var/vcap/jobs/cloud_controller_clock/bin/bbr/post-backup-unlock",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.PostBackupUnlockOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/bbr/post-backup-unlock"}))
		})

		It("returns empty when it has none", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/baz",
				"/var/vcap/jobs/cloud_controller_clock/bin/cloud_controller_clock_ctl",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.PostBackupUnlockOnly()).To(Equal(BackupAndRestoreScripts{}))
		})

		It("returns all post-backup-unlock scripts when there are several", func() {
			s := BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/bbr/post-backup-unlock",
				"/var/vcap/jobs/cloud_controller/bin/bbr/post-backup-unlock",
				"/var/vcap/jobs/cloud_controller_clock/bin/foo/bar",
				"/var/vcap/jobs/cloud_controller_clock/bin/pre-start"}
			Expect(s.PostBackupUnlockOnly()).To(Equal(BackupAndRestoreScripts{"/var/vcap/jobs/cloud_controller_clock/bin/bbr/post-backup-unlock",
				"/var/vcap/jobs/cloud_controller/bin/bbr/post-backup-unlock",
			}))
		})
	})
})
