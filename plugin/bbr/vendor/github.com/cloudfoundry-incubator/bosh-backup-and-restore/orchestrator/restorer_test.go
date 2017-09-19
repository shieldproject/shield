package orchestrator_test

import (
	"errors"
	"fmt"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("restorer", func() {
	Context("restores a deployment from backup", func() {
		var (
			restoreError      orchestrator.Error
			artifactManager   *fakes.FakeBackupManager
			artifact          *fakes.FakeBackup
			logger            *fakes.FakeLogger
			instances         []orchestrator.Instance
			b                 *orchestrator.Restorer
			deploymentName    string
			deploymentManager *fakes.FakeDeploymentManager
			deployment        *fakes.FakeDeployment
			artifactPath      string
		)

		BeforeEach(func() {
			instances = []orchestrator.Instance{new(fakes.FakeInstance)}
			logger = new(fakes.FakeLogger)
			artifactManager = new(fakes.FakeBackupManager)
			artifact = new(fakes.FakeBackup)
			deploymentManager = new(fakes.FakeDeploymentManager)
			deployment = new(fakes.FakeDeployment)

			artifactManager.OpenReturns(artifact, nil)
			deploymentManager.FindReturns(deployment, nil)
			deployment.IsRestorableReturns(true)
			deployment.InstancesReturns(instances)
			artifact.DeploymentMatchesReturns(true, nil)
			artifact.ValidReturns(true, nil)

			b = orchestrator.NewRestorer(artifactManager, logger, deploymentManager)

			deploymentName = "deployment-to-restore"
			artifactPath = "/some/path"
		})

		JustBeforeEach(func() {
			restoreError = b.Restore(deploymentName, artifactPath)
		})

		It("does not fail", func() {
			Expect(restoreError).NotTo(HaveOccurred())
		})

		It("ensures that instance is cleaned up", func() {
			Expect(deployment.CleanupCallCount()).To(Equal(1))
		})

		It("ensures artifact is valid", func() {
			Expect(artifact.ValidCallCount()).To(Equal(1))
		})

		It("opens the artifact", func() {
			Expect(artifactManager.OpenCallCount()).To(Equal(1))
			openedArtifactName, _ := artifactManager.OpenArgsForCall(0)
			Expect(openedArtifactName).To(Equal(artifactPath))
		})

		It("finds the deployment", func() {
			Expect(deploymentManager.FindCallCount()).To(Equal(1))
			Expect(deploymentManager.FindArgsForCall(0)).To(Equal(deploymentName))
		})

		It("checks if the deployment is restorable", func() {
			Expect(deployment.IsRestorableCallCount()).To(Equal(1))
		})

		It("checks that the deployment topology matches the topology of the backup", func() {
			Expect(artifactManager.OpenCallCount()).To(Equal(1))
			Expect(artifact.DeploymentMatchesCallCount()).To(Equal(1))

			name, actualInstances := artifact.DeploymentMatchesArgsForCall(0)
			Expect(name).To(Equal(deploymentName))
			Expect(actualInstances).To(Equal(instances))
		})

		It("streams the local backup to the deployment", func() {
			Expect(deployment.CopyLocalBackupToRemoteCallCount()).To(Equal(1))
			Expect(deployment.CopyLocalBackupToRemoteArgsForCall(0)).To(Equal(artifact))
		})

		It("calls restore on the deployment", func() {
			Expect(deployment.RestoreCallCount()).To(Equal(1))
		})

		It("calls post-restore-unlock on the deployment", func() {
			Expect(deployment.PostRestoreUnlockCallCount()).To(Equal(1))
		})

		Describe("failures", func() {

			var assertCleanupError = func() {
				var cleanupError = fmt.Errorf("too dirty")
				BeforeEach(func() {
					deployment.CleanupReturns(cleanupError)
				})

				It("includes the cleanup error in the returned error", func() {
					Expect(restoreError).To(MatchError(ContainSubstring(cleanupError.Error())))
				})
			}

			Context("fails to find deployment", func() {
				BeforeEach(func() {
					deploymentManager.FindReturns(nil, fmt.Errorf("no deployment here"))
				})

				It("returns an error", func() {
					Expect(restoreError).To(MatchError(ContainSubstring("no deployment here")))
				})
			})

			Context("fails if the artifact cant be opened", func() {
				var artifactOpenError = "I can't open this"
				BeforeEach(func() {
					deploymentManager.FindReturns(deployment, nil)
					artifactManager.OpenReturns(nil, errors.New(artifactOpenError))
				})
				It("returns an error", func() {
					Expect(restoreError).To(MatchError(ContainSubstring(artifactOpenError)))
				})
			})

			Context("fails if the artifact is invalid", func() {
				BeforeEach(func() {
					deploymentManager.FindReturns(deployment, nil)
					artifactManager.OpenReturns(artifact, nil)
					artifact.ValidReturns(false, nil)
				})
				It("returns an error", func() {
					Expect(restoreError).To(MatchError(ContainSubstring("Backup is corrupted")))
				})
			})

			Context("fails, if the cleanup fails", func() {
				var cleanupError = fmt.Errorf("still too dirty")
				BeforeEach(func() {
					deploymentManager.FindReturns(deployment, nil)
					artifactManager.OpenReturns(artifact, nil)
					artifact.ValidReturns(true, nil)
					deployment.CleanupReturns(cleanupError)
				})

				It("returns an error", func() {
					Expect(restoreError).To(MatchError(ContainSubstring(cleanupError.Error())))
				})

				It("returns an error of type, cleanup error", func() {
					Expect(restoreError[0]).To(BeAssignableToTypeOf(orchestrator.CleanupError{}))
				})
			})

			Context("fails if can't check if artifact is valid", func() {
				var artifactValidError = "I don't like this artifact"

				BeforeEach(func() {
					deploymentManager.FindReturns(deployment, nil)
					artifactManager.OpenReturns(artifact, nil)
					artifact.ValidReturns(false, errors.New(artifactValidError))
				})
				It("returns an error", func() {
					Expect(restoreError).To(MatchError(ContainSubstring(artifactValidError)))
				})
			})

			Context("if deployment not restorable", func() {
				BeforeEach(func() {
					deployment.IsRestorableReturns(false)
				})

				It("returns an error", func() {
					Expect(restoreError).To(HaveOccurred())
				})

				It("should cleanup", func() {
					Expect(deployment.CleanupCallCount()).To(Equal(1))
				})
				assertCleanupError()
			})

			Context("if the deployment's topology doesn't match that of the backup", func() {
				BeforeEach(func() {
					artifact.DeploymentMatchesReturns(false, nil)
				})

				It("returns an error", func() {
					Expect(restoreError).To(HaveOccurred())
				})

				It("should cleanup", func() {
					Expect(deployment.CleanupCallCount()).To(Equal(1))
				})
				assertCleanupError()
			})

			Context("if checking the deployment topology fails", func() {
				BeforeEach(func() {
					artifact.DeploymentMatchesReturns(true, fmt.Errorf("I am not the same"))
				})

				It("returns an error", func() {
					Expect(restoreError).To(HaveOccurred())
				})

				It("should cleanup", func() {
					Expect(deployment.CleanupCallCount()).To(Equal(1))
				})
				assertCleanupError()
			})

			Context("if a backup artifact already exists on any of the instances", func() {
				BeforeEach(func() {
					deployment.CheckArtifactDirReturns(fmt.Errorf("this is a problem"))
				})

				It("returns an error with the name of the instance with the extant backup artifact", func() {
					Expect(restoreError).To(MatchError(ContainSubstring("this is a problem")))
				})

				It("cleans up", func() {
					Expect(deployment.CleanupCallCount()).To(Equal(1))
				})
				assertCleanupError()
			})

			Context("if streaming the backup to the remote fails", func() {
				BeforeEach(func() {
					deployment.CopyLocalBackupToRemoteReturns(fmt.Errorf("Broken pipe"))
				})

				It("returns an error", func() {
					Expect(restoreError).To(MatchError(ContainSubstring("Unable to send backup to remote machine. Got error: Broken pipe")))
				})

				It("should cleanup", func() {
					Expect(deployment.CleanupCallCount()).To(Equal(1))
				})

				assertCleanupError()
			})

			Context("if post-restore-unlock fails", func() {
				var expectedPostRestoreUnlockError = fmt.Errorf("I will not restart this thing")

				BeforeEach(func() {
					deployment.PostRestoreUnlockReturns(expectedPostRestoreUnlockError)
				})

				It("returns an error", func() {
					Expect(restoreError).To(MatchError(ContainSubstring(expectedPostRestoreUnlockError.Error())))
				})

				It("should cleanup", func() {
					Expect(deployment.CleanupCallCount()).To(Equal(1))
				})
			})

			Context("if running the restore script fails", func() {
				BeforeEach(func() {
					deployment.RestoreReturns(fmt.Errorf("I will not restore this thing"))
				})

				It("returns an error", func() {
					Expect(restoreError).To(MatchError(ContainSubstring("Failed to restore: I will not restore this thing")))
				})

				It("should cleanup", func() {
					Expect(deployment.CleanupCallCount()).To(Equal(1))
				})

				It("calls post-restore-unlock on the deployment", func() {
					Expect(deployment.PostRestoreUnlockCallCount()).To(Equal(1))
				})

				Context("if post-restore unlock fails", func() {
					BeforeEach(func() {
						deployment.PostRestoreUnlockReturns(fmt.Errorf("I will not restart this thing"))
					})

					It("returns both errors", func() {
						Expect(restoreError).To(MatchError(ContainSubstring("post-restore-unlock failed:")))
						Expect(restoreError).To(MatchError(ContainSubstring("I will not restart this thing")))
						Expect(restoreError).To(MatchError(ContainSubstring("I will not restore this thing")))
					})

					assertCleanupError()
				})

				assertCleanupError()
			})
		})
	})
})
