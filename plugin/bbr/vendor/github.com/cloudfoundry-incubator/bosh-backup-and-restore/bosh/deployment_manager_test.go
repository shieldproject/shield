package bosh_test

import (
	"fmt"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh/fakes"
	orchestrator_fakes "github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator/fakes"
)

var _ = Describe("DeploymentManager", func() {
	var boshClient *fakes.FakeBoshClient
	var logger *fakes.FakeLogger
	var deploymentName = "brownie"
	var fakeBackup *orchestrator_fakes.FakeBackup
	var manifest string

	var deploymentManager *bosh.DeploymentManager
	BeforeEach(func() {
		boshClient = new(fakes.FakeBoshClient)
		logger = new(fakes.FakeLogger)
	})
	JustBeforeEach(func() {
		deploymentManager = bosh.NewDeploymentManager(boshClient, logger, true)
	})

	Context("Find", func() {
		var findError error
		var deployment orchestrator.Deployment
		var instances []orchestrator.Instance
		BeforeEach(func() {
			instances = []orchestrator.Instance{new(orchestrator_fakes.FakeInstance)}
			boshClient.FindInstancesReturns(instances, nil)
		})
		JustBeforeEach(func() {
			deployment, findError = deploymentManager.Find(deploymentName)
		})
		It("asks the bosh director for instances", func() {
			Expect(boshClient.FindInstancesCallCount()).To(Equal(1))
			Expect(boshClient.FindInstancesArgsForCall(0)).To(Equal(deploymentName))
		})
		It("returns the deployment manager with instances", func() {
			Expect(deployment).To(Equal(orchestrator.NewDeployment(logger, instances)))
		})

		Context("error finding instances", func() {
			var expectedFindError = fmt.Errorf("a tuna sandwich")
			BeforeEach(func() {
				boshClient.FindInstancesReturns(nil, expectedFindError)
			})

			It("returns an error", func() {
				Expect(findError).To(MatchError(ContainSubstring("failed to find instances")))
			})
		})
	})

	Describe("SaveManifest", func() {
		var saveManifestError error

		Context("when downloading the manifest", func() {
			JustBeforeEach(func() {
				saveManifestError = deploymentManager.SaveManifest(deploymentName, fakeBackup)
			})

			Context("successfully saves the manifest", func() {
				BeforeEach(func() {
					fakeBackup = new(orchestrator_fakes.FakeBackup)
					manifest = "foo"
					boshClient.GetManifestReturns(manifest, nil)
				})

				It("asks the bosh director for the manifest", func() {
					Expect(boshClient.GetManifestCallCount()).To(Equal(1))
					Expect(boshClient.GetManifestArgsForCall(0)).To(Equal(deploymentName))
				})

				It("saves the manifest to the backup", func() {
					Expect(fakeBackup.SaveManifestCallCount()).To(Equal(1))
					Expect(fakeBackup.SaveManifestArgsForCall(0)).To(Equal(manifest))
				})
				It("should succeed", func() {
					Expect(saveManifestError).To(Succeed())
				})
			})

			Context("fails to fetch the manifest", func() {
				var manifestFetchError = fmt.Errorf("Boring error")
				BeforeEach(func() {
					fakeBackup = new(orchestrator_fakes.FakeBackup)
					boshClient.GetManifestReturns("", manifestFetchError)
				})

				It("asks the bosh director for the manifest", func() {
					Expect(boshClient.GetManifestCallCount()).To(Equal(1))
					Expect(boshClient.GetManifestArgsForCall(0)).To(Equal(deploymentName))
				})

				It("does not save the manifest to the backup", func() {
					Expect(fakeBackup.SaveManifestCallCount()).To(BeZero())
				})

				It("should fail", func() {
					Expect(saveManifestError).To(MatchError(ContainSubstring("failed to get manifest")))
				})
			})

			Context("fails to save the manifest", func() {
				var manifestSaveError = fmt.Errorf("Boring")

				BeforeEach(func() {
					fakeBackup = new(orchestrator_fakes.FakeBackup)
					boshClient.GetManifestReturns(manifest, nil)
					fakeBackup.SaveManifestReturns(manifestSaveError)
				})

				It("asks the bosh director for the manifest", func() {
					Expect(boshClient.GetManifestCallCount()).To(Equal(1))
					Expect(boshClient.GetManifestArgsForCall(0)).To(Equal(deploymentName))
				})

				It("saves the manifest to the backup", func() {
					Expect(fakeBackup.SaveManifestCallCount()).To(Equal(1))
					Expect(fakeBackup.SaveManifestArgsForCall(0)).To(Equal(manifest))
				})

				It("should fail", func() {
					Expect(saveManifestError).To(MatchError(manifestSaveError))
				})
			})
		})

		Context("when not downloading the manifest", func() {
			BeforeEach(func() {
				fakeBackup = new(orchestrator_fakes.FakeBackup)
			})

			JustBeforeEach(func() {
				deploymentManager = bosh.NewDeploymentManager(boshClient, logger, false)
				saveManifestError = deploymentManager.SaveManifest(deploymentName, fakeBackup)
			})

			It("doesn't download the manifest", func() {
				Expect(boshClient.GetManifestCallCount()).To(BeZero())
				Expect(fakeBackup.SaveManifestCallCount()).To(BeZero())
			})
		})
	})
})
