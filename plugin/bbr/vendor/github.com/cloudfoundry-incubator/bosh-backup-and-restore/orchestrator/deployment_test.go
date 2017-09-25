package orchestrator_test

import (
	"fmt"
	"io"

	"bytes"

	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"
	"github.com/pkg/errors"
)

var _ = Describe("Deployment", func() {
	var (
		deployment orchestrator.Deployment
		logger     *fakes.FakeLogger
		instances  []orchestrator.Instance
		instance1  *fakes.FakeInstance
		instance2  *fakes.FakeInstance
		instance3  *fakes.FakeInstance
	)

	BeforeEach(func() {
		logger = new(fakes.FakeLogger)

		instance1 = new(fakes.FakeInstance)
		instance2 = new(fakes.FakeInstance)
		instance3 = new(fakes.FakeInstance)
	})

	JustBeforeEach(func() {
		deployment = orchestrator.NewDeployment(logger, instances)
	})

	Context("PreBackupLock", func() {
		var lockError error

		JustBeforeEach(func() {
			lockError = deployment.PreBackupLock()
		})

		Context("Single instance", func() {
			BeforeEach(func() {
				instances = []orchestrator.Instance{instance1}
			})

			It("does not fail", func() {
				Expect(lockError).NotTo(HaveOccurred())
			})

			It("locks the instance", func() {
				Expect(instance1.PreBackupLockCallCount()).To(Equal(1))
			})

			Context("if the pre-backup-lock fails", func() {
				lockErr := fmt.Errorf("something")

				BeforeEach(func() {
					instance1.PreBackupLockReturns(lockErr)
				})

				It("fails", func() {
					Expect(lockErr).To(HaveOccurred())
				})
			})
		})

		Context("Multiple instances", func() {
			BeforeEach(func() {
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("does not fail", func() {
				Expect(lockError).NotTo(HaveOccurred())
			})

			It("runs pre-backup-lock on all the instances", func() {
				Expect(instance1.PreBackupLockCallCount()).To(Equal(1))
				Expect(instance2.PreBackupLockCallCount()).To(Equal(1))
			})
		})

	})

	Context("Backup", func() {
		var (
			backupError error
		)
		JustBeforeEach(func() {
			backupError = deployment.Backup()
		})

		Context("Single instance, backupable", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(true)
				instances = []orchestrator.Instance{instance1}
			})
			It("does not fail", func() {
				Expect(backupError).NotTo(HaveOccurred())
			})
			It("backs up the instance", func() {
				Expect(instance1.BackupCallCount()).To(Equal(1))
			})
		})

		Context("Multiple instances, all backupable", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(true)
				instance2.IsBackupableReturns(true)
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("does not fail", func() {
				Expect(backupError).NotTo(HaveOccurred())
			})
			It("backs up the only the backupable instance", func() {
				Expect(instance1.BackupCallCount()).To(Equal(1))
				Expect(instance2.BackupCallCount()).To(Equal(1))
			})
		})
		Context("Multiple instances, some backupable", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(true)
				instance2.IsBackupableReturns(false)
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("does not fail", func() {
				Expect(backupError).NotTo(HaveOccurred())
			})
			It("backs up the only the backupable instance", func() {
				Expect(instance1.BackupCallCount()).To(Equal(1))
			})
			It("does not back up the non backupable instance", func() {
				Expect(instance2.BackupCallCount()).To(Equal(0))
			})
		})

		Context("Multiple instances, some failing to backup", func() {
			BeforeEach(func() {
				backupError := fmt.Errorf("very clever sandwich")
				instance1.IsBackupableReturns(true)
				instance2.IsBackupableReturns(true)
				instance1.BackupReturns(backupError)
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("does not fail", func() {
				Expect(backupError).To(HaveOccurred())
			})

			It("stops invoking backup, after the error", func() {
				Expect(instance1.BackupCallCount()).To(Equal(1))
				Expect(instance2.BackupCallCount()).To(Equal(0))
			})
		})
	})

	Context("PostBackupUnlock", func() {
		var unlockError, expectedError error

		BeforeEach(func() {
			expectedError = fmt.Errorf("something went terribly wrong")
		})

		JustBeforeEach(func() {
			unlockError = deployment.PostBackupUnlock()
		})

		Context("Single instance", func() {
			BeforeEach(func() {
				instance1.PostBackupUnlockReturns(nil)
				instances = []orchestrator.Instance{instance1}
			})

			It("does not fail", func() {
				Expect(unlockError).NotTo(HaveOccurred())
			})

			It("unlocks the instance", func() {
				Expect(instance1.PostBackupUnlockCallCount()).To(Equal(1))
			})
		})

		Context("single instance that fails to unlock", func() {
			BeforeEach(func() {
				instance1.PostBackupUnlockReturns(expectedError)
				instances = []orchestrator.Instance{instance1}
			})

			It("fails", func() {
				Expect(unlockError).To(HaveOccurred())
			})

			It("attempts to unlock the instance", func() {
				Expect(instance1.PostBackupUnlockCallCount()).To(Equal(1))
			})
		})

		Context("Multiple instances", func() {
			BeforeEach(func() {
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("does not fail", func() {
				Expect(unlockError).NotTo(HaveOccurred())
			})

			It("unlocks the instances", func() {
				Expect(instance1.PostBackupUnlockCallCount()).To(Equal(1))
				Expect(instance2.PostBackupUnlockCallCount()).To(Equal(1))
			})
		})

		Context("Multiple instances, where one fails to unlock", func() {
			BeforeEach(func() {
				instance1.PostBackupUnlockReturns(expectedError)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("fails", func() {
				Expect(unlockError).To(HaveOccurred())
			})

			It("attempts to unlock both instances", func() {
				Expect(instance1.PostBackupUnlockCallCount()).To(Equal(1))
				Expect(instance2.PostBackupUnlockCallCount()).To(Equal(1))
			})

			It("returns the expected single error", func() {
				Expect(unlockError).To(MatchError(ContainSubstring(expectedError.Error())))
			})
		})

		Context("Multiple instances, all fail to unlock", func() {
			var secondError error

			BeforeEach(func() {
				instance1.PostBackupUnlockReturns(expectedError)
				secondError = fmt.Errorf("something else went wrong")
				instance2.PostBackupUnlockReturns(secondError)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("fails", func() {
				Expect(unlockError).To(HaveOccurred())
			})

			It("attempts to unlock both instances", func() {
				Expect(instance1.PostBackupUnlockCallCount()).To(Equal(1))
				Expect(instance2.PostBackupUnlockCallCount()).To(Equal(1))
			})

			It("returns all the expected errors", func() {
				Expect(unlockError).To(MatchError(ContainSubstring(expectedError.Error())))
				Expect(unlockError).To(MatchError(ContainSubstring(secondError.Error())))
			})
		})
	})

	Context("IsBackupable", func() {
		var IsBackupable bool

		JustBeforeEach(func() {
			IsBackupable = deployment.IsBackupable()
		})

		Context("Single instance with a backup script", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(true)
				instances = []orchestrator.Instance{instance1}
			})

			It("checks if the instance has a backup script", func() {
				Expect(instance1.IsBackupableCallCount()).To(Equal(1))
			})

			It("returns true", func() {
				Expect(IsBackupable).To(BeTrue())
			})
		})

		Context("Single instance, no backup script", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(false)
				instances = []orchestrator.Instance{instance1}
			})

			It("checks if the instance has a backup script", func() {
				Expect(instance1.IsBackupableCallCount()).To(Equal(1))
			})

			It("returns true", func() {
				Expect(IsBackupable).To(BeFalse())
			})
		})

		Context("Multiple instances, some with backup scripts", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(false)
				instance2.IsBackupableReturns(true)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("returns true", func() {
				Expect(instance1.IsBackupableCallCount()).To(Equal(1))
				Expect(instance2.IsBackupableCallCount()).To(Equal(1))
				Expect(IsBackupable).To(BeTrue())
			})
		})
		Context("Multiple instances, none with backup scripts", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(false)
				instance2.IsBackupableReturns(false)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("returns false", func() {
				Expect(instance1.IsBackupableCallCount()).To(Equal(1))
				Expect(instance2.IsBackupableCallCount()).To(Equal(1))
				Expect(IsBackupable).To(BeFalse())
			})
		})
	})

	Context("CheckArtifactDir", func() {
		var artifactDirError error

		BeforeEach(func() {
			instance1.NameReturns("foo")
			instance1.IDReturns("0")

			instance2.NameReturns("bar")
			instance2.IDReturns("0")

			instance1.ArtifactDirExistsReturns(false, nil)
			instance2.ArtifactDirExistsReturns(false, nil)
			instances = []orchestrator.Instance{instance1, instance2}
		})

		JustBeforeEach(func() {
			artifactDirError = deployment.CheckArtifactDir()
		})

		Context("when artifact directory does not exist", func() {
			It("does not fail", func() {
				Expect(artifactDirError).NotTo(HaveOccurred())
			})
		})

		Context("when artifact directory exists", func() {
			BeforeEach(func() {
				instance1.ArtifactDirExistsReturns(true, nil)
				instance2.ArtifactDirExistsReturns(true, nil)
			})

			It("fails", func() {
				Expect(artifactDirError).To(HaveOccurred())
			})

			It("the error includes the names of the instances on which the directory exists", func() {
				Expect(artifactDirError.Error()).To(ContainSubstring("Directory /var/vcap/store/bbr-backup already exists on instance foo/0"))
				Expect(artifactDirError.Error()).To(ContainSubstring("Directory /var/vcap/store/bbr-backup already exists on instance bar/0"))
			})
		})

		Context("when call to check artifact directory fails", func() {
			BeforeEach(func() {
				instances = []orchestrator.Instance{instance1}
				instance1.ArtifactDirExistsReturns(false, fmt.Errorf("oh dear"))
			})

			It("fails", func() {
				Expect(artifactDirError).To(HaveOccurred())
			})

			It("the error includes the names of the instances on which the error occurred", func() {
				Expect(artifactDirError.Error()).To(ContainSubstring("Error checking /var/vcap/store/bbr-backup on instance foo/0"))
			})
		})

	})

	Context("CustomArtifactNamesMatch", func() {
		var artifactMatchError error

		JustBeforeEach(func() {
			artifactMatchError = deployment.CustomArtifactNamesMatch()
		})
		BeforeEach(func() {
			instances = []orchestrator.Instance{instance1, instance2}
		})

		Context("when the custom names match", func() {
			BeforeEach(func() {
				instance1.CustomBackupArtifactNamesReturns([]string{"custom1"})
				instance2.CustomRestoreArtifactNamesReturns([]string{"custom1"})
			})

			It("is nil", func() {
				Expect(artifactMatchError).NotTo(HaveOccurred())
			})
		})

		Context("when the multiple custom names match", func() {
			BeforeEach(func() {
				instance1.CustomBackupArtifactNamesReturns([]string{"custom1"})
				instance1.CustomRestoreArtifactNamesReturns([]string{"custom2"})
				instance2.CustomBackupArtifactNamesReturns([]string{"custom2"})
				instance2.CustomRestoreArtifactNamesReturns([]string{"custom1"})
			})

			It("is nil", func() {
				Expect(artifactMatchError).NotTo(HaveOccurred())
			})
		})
		Context("when the custom dont match", func() {
			BeforeEach(func() {
				instance1.CustomBackupArtifactNamesReturns([]string{"custom1"})
				instance2.NameReturns("job2Name")
				instance2.CustomRestoreArtifactNamesReturns([]string{"custom2"})
			})

			It("to return an error", func() {
				Expect(artifactMatchError).To(MatchError("The job2Name restore script expects a backup script which produces custom2 artifact which is not present in the deployment."))
			})
		})
	})

	Context("HasUniqueCustomArtifactNames", func() {
		var isValid bool

		JustBeforeEach(func() {
			isValid = deployment.HasUniqueCustomArtifactNames()
		})

		Context("Single instance, with unique metadata", func() {
			BeforeEach(func() {
				instance1.CustomBackupArtifactNamesReturns([]string{"custom1", "custom2"})
				instances = []orchestrator.Instance{instance1}
			})

			It("returns true", func() {
				Expect(isValid).To(BeTrue())
			})
		})

		Context("Single instance, with non-unique metadata", func() {
			BeforeEach(func() {
				instance1.CustomBackupArtifactNamesReturns([]string{"the-same", "the-same"})
				instances = []orchestrator.Instance{instance1}
			})

			It("returns false", func() {
				Expect(isValid).To(BeFalse())
			})
		})

		Context("multiple instances, with unique metadata", func() {
			BeforeEach(func() {
				instance1.CustomBackupArtifactNamesReturns([]string{"custom1", "custom2"})
				instance2.CustomBackupArtifactNamesReturns([]string{"custom3", "custom4"})
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("returns true", func() {
				Expect(isValid).To(BeTrue())
			})
		})

		Context("multiple instances, with non-unique metadata", func() {
			BeforeEach(func() {
				instance1.CustomBackupArtifactNamesReturns([]string{"custom1", "custom2"})
				instance2.CustomBackupArtifactNamesReturns([]string{"custom2", "custom4"})
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("returns false", func() {
				Expect(isValid).To(BeFalse())
			})
		})

		Context("multiple instances, with no metadata", func() {
			BeforeEach(func() {
				instance1.CustomBackupArtifactNamesReturns([]string{})
				instance2.CustomBackupArtifactNamesReturns([]string{})
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("returns true", func() {
				Expect(isValid).To(BeTrue())
			})
		})
	})

	Context("Restore", func() {
		var (
			restoreError error
		)
		JustBeforeEach(func() {
			restoreError = deployment.Restore()
		})

		Context("Single instance, restoreable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(true)
				instances = []orchestrator.Instance{instance1}
			})
			It("does not fail", func() {
				Expect(restoreError).NotTo(HaveOccurred())
			})
			It("restores the instance", func() {
				Expect(instance1.RestoreCallCount()).To(Equal(1))
			})
			It("logs before restoring", func() {
				_, message, _ := logger.InfoArgsForCall(0)
				Expect(message).To(Equal("Running restore scripts..."))
			})
		})

		Context("Single instance, not restoreable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(false)
				instances = []orchestrator.Instance{instance1}
			})
			It("does not fail", func() {
				Expect(restoreError).NotTo(HaveOccurred())
			})
			It("restores the instance", func() {
				Expect(instance1.RestoreCallCount()).To(Equal(0))
			})
		})

		Context("Multiple instances, all restoreable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(true)
				instance2.IsRestorableReturns(true)
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("does not fail", func() {
				Expect(restoreError).NotTo(HaveOccurred())
			})
			It("backs up the only the restoreable instance", func() {
				Expect(instance1.RestoreCallCount()).To(Equal(1))
				Expect(instance2.RestoreCallCount()).To(Equal(1))
			})
		})
		Context("Multiple instances, some restorable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(true)
				instance2.IsRestorableReturns(false)
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("does not fail", func() {
				Expect(restoreError).NotTo(HaveOccurred())
			})
			It("backs up the only the restorable instance", func() {
				Expect(instance1.RestoreCallCount()).To(Equal(1))
			})
			It("does not back up the non restorable instance", func() {
				Expect(instance2.RestoreCallCount()).To(Equal(0))
			})
		})

		Context("Multiple instances, some failing to restore", func() {
			var restoreError = fmt.Errorf("and some salt and vinegar crisps")

			BeforeEach(func() {
				instance1.IsRestorableReturns(true)
				instance2.IsRestorableReturns(true)
				instance1.RestoreReturns(restoreError)
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("does not fail", func() {
				Expect(restoreError).To(MatchError(restoreError))
			})

			It("stops invoking backup, after the error", func() {
				Expect(instance1.RestoreCallCount()).To(Equal(1))
				Expect(instance2.RestoreCallCount()).To(Equal(0))
			})
		})
	})

	Context("PostRestoreUnlock", func() {
		var unlockError error

		BeforeEach(func() {
			instances = []orchestrator.Instance{instance1, instance2, instance3}
		})

		JustBeforeEach(func() {
			unlockError = deployment.PostRestoreUnlock()
		})

		It("Calls PostRestoreUnlock on all instances", func() {
			Expect(instance1.PostRestoreUnlockCallCount()).To(Equal(1))
			Expect(instance2.PostRestoreUnlockCallCount()).To(Equal(1))
			Expect(instance3.PostRestoreUnlockCallCount()).To(Equal(1))
		})

		Context("when some instances fail to PostRestoreUnlock", func() {
			BeforeEach(func() {
				instance1.PostRestoreUnlockReturns(errors.New("instance 1 failed to unlock"))
				instance2.PostRestoreUnlockReturns(errors.New("instance 2 failed to unlock"))
			})

			It("Continues attempting unlock on the rest of the deployment", func() {
				By("attempting unlock on subsequent instances", func() {
					Expect(instance3.PostRestoreUnlockCallCount()).To(Equal(1))
				})

				By("returning an amalgamated error", func() {
					Expect(unlockError).To(HaveOccurred())
					Expect(unlockError.Error()).To(ContainSubstring("instance 1 failed to unlock"))
					Expect(unlockError.Error()).To(ContainSubstring("instance 2 failed to unlock"))
				})
			})
		})
	})

	Context("CopyLocalBackupToRemote", func() {
		var (
			artifact                     *fakes.FakeBackup
			backupArtifact               *fakes.FakeBackupArtifact
			copyLocalBackupToRemoteError error
			reader                       io.ReadCloser
		)

		BeforeEach(func() {
			reader = ioutil.NopCloser(bytes.NewBufferString("this-is-some-backup-data"))
			instance1.IsRestorableReturns(true)
			instances = []orchestrator.Instance{instance1}
			artifact = new(fakes.FakeBackup)
			artifact.ReadArtifactReturns(reader, nil)
			backupArtifact = new(fakes.FakeBackupArtifact)
		})

		JustBeforeEach(func() {
			copyLocalBackupToRemoteError = deployment.CopyLocalBackupToRemote(artifact)
		})

		Context("Single instance, restorable", func() {
			var checkSums = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}

			BeforeEach(func() {
				artifact.FetchChecksumReturns(checkSums, nil)
				backupArtifact.ChecksumReturns(checkSums, nil)
				instance1.ArtifactsToRestoreReturns([]orchestrator.BackupArtifact{backupArtifact})
			})

			It("does not fail", func() {
				Expect(copyLocalBackupToRemoteError).NotTo(HaveOccurred())
			})

			It("checks the remote after transfer", func() {
				Expect(backupArtifact.ChecksumCallCount()).To(Equal(1))
			})

			It("marks the artifact directory as created after transfer", func() {
				Expect(instance1.MarkArtifactDirCreatedCallCount()).To(Equal(1))
			})

			It("checks the local checksum", func() {
				Expect(artifact.FetchChecksumCallCount()).To(Equal(1))
				Expect(artifact.FetchChecksumArgsForCall(0)).To(Equal(backupArtifact))
			})

			It("logs when the copy starts and finishes", func() {
				Expect(logger.InfoCallCount()).To(Equal(2))
				_, logLine, _ := logger.InfoArgsForCall(0)
				Expect(logLine).To(ContainSubstring("Copying backup"))

				_, logLine, _ = logger.InfoArgsForCall(1)
				Expect(logLine).To(ContainSubstring("Done"))
			})

			It("streams the backup file to the restorable instance", func() {
				Expect(backupArtifact.StreamToRemoteCallCount()).To(Equal(1))
				expectedReader := backupArtifact.StreamToRemoteArgsForCall(0)
				Expect(expectedReader).To(Equal(reader))
			})

			Context("problem occurs streaming to instance", func() {
				BeforeEach(func() {
					backupArtifact.StreamToRemoteReturns(fmt.Errorf("streaming had a problem"))
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError("streaming had a problem"))
				})
			})
			Context("problem calculating shasum on local", func() {
				var checksumError = fmt.Errorf("I am so clever")
				BeforeEach(func() {
					artifact.FetchChecksumReturns(nil, checksumError)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError(checksumError))
				})
			})
			Context("problem calculating shasum on remote", func() {
				var checksumError = fmt.Errorf("grr")
				BeforeEach(func() {
					backupArtifact.ChecksumReturns(nil, checksumError)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError(checksumError))
				})
			})

			Context("shas dont match after transfer", func() {
				BeforeEach(func() {
					backupArtifact.ChecksumReturns(orchestrator.BackupChecksum{"shas": "they dont match"}, nil)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(MatchError(ContainSubstring("Backup couldn't be transfered, checksum failed")))
				})
			})

			Context("problem occurs while reading from backup", func() {
				BeforeEach(func() {
					artifact.ReadArtifactReturns(nil, fmt.Errorf("leave me alone"))
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError("leave me alone"))
				})
			})
		})

		Context("multiple instances, one restorable", func() {
			var checkSums = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}

			BeforeEach(func() {
				instance2.IsRestorableReturns(false)

				artifact.FetchChecksumReturns(checkSums, nil)
				backupArtifact.ChecksumReturns(checkSums, nil)
				instance1.ArtifactsToRestoreReturns([]orchestrator.BackupArtifact{backupArtifact})
			})

			It("does not fail", func() {
				Expect(copyLocalBackupToRemoteError).NotTo(HaveOccurred())
			})

			It("does not ask artifacts for the non restorable instance", func() {
				Expect(instance2.ArtifactsToRestoreCallCount()).To(BeZero())
			})

			It("checks the remote after transfer", func() {
				Expect(backupArtifact.ChecksumCallCount()).To(Equal(1))
			})

			It("checks the local checksum", func() {
				Expect(artifact.FetchChecksumCallCount()).To(Equal(1))
				Expect(artifact.FetchChecksumArgsForCall(0)).To(Equal(backupArtifact))
			})

			It("streams the backup file to the restorable instance", func() {
				Expect(backupArtifact.StreamToRemoteCallCount()).To(Equal(1))
				expectedReader := backupArtifact.StreamToRemoteArgsForCall(0)
				Expect(expectedReader).To(Equal(reader))
			})

			Context("problem occurs streaming to instance", func() {
				BeforeEach(func() {
					backupArtifact.StreamToRemoteReturns(fmt.Errorf("I'm still here"))
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError("I'm still here"))
				})
			})
			Context("problem calculating shasum on local", func() {
				var checksumError = fmt.Errorf("oh well")
				BeforeEach(func() {
					artifact.FetchChecksumReturns(nil, checksumError)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError(checksumError))
				})
			})
			Context("problem calculating shasum on remote", func() {
				var checksumError = fmt.Errorf("grr")
				BeforeEach(func() {
					backupArtifact.ChecksumReturns(nil, checksumError)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError(checksumError))
				})
			})

			Context("shas dont match after transfer", func() {
				BeforeEach(func() {
					backupArtifact.ChecksumReturns(orchestrator.BackupChecksum{"shas": "they dont match"}, nil)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(MatchError(ContainSubstring("Backup couldn't be transfered, checksum failed")))
				})
			})

			Context("problem occurs while reading from backup", func() {
				BeforeEach(func() {
					artifact.ReadArtifactReturns(nil, fmt.Errorf("foo bar baz read error"))
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError("foo bar baz read error"))
				})
			})
		})

		Context("Single instance, restorable, with multiple artifacts", func() {
			var checkSums = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}
			var anotherBackupArtifact *fakes.FakeBackupArtifact

			BeforeEach(func() {
				anotherBackupArtifact = new(fakes.FakeBackupArtifact)
				artifact.FetchChecksumReturns(checkSums, nil)
				backupArtifact.ChecksumReturns(checkSums, nil)
				anotherBackupArtifact.ChecksumReturns(checkSums, nil)

				instance1.ArtifactsToRestoreReturns([]orchestrator.BackupArtifact{backupArtifact, anotherBackupArtifact})
			})

			It("does not fail", func() {
				Expect(copyLocalBackupToRemoteError).NotTo(HaveOccurred())
			})

			It("checks the remote after transfer", func() {
				Expect(backupArtifact.ChecksumCallCount()).To(Equal(1))
			})

			It("checks the local checksum", func() {
				Expect(artifact.FetchChecksumCallCount()).To(Equal(2))
				Expect(artifact.FetchChecksumArgsForCall(0)).To(Equal(backupArtifact))
				Expect(artifact.FetchChecksumArgsForCall(0)).To(Equal(anotherBackupArtifact))
			})

			It("streams the backup file to the restorable instance", func() {
				Expect(backupArtifact.StreamToRemoteCallCount()).To(Equal(1))
				expectedReader := backupArtifact.StreamToRemoteArgsForCall(0)
				Expect(expectedReader).To(Equal(reader))
			})

			Context("problem occurs streaming to instance", func() {
				BeforeEach(func() {
					backupArtifact.StreamToRemoteReturns(fmt.Errorf("this is a problem"))
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError("this is a problem"))
				})
			})
			Context("problem calculating shasum on local", func() {
				var checksumError = fmt.Errorf("checksum error occurred")
				BeforeEach(func() {
					artifact.FetchChecksumReturns(nil, checksumError)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError(checksumError))
				})
			})
			Context("problem calculating shasum on remote", func() {
				var checksumError = fmt.Errorf("grr")
				BeforeEach(func() {
					backupArtifact.ChecksumReturns(nil, checksumError)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError(checksumError))
				})
			})

			Context("shas dont match after transfer", func() {
				BeforeEach(func() {
					backupArtifact.ChecksumReturns(orchestrator.BackupChecksum{"shas": "they dont match"}, nil)
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(MatchError(ContainSubstring("Backup couldn't be transfered, checksum failed")))
				})
			})

			Context("problem occurs while reading from backup", func() {
				BeforeEach(func() {
					artifact.ReadArtifactReturns(nil, fmt.Errorf("a huge problem"))
				})

				It("fails", func() {
					Expect(copyLocalBackupToRemoteError).To(HaveOccurred())
					Expect(copyLocalBackupToRemoteError).To(MatchError("a huge problem"))
				})
			})
		})

	})

	Context("IsRestorable", func() {
		var (
			isRestorableError error
			isRestorable      bool
		)
		JustBeforeEach(func() {
			isRestorable = deployment.IsRestorable()
		})

		Context("Single instance, restorable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(true)
				instances = []orchestrator.Instance{instance1}
			})

			It("does not fail", func() {
				Expect(isRestorableError).NotTo(HaveOccurred())
			})

			It("checks if the instance is restorable", func() {
				Expect(instance1.IsRestorableCallCount()).To(Equal(1))
			})

			It("returns true if the instance is restorable", func() {
				Expect(isRestorable).To(BeTrue())
			})
		})
		Context("Single instance, not restorable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(false)
				instances = []orchestrator.Instance{instance1}
			})

			It("does not fail", func() {
				Expect(isRestorableError).NotTo(HaveOccurred())
			})

			It("checks if the instance is restorable", func() {
				Expect(instance1.IsRestorableCallCount()).To(Equal(1))
			})

			It("returns true if any instance is restorable", func() {
				Expect(isRestorable).To(BeFalse())
			})
		})

		Context("Multiple instances, some restorable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(false)
				instance2.IsRestorableReturns(true)
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("does not fail", func() {
				Expect(isRestorableError).NotTo(HaveOccurred())
			})

			It("returns true if any instance is restorable", func() {
				Expect(instance1.IsRestorableCallCount()).To(Equal(1))
				Expect(instance2.IsRestorableCallCount()).To(Equal(1))
				Expect(isRestorable).To(BeTrue())
			})
		})
		Context("Multiple instances, none restorable", func() {
			BeforeEach(func() {
				instance1.IsRestorableReturns(false)
				instance2.IsRestorableReturns(false)
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("does not fail", func() {
				Expect(isRestorableError).NotTo(HaveOccurred())
			})

			It("returns true if any instance is restorable", func() {
				Expect(instance1.IsRestorableCallCount()).To(Equal(1))
				Expect(instance2.IsRestorableCallCount()).To(Equal(1))
				Expect(isRestorable).To(BeFalse())
			})
		})
	})

	Context("Cleanup", func() {
		var (
			actualCleanupError error
		)
		JustBeforeEach(func() {
			actualCleanupError = deployment.Cleanup()
		})

		Context("Single instance", func() {
			BeforeEach(func() {
				instances = []orchestrator.Instance{instance1}
			})
			It("does not fail", func() {
				Expect(actualCleanupError).NotTo(HaveOccurred())
			})
			It("cleans up the instance", func() {
				Expect(instance1.CleanupCallCount()).To(Equal(1))
			})
		})

		Context("Multiple instances", func() {
			BeforeEach(func() {
				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("does not fail", func() {
				Expect(actualCleanupError).NotTo(HaveOccurred())
			})
			It("cleanup all the instances", func() {
				Expect(instance1.CleanupCallCount()).To(Equal(1))
				Expect(instance2.CleanupCallCount()).To(Equal(1))
			})
		})

		Context("Multiple instances, some failing", func() {
			var cleanupError1 = fmt.Errorf("foo")

			BeforeEach(func() {
				instance1.CleanupReturns(cleanupError1)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("fails", func() {
				Expect(actualCleanupError).To(MatchError(ContainSubstring(cleanupError1.Error())))
			})

			It("continues cleanup of instances", func() {
				Expect(instance1.CleanupCallCount()).To(Equal(1))
				Expect(instance2.CleanupCallCount()).To(Equal(1))
			})
		})

		Context("Multiple instances, all failing", func() {
			var cleanupError1 = fmt.Errorf("foo")
			var cleanupError2 = fmt.Errorf("bar")
			BeforeEach(func() {
				instance1.CleanupReturns(cleanupError1)
				instance2.CleanupReturns(cleanupError2)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("fails with both error messages", func() {
				Expect(actualCleanupError).To(MatchError(ContainSubstring(cleanupError1.Error())))
				Expect(actualCleanupError).To(MatchError(ContainSubstring(cleanupError2.Error())))
			})

			It("continues cleanup of instances", func() {
				Expect(instance1.CleanupCallCount()).To(Equal(1))
				Expect(instance2.CleanupCallCount()).To(Equal(1))
			})
		})
	})

	Context("CleanupPrevious", func() {
		var (
			actualCleanupError error
		)
		JustBeforeEach(func() {
			actualCleanupError = deployment.CleanupPrevious()
		})

		Context("Single instance is backupable", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(true)
				instance1.IsRestorableReturns(false)
				instances = []orchestrator.Instance{instance1}
			})
			It("does not fail", func() {
				Expect(actualCleanupError).NotTo(HaveOccurred())
			})
			It("cleans up the instance", func() {
				Expect(instance1.CleanupPreviousCallCount()).To(Equal(1))
			})
		})
		Context("Single instance is restorable", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(false)
				instance1.IsRestorableReturns(true)
				instances = []orchestrator.Instance{instance1}
			})
			It("does not fail", func() {
				Expect(actualCleanupError).NotTo(HaveOccurred())
			})
			It("cleans up the instance", func() {
				Expect(instance1.CleanupPreviousCallCount()).To(Equal(1))
			})
		})
		Context("Single instance is neither backupable nor restorable", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(false)
				instance1.IsRestorableReturns(false)
				instances = []orchestrator.Instance{instance1}
			})
			It("does not fail", func() {
				Expect(actualCleanupError).NotTo(HaveOccurred())
			})
			It("does not clean up the instance", func() {
				Expect(instance1.CleanupPreviousCallCount()).To(Equal(0))
			})
		})

		Context("Multiple backupable instances", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(true)
				instance1.IsRestorableReturns(false)

				instance2.IsBackupableReturns(true)
				instance2.IsRestorableReturns(false)

				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("does not fail", func() {
				Expect(actualCleanupError).NotTo(HaveOccurred())
			})
			It("cleanup all the instances", func() {
				Expect(instance1.CleanupPreviousCallCount()).To(Equal(1))
				Expect(instance2.CleanupPreviousCallCount()).To(Equal(1))
			})
		})

		Context("Multiple restoable instances", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(false)
				instance1.IsRestorableReturns(true)

				instance2.IsBackupableReturns(true)
				instance2.IsRestorableReturns(true)

				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("does not fail", func() {
				Expect(actualCleanupError).NotTo(HaveOccurred())
			})
			It("cleanup all the instances", func() {
				Expect(instance1.CleanupPreviousCallCount()).To(Equal(1))
				Expect(instance2.CleanupPreviousCallCount()).To(Equal(1))
			})
		})

		Context("Multiple instances neither backupable nor restoable", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(false)
				instance1.IsRestorableReturns(false)

				instance2.IsBackupableReturns(false)
				instance2.IsRestorableReturns(false)

				instances = []orchestrator.Instance{instance1, instance2}
			})
			It("does not fail", func() {
				Expect(actualCleanupError).NotTo(HaveOccurred())
			})
			It("cleanup none of the instances", func() {
				Expect(instance1.CleanupPreviousCallCount()).To(Equal(0))
				Expect(instance2.CleanupPreviousCallCount()).To(Equal(0))
			})
		})

		Context("Multiple instances some backuable and restorable", func() {
			BeforeEach(func() {
				instance1.IsBackupableReturns(true)
				instance1.IsRestorableReturns(false)

				instance2.IsBackupableReturns(false)
				instance2.IsRestorableReturns(true)

				instance3.IsBackupableReturns(false)
				instance3.IsRestorableReturns(false)

				instances = []orchestrator.Instance{instance1, instance2, instance3}
			})
			It("does not fail", func() {
				Expect(actualCleanupError).NotTo(HaveOccurred())
			})
			It("cleanup backupable or restoreable the instances", func() {
				Expect(instance1.CleanupPreviousCallCount()).To(Equal(1))
				Expect(instance2.CleanupPreviousCallCount()).To(Equal(1))
				Expect(instance3.CleanupPreviousCallCount()).To(Equal(0))
			})
		})

		Context("Multiple instances, some failing", func() {
			var cleanupError1 = fmt.Errorf("foo")

			BeforeEach(func() {
				instance1.IsBackupableReturns(true)
				instance1.CleanupPreviousReturns(cleanupError1)
				instance2.IsBackupableReturns(true)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("fails", func() {
				Expect(actualCleanupError).To(MatchError(ContainSubstring(cleanupError1.Error())))
			})

			It("continues cleanup of instances", func() {
				Expect(instance1.CleanupPreviousCallCount()).To(Equal(1))
				Expect(instance2.CleanupPreviousCallCount()).To(Equal(1))
			})
		})

		Context("Multiple instances, all failing", func() {
			var cleanupError1 = fmt.Errorf("foo")
			var cleanupError2 = fmt.Errorf("bar")
			BeforeEach(func() {
				instance1.IsBackupableReturns(true)
				instance1.CleanupPreviousReturns(cleanupError1)
				instance2.IsRestorableReturns(true)
				instance2.CleanupPreviousReturns(cleanupError2)
				instances = []orchestrator.Instance{instance1, instance2}
			})

			It("fails with both error messages", func() {
				Expect(actualCleanupError).To(MatchError(ContainSubstring(cleanupError1.Error())))
				Expect(actualCleanupError).To(MatchError(ContainSubstring(cleanupError2.Error())))
			})

			It("continues cleanup of instances", func() {
				Expect(instance1.CleanupPreviousCallCount()).To(Equal(1))
				Expect(instance2.CleanupPreviousCallCount()).To(Equal(1))
			})
		})
	})

	Context("Instances", func() {
		BeforeEach(func() {
			instances = []orchestrator.Instance{instance1, instance2, instance3}
		})
		It("returns instances for the deployment", func() {
			Expect(deployment.Instances()).To(ConsistOf(instance1, instance2, instance3))
		})
	})

	Context("CopyRemoteBackupsToLocal", func() {
		var (
			artifact                              *fakes.FakeBackup
			backupArtifact                        *fakes.FakeBackupArtifact
			copyRemoteBackupsToLocalArtifactError error
		)
		BeforeEach(func() {
			artifact = new(fakes.FakeBackup)
			backupArtifact = new(fakes.FakeBackupArtifact)
		})
		JustBeforeEach(func() {
			copyRemoteBackupsToLocalArtifactError = deployment.CopyRemoteBackupToLocal(artifact)
		})

		Context("One instance, backupable", func() {
			var localArtifactWriteCloser *fakes.FakeWriteCloser
			var remoteArtifactChecksum = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}

			BeforeEach(func() {
				localArtifactWriteCloser = new(fakes.FakeWriteCloser)
				artifact.CreateArtifactReturns(localArtifactWriteCloser, nil)

				instance1.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{backupArtifact})
				instance1.IsBackupableReturns(true)
				artifact.CalculateChecksumReturns(remoteArtifactChecksum, nil)
				backupArtifact.ChecksumReturns(remoteArtifactChecksum, nil)

				instances = []orchestrator.Instance{instance1}
			})

			It("creates an artifact file with the instance", func() {
				Expect(artifact.CreateArtifactCallCount()).To(Equal(1))
				Expect(artifact.CreateArtifactArgsForCall(0)).To(Equal(backupArtifact))
			})

			It("streams the backup to the writer for the artifact file", func() {
				Expect(backupArtifact.StreamFromRemoteCallCount()).To(Equal(1))
				Expect(backupArtifact.StreamFromRemoteArgsForCall(0)).To(Equal(localArtifactWriteCloser))
			})

			It("closes the writer after its been streamed", func() {
				Expect(localArtifactWriteCloser.CloseCallCount()).To(Equal(1))
			})

			It("deletes the artifact on the remote", func() {
				Expect(backupArtifact.DeleteCallCount()).To(Equal(1))
			})

			It("calculates checksum for the artifact", func() {
				Expect(artifact.CalculateChecksumCallCount()).To(Equal(1))
				Expect(artifact.CalculateChecksumArgsForCall(0)).To(Equal(backupArtifact))
			})

			It("calculates checksum for the instance on remote", func() {
				Expect(backupArtifact.ChecksumCallCount()).To(Equal(1))
			})

			It("appends the checksum for the instance on the artifact", func() {
				Expect(artifact.AddChecksumCallCount()).To(Equal(1))
				actualRemoteArtifact, acutalChecksum := artifact.AddChecksumArgsForCall(0)
				Expect(actualRemoteArtifact).To(Equal(backupArtifact))
				Expect(acutalChecksum).To(Equal(remoteArtifactChecksum))
			})
		})

		Context("Many instances, backupable", func() {
			var instanceChecksum = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}
			var writeCloser1 *fakes.FakeWriteCloser
			var writeCloser2 *fakes.FakeWriteCloser

			var artifact1 *fakes.FakeBackupArtifact
			var artifact2 *fakes.FakeBackupArtifact

			BeforeEach(func() {
				writeCloser1 = new(fakes.FakeWriteCloser)
				writeCloser2 = new(fakes.FakeWriteCloser)
				artifact1 = new(fakes.FakeBackupArtifact)
				artifact2 = new(fakes.FakeBackupArtifact)

				artifact.CreateArtifactStub = func(i orchestrator.ArtifactIdentifier) (io.WriteCloser, error) {
					if i == artifact1 {
						return writeCloser1, nil
					} else {
						return writeCloser2, nil
					}
				}

				instance1.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{artifact1})
				instance2.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{artifact2})

				instance1.IsBackupableReturns(true)
				instance2.IsBackupableReturns(true)

				artifact.CalculateChecksumReturns(instanceChecksum, nil)

				instances = []orchestrator.Instance{instance1, instance2}
				artifact1.ChecksumReturns(instanceChecksum, nil)
				artifact2.ChecksumReturns(instanceChecksum, nil)
			})
			It("creates an artifact file with the instance", func() {
				Expect(artifact.CreateArtifactCallCount()).To(Equal(2))
				Expect(artifact.CreateArtifactArgsForCall(0)).To(Equal(artifact1))
				Expect(artifact.CreateArtifactArgsForCall(1)).To(Equal(artifact2))
			})

			It("streams the backup to the writer for the artifact file", func() {
				Expect(artifact1.StreamFromRemoteCallCount()).To(Equal(1))
				Expect(artifact1.StreamFromRemoteArgsForCall(0)).To(Equal(writeCloser1))

				Expect(artifact2.StreamFromRemoteCallCount()).To(Equal(1))
				Expect(artifact2.StreamFromRemoteArgsForCall(0)).To(Equal(writeCloser2))
			})

			It("closes the writer after its been streamed", func() {
				Expect(writeCloser1.CloseCallCount()).To(Equal(1))
				Expect(writeCloser2.CloseCallCount()).To(Equal(1))
			})

			It("calculates checksum for the instance on the artifact", func() {
				Expect(artifact.CalculateChecksumCallCount()).To(Equal(2))
				Expect(artifact.CalculateChecksumArgsForCall(0)).To(Equal(artifact1))
				Expect(artifact.CalculateChecksumArgsForCall(1)).To(Equal(artifact2))
			})

			It("calculates checksum for the instance on remote", func() {
				Expect(artifact1.ChecksumCallCount()).To(Equal(1))
				Expect(artifact2.ChecksumCallCount()).To(Equal(1))
			})

			It("deletes both the artifacts", func() {
				Expect(artifact1.DeleteCallCount()).To(Equal(1))
				Expect(artifact2.DeleteCallCount()).To(Equal(1))
			})

			It("appends the checksum for the instance on the artifact", func() {
				Expect(artifact.AddChecksumCallCount()).To(Equal(2))
				actualRemoteArtifact, acutalChecksum := artifact.AddChecksumArgsForCall(0)
				Expect(actualRemoteArtifact).To(Equal(artifact1))
				Expect(acutalChecksum).To(Equal(instanceChecksum))

				actualRemoteArtifact, acutalChecksum = artifact.AddChecksumArgsForCall(1)
				Expect(actualRemoteArtifact).To(Equal(artifact2))
				Expect(acutalChecksum).To(Equal(instanceChecksum))
			})
		})

		Context("Many instances, one backupable", func() {
			var instanceChecksum = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}
			var writeCloser1 *fakes.FakeWriteCloser
			var remoteArtifact1 *fakes.FakeBackupArtifact

			BeforeEach(func() {
				writeCloser1 = new(fakes.FakeWriteCloser)
				remoteArtifact1 = new(fakes.FakeBackupArtifact)

				artifact.CreateArtifactReturns(writeCloser1, nil)

				instance1.IsBackupableReturns(true)
				instance1.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{remoteArtifact1})

				instance2.IsBackupableReturns(false)
				artifact.CalculateChecksumReturns(instanceChecksum, nil)

				instances = []orchestrator.Instance{instance1, instance2}
				remoteArtifact1.ChecksumReturns(instanceChecksum, nil)
			})
			It("succeeds", func() {
				Expect(copyRemoteBackupsToLocalArtifactError).To(Succeed())
			})
			It("creates an artifact file with the instance", func() {
				Expect(artifact.CreateArtifactCallCount()).To(Equal(1))
				Expect(artifact.CreateArtifactArgsForCall(0)).To(Equal(remoteArtifact1))
			})

			It("streams the backup to the writer for the artifact file", func() {
				Expect(remoteArtifact1.StreamFromRemoteCallCount()).To(Equal(1))
				Expect(remoteArtifact1.StreamFromRemoteArgsForCall(0)).To(Equal(writeCloser1))
			})

			It("closes the writer after its been streamed", func() {
				Expect(writeCloser1.CloseCallCount()).To(Equal(1))
			})

			It("calculates checksum for the instance on the artifact", func() {
				Expect(artifact.CalculateChecksumCallCount()).To(Equal(1))
				Expect(artifact.CalculateChecksumArgsForCall(0)).To(Equal(remoteArtifact1))
			})

			It("calculates checksum for the instance on remote", func() {
				Expect(remoteArtifact1.ChecksumCallCount()).To(Equal(1))
			})

			It("appends the checksum for the instance on the artifact", func() {
				Expect(artifact.AddChecksumCallCount()).To(Equal(1))
				actualRemoteArtifact, acutalChecksum := artifact.AddChecksumArgsForCall(0)
				Expect(actualRemoteArtifact).To(Equal(remoteArtifact1))
				Expect(acutalChecksum).To(Equal(instanceChecksum))
			})
		})

		Describe("failures", func() {
			Context("fails if backup cannot be drained", func() {
				var drainError = fmt.Errorf("please make it stop")

				BeforeEach(func() {
					backupArtifact = new(fakes.FakeBackupArtifact)

					instances = []orchestrator.Instance{instance1}
					instance1.IsBackupableReturns(true)
					instance1.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{backupArtifact})

					backupArtifact.StreamFromRemoteReturns(drainError)
				})

				It("fails the transfer process", func() {
					Expect(copyRemoteBackupsToLocalArtifactError).To(MatchError(drainError))
				})
			})

			Context("fails if file cannot be created", func() {
				var fileError = fmt.Errorf("not a good file")
				BeforeEach(func() {
					instances = []orchestrator.Instance{instance1}
					instance1.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{backupArtifact})
					instance1.IsBackupableReturns(true)
					artifact.CreateArtifactReturns(nil, fileError)
				})

				It("fails the backup process", func() {
					Expect(copyRemoteBackupsToLocalArtifactError).To(MatchError(fileError))
				})
			})

			Context("fails if local shasum calculation fails", func() {
				shasumError := fmt.Errorf("yuuuge")
				var writeCloser1 *fakes.FakeWriteCloser

				BeforeEach(func() {
					writeCloser1 = new(fakes.FakeWriteCloser)
					instances = []orchestrator.Instance{instance1}
					instance1.IsBackupableReturns(true)
					instance1.BackupReturns(nil)
					instance1.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{backupArtifact})

					artifact.CreateArtifactReturns(writeCloser1, nil)
					artifact.CalculateChecksumReturns(nil, shasumError)
				})

				It("fails the backup process", func() {
					Expect(copyRemoteBackupsToLocalArtifactError).To(MatchError(shasumError))
				})
			})

			Context("fails if the remote shasum cant be calulated", func() {
				remoteShasumError := fmt.Errorf("this shasum is not happy")
				var writeCloser1 *fakes.FakeWriteCloser

				BeforeEach(func() {
					writeCloser1 = new(fakes.FakeWriteCloser)
					instances = []orchestrator.Instance{instance1}

					instance1.IsBackupableReturns(true)
					instance1.BackupReturns(nil)
					instance1.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{backupArtifact})
					backupArtifact.ChecksumReturns(nil, remoteShasumError)

					artifact.CreateArtifactReturns(writeCloser1, nil)
				})

				It("fails the backup process", func() {
					Expect(copyRemoteBackupsToLocalArtifactError).To(MatchError(remoteShasumError))
				})

				It("dosen't try to append shasum to metadata", func() {
					Expect(artifact.AddChecksumCallCount()).To(BeZero())
				})
			})

			Context("fails if the remote shasum dosen't match the local shasum", func() {
				var writeCloser1 *fakes.FakeWriteCloser

				BeforeEach(func() {
					writeCloser1 = new(fakes.FakeWriteCloser)
					instances = []orchestrator.Instance{instance1}

					instance1.IsBackupableReturns(true)
					instance1.BackupReturns(nil)
					instance1.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{backupArtifact})

					artifact.CreateArtifactReturns(writeCloser1, nil)

					artifact.CalculateChecksumReturns(orchestrator.BackupChecksum{"file": "this won't match"}, nil)
					backupArtifact.ChecksumReturns(orchestrator.BackupChecksum{"file": "this wont match"}, nil)
				})

				It("fails the backup process", func() {
					Expect(copyRemoteBackupsToLocalArtifactError).To(MatchError(ContainSubstring("Backup is corrupted")))
				})

				It("dosen't try to append shasum to metadata", func() {
					Expect(artifact.AddChecksumCallCount()).To(BeZero())
				})
			})

			Context("fails if the number of files in the artifact dont match", func() {
				var writeCloser1 *fakes.FakeWriteCloser

				BeforeEach(func() {
					writeCloser1 = new(fakes.FakeWriteCloser)
					instances = []orchestrator.Instance{instance1}

					instance1.IsBackupableReturns(true)
					instance1.BackupReturns(nil)
					instance1.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{backupArtifact})

					artifact.CreateArtifactReturns(writeCloser1, nil)
					artifact.CalculateChecksumReturns(orchestrator.BackupChecksum{"file": "this will match", "extra": "this won't match"}, nil)
					backupArtifact.ChecksumReturns(orchestrator.BackupChecksum{"file": "this will match"}, nil)
				})

				It("fails the backup process", func() {
					Expect(copyRemoteBackupsToLocalArtifactError).To(MatchError(ContainSubstring("Backup is corrupted")))
				})

				It("dosen't try to append shasum to metadata", func() {
					Expect(artifact.AddChecksumCallCount()).To(BeZero())
				})
			})

			Context("fails if unable to delete artifacts", func() {
				var writeCloser1 *fakes.FakeWriteCloser
				var instanceChecksum = orchestrator.BackupChecksum{"file1": "abcd", "file2": "efgh"}
				var expectedError = fmt.Errorf("brr")

				BeforeEach(func() {
					writeCloser1 = new(fakes.FakeWriteCloser)
					instances = []orchestrator.Instance{instance1}

					instance1.IsBackupableReturns(true)
					instance1.BackupReturns(nil)
					instance1.ArtifactsToBackupReturns([]orchestrator.BackupArtifact{backupArtifact})

					artifact.CreateArtifactReturns(writeCloser1, nil)
					artifact.CalculateChecksumReturns(orchestrator.BackupChecksum{"file": "this will match", "extra": "this won't match"}, nil)
					artifact.CalculateChecksumReturns(instanceChecksum, nil)
					backupArtifact.ChecksumReturns(instanceChecksum, nil)

					backupArtifact.DeleteReturns(expectedError)
				})

				It("fails the backup process", func() {
					Expect(copyRemoteBackupsToLocalArtifactError).To(MatchError(expectedError))
				})
			})
		})
	})
})
