package deployment_test

import (
	. "github.com/cloudfoundry/bosh-cli/deployment"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"

	mock_agentclient "github.com/cloudfoundry/bosh-cli/agentclient/mocks"
	mock_blobstore "github.com/cloudfoundry/bosh-cli/blobstore/mocks"
	mock_cloud "github.com/cloudfoundry/bosh-cli/cloud/mocks"
	mock_instance_state "github.com/cloudfoundry/bosh-cli/deployment/instance/state/mocks"
	"github.com/golang/mock/gomock"

	bias "github.com/cloudfoundry/bosh-agent/agentclient/applyspec"
	bicloud "github.com/cloudfoundry/bosh-cli/cloud"
	biconfig "github.com/cloudfoundry/bosh-cli/config"
	bidisk "github.com/cloudfoundry/bosh-cli/deployment/disk"
	biinstance "github.com/cloudfoundry/bosh-cli/deployment/instance"
	bisshtunnel "github.com/cloudfoundry/bosh-cli/deployment/sshtunnel"
	bivm "github.com/cloudfoundry/bosh-cli/deployment/vm"
	bistemcell "github.com/cloudfoundry/bosh-cli/stemcell"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"

	fakebiui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("Deployment", func() {
	var mockCtrl *gomock.Controller

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Delete", func() {
		var (
			logger boshlog.Logger
			fs     boshsys.FileSystem

			fakeUUIDGenerator      *fakeuuid.FakeGenerator
			fakeRepoUUIDGenerator  *fakeuuid.FakeGenerator
			deploymentStateService biconfig.DeploymentStateService
			vmRepo                 biconfig.VMRepo
			diskRepo               biconfig.DiskRepo
			stemcellRepo           biconfig.StemcellRepo

			mockCloud       *mock_cloud.MockCloud
			mockAgentClient *mock_agentclient.MockAgentClient

			mockStateBuilderFactory *mock_instance_state.MockBuilderFactory
			mockStateBuilder        *mock_instance_state.MockBuilder
			mockState               *mock_instance_state.MockState

			mockBlobstore *mock_blobstore.MockBlobstore

			fakeStage *fakebiui.FakeStage

			deploymentFactory Factory

			deployment Deployment
		)

		var expectNormalFlow = func() {
			gomock.InOrder(
				mockCloud.EXPECT().HasVM("fake-vm-cid").Return(true, nil),
				mockAgentClient.EXPECT().Ping().Return("any-state", nil),                   // ping to make sure agent is responsive
				mockAgentClient.EXPECT().Stop(),                                            // stop all jobs
				mockAgentClient.EXPECT().ListDisk().Return([]string{"fake-disk-cid"}, nil), // get mounted disks to be unmounted
				mockAgentClient.EXPECT().UnmountDisk("fake-disk-cid"),
				mockCloud.EXPECT().DeleteVM("fake-vm-cid"),
				mockCloud.EXPECT().DeleteDisk("fake-disk-cid"),
				mockCloud.EXPECT().DeleteStemcell("fake-stemcell-cid"),
			)
		}

		var allowApplySpecToBeCreated = func() {
			jobName := "fake-job-name"
			jobIndex := 0

			applySpec := bias.ApplySpec{
				Deployment: "test-release",
				Index:      jobIndex,
				Packages:   map[string]bias.Blob{},
				Networks: map[string]interface{}{
					"network-1": map[string]interface{}{
						"cloud_properties": map[string]interface{}{},
						"type":             "dynamic",
						"ip":               "",
					},
				},
				Job: bias.Job{
					Name:      jobName,
					Templates: []bias.Blob{},
				},
				RenderedTemplatesArchive: bias.RenderedTemplatesArchiveSpec{},
				ConfigurationHash:        "",
			}

			mockStateBuilderFactory.EXPECT().NewBuilder(mockBlobstore, mockAgentClient).Return(mockStateBuilder).AnyTimes()
			mockState.EXPECT().ToApplySpec().Return(applySpec).AnyTimes()
		}

		BeforeEach(func() {
			logger = boshlog.NewLogger(boshlog.LevelNone)
			fs = fakesys.NewFakeFileSystem()

			fakeUUIDGenerator = fakeuuid.NewFakeGenerator()
			deploymentStateService = biconfig.NewFileSystemDeploymentStateService(fs, fakeUUIDGenerator, logger, "/deployment.json")

			fakeRepoUUIDGenerator = fakeuuid.NewFakeGenerator()
			vmRepo = biconfig.NewVMRepo(deploymentStateService)
			diskRepo = biconfig.NewDiskRepo(deploymentStateService, fakeRepoUUIDGenerator)
			stemcellRepo = biconfig.NewStemcellRepo(deploymentStateService, fakeRepoUUIDGenerator)

			mockCloud = mock_cloud.NewMockCloud(mockCtrl)
			mockAgentClient = mock_agentclient.NewMockAgentClient(mockCtrl)

			fakeStage = fakebiui.NewFakeStage()

			pingTimeout := 10 * time.Second
			pingDelay := 500 * time.Millisecond
			deploymentFactory = NewFactory(pingTimeout, pingDelay)
		})

		JustBeforeEach(func() {
			// all these local factories & managers are just used to construct a Deployment based on the deployment state
			diskManagerFactory := bidisk.NewManagerFactory(diskRepo, logger)
			diskDeployer := bivm.NewDiskDeployer(diskManagerFactory, diskRepo, logger)

			vmManagerFactory := bivm.NewManagerFactory(vmRepo, stemcellRepo, diskDeployer, fakeUUIDGenerator, fs, logger)
			sshTunnelFactory := bisshtunnel.NewFactory(logger)

			mockStateBuilderFactory = mock_instance_state.NewMockBuilderFactory(mockCtrl)
			mockStateBuilder = mock_instance_state.NewMockBuilder(mockCtrl)
			mockState = mock_instance_state.NewMockState(mockCtrl)

			instanceFactory := biinstance.NewFactory(mockStateBuilderFactory)
			instanceManagerFactory := biinstance.NewManagerFactory(sshTunnelFactory, instanceFactory, logger)
			stemcellManagerFactory := bistemcell.NewManagerFactory(stemcellRepo)

			mockBlobstore = mock_blobstore.NewMockBlobstore(mockCtrl)

			deploymentManagerFactory := NewManagerFactory(vmManagerFactory, instanceManagerFactory, diskManagerFactory, stemcellManagerFactory, deploymentFactory)
			deploymentManager := deploymentManagerFactory.NewManager(mockCloud, mockAgentClient, mockBlobstore)

			allowApplySpecToBeCreated()

			var err error
			deployment, _, err = deploymentManager.FindCurrent()
			Expect(err).ToNot(HaveOccurred())
			//Note: deployment will be nil if the config has no vms, disks, or stemcells
		})

		Context("when the deployment has been deployed", func() {
			BeforeEach(func() {
				// create deployment manifest yaml file
				deploymentStateService.Save(biconfig.DeploymentState{
					DirectorID:        "fake-director-id",
					InstallationID:    "fake-installation-id",
					CurrentVMCID:      "fake-vm-cid",
					CurrentStemcellID: "fake-stemcell-guid",
					CurrentDiskID:     "fake-disk-guid",
					Disks: []biconfig.DiskRecord{
						{
							ID:   "fake-disk-guid",
							CID:  "fake-disk-cid",
							Size: 100,
						},
					},
					Stemcells: []biconfig.StemcellRecord{
						{
							ID:  "fake-stemcell-guid",
							CID: "fake-stemcell-cid",
						},
					},
				})
			})

			It("stops agent, unmounts disk, deletes vm, deletes disk, deletes stemcell", func() {
				expectNormalFlow()

				err := deployment.Delete(fakeStage)
				Expect(err).ToNot(HaveOccurred())
			})

			It("logs validation stages", func() {
				expectNormalFlow()

				err := deployment.Delete(fakeStage)
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeStage.PerformCalls).To(Equal([]*fakebiui.PerformCall{
					{Name: "Waiting for the agent on VM 'fake-vm-cid'"},
					{Name: "Stopping jobs on instance 'unknown/0'"},
					{Name: "Unmounting disk 'fake-disk-cid'"},
					{Name: "Deleting VM 'fake-vm-cid'"},
					{Name: "Deleting disk 'fake-disk-cid'"},
					{Name: "Deleting stemcell 'fake-stemcell-cid'"},
				}))
			})

			It("clears current vm, disk and stemcell", func() {
				expectNormalFlow()

				err := deployment.Delete(fakeStage)
				Expect(err).ToNot(HaveOccurred())

				_, found, err := vmRepo.FindCurrent()
				Expect(found).To(BeFalse(), "should be no current VM")

				_, found, err = diskRepo.FindCurrent()
				Expect(found).To(BeFalse(), "should be no current disk")

				diskRecords, err := diskRepo.All()
				Expect(err).ToNot(HaveOccurred())
				Expect(diskRecords).To(BeEmpty(), "expected no disk records")

				_, found, err = stemcellRepo.FindCurrent()
				Expect(found).To(BeFalse(), "should be no current stemcell")

				stemcellRecords, err := stemcellRepo.All()
				Expect(err).ToNot(HaveOccurred())
				Expect(stemcellRecords).To(BeEmpty(), "expected no stemcell records")
			})

			//TODO: It'd be nice to test recovering after agent was responsive, before timeout (hard to do with gomock)
			Context("when agent is unresponsive", func() {
				BeforeEach(func() {
					// reduce timout & delay to reduce test duration
					pingTimeout := 1 * time.Second
					pingDelay := 100 * time.Millisecond
					deploymentFactory = NewFactory(pingTimeout, pingDelay)
				})

				It("times out pinging agent, deletes vm, deletes disk, deletes stemcell", func() {
					gomock.InOrder(
						mockCloud.EXPECT().HasVM("fake-vm-cid").Return(true, nil),
						mockAgentClient.EXPECT().Ping().Return("", bosherr.Error("unresponsive agent")).AnyTimes(), // ping to make sure agent is responsive
						mockCloud.EXPECT().DeleteVM("fake-vm-cid"),
						mockCloud.EXPECT().DeleteDisk("fake-disk-cid"),
						mockCloud.EXPECT().DeleteStemcell("fake-stemcell-cid"),
					)

					err := deployment.Delete(fakeStage)
					Expect(err).ToNot(HaveOccurred())
				})
			})

			Context("and delete previously suceeded", func() {
				JustBeforeEach(func() {
					expectNormalFlow()

					err := deployment.Delete(fakeStage)
					Expect(err).ToNot(HaveOccurred())

					// reset event log recording
					fakeStage = fakebiui.NewFakeStage()
				})

				It("does not delete anything", func() {
					err := deployment.Delete(fakeStage)
					Expect(err).ToNot(HaveOccurred())

					Expect(fakeStage.PerformCalls).To(BeEmpty())
				})
			})
		})

		Context("when nothing has been deployed", func() {
			BeforeEach(func() {
				deploymentStateService.Save(biconfig.DeploymentState{})
			})

			JustBeforeEach(func() {
				// A previous JustBeforeEach uses FindCurrent to define deployment,
				// which would return a nil if the config is empty.
				// So we have to make a fake empty deployment to test it.
				deployment = deploymentFactory.NewDeployment([]biinstance.Instance{}, []bidisk.Disk{}, []bistemcell.CloudStemcell{})
			})

			It("does not delete anything", func() {
				err := deployment.Delete(fakeStage)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeStage.PerformCalls).To(BeEmpty())
			})
		})

		Context("when VM has been deployed", func() {
			var (
				expectHasVM *gomock.Call
			)
			BeforeEach(func() {
				deploymentStateService.Save(biconfig.DeploymentState{})
				vmRepo.UpdateCurrent("fake-vm-cid")

				expectHasVM = mockCloud.EXPECT().HasVM("fake-vm-cid").Return(true, nil)
			})

			It("stops the agent and deletes the VM", func() {
				gomock.InOrder(
					mockAgentClient.EXPECT().Ping().Return("any-state", nil),                   // ping to make sure agent is responsive
					mockAgentClient.EXPECT().Stop(),                                            // stop all jobs
					mockAgentClient.EXPECT().ListDisk().Return([]string{"fake-disk-cid"}, nil), // get mounted disks to be unmounted
					mockAgentClient.EXPECT().UnmountDisk("fake-disk-cid"),
					mockCloud.EXPECT().DeleteVM("fake-vm-cid"),
				)

				err := deployment.Delete(fakeStage)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("when VM has been deleted manually (outside of bosh)", func() {
				BeforeEach(func() {
					expectHasVM.Return(false, nil)
				})

				It("skips agent shutdown & deletes the VM (to ensure related resources are released by the CPI)", func() {
					mockCloud.EXPECT().DeleteVM("fake-vm-cid")

					err := deployment.Delete(fakeStage)
					Expect(err).ToNot(HaveOccurred())
				})

				It("ignores VMNotFound errors", func() {
					mockCloud.EXPECT().DeleteVM("fake-vm-cid").Return(bicloud.NewCPIError("delete_vm", bicloud.CmdError{
						Type:    bicloud.VMNotFoundError,
						Message: "fake-vm-not-found-message",
					}))

					err := deployment.Delete(fakeStage)
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

		Context("when a current disk exists", func() {
			BeforeEach(func() {
				deploymentStateService.Save(biconfig.DeploymentState{})
				diskRecord, err := diskRepo.Save("fake-disk-cid", 100, nil)
				Expect(err).ToNot(HaveOccurred())
				diskRepo.UpdateCurrent(diskRecord.ID)
			})

			It("deletes the disk", func() {
				mockCloud.EXPECT().DeleteDisk("fake-disk-cid")

				err := deployment.Delete(fakeStage)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("when current disk has been deleted manually (outside of bosh)", func() {
				It("deletes the disk (to ensure related resources are released by the CPI)", func() {
					mockCloud.EXPECT().DeleteDisk("fake-disk-cid")

					err := deployment.Delete(fakeStage)
					Expect(err).ToNot(HaveOccurred())
				})

				It("ignores DiskNotFound errors", func() {
					mockCloud.EXPECT().DeleteDisk("fake-disk-cid").Return(bicloud.NewCPIError("delete_disk", bicloud.CmdError{
						Type:    bicloud.DiskNotFoundError,
						Message: "fake-disk-not-found-message",
					}))

					err := deployment.Delete(fakeStage)
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

		Context("when a current stemcell exists", func() {
			BeforeEach(func() {
				deploymentStateService.Save(biconfig.DeploymentState{})
				stemcellRecord, err := stemcellRepo.Save("fake-stemcell-name", "fake-stemcell-version", "fake-stemcell-cid")
				Expect(err).ToNot(HaveOccurred())
				stemcellRepo.UpdateCurrent(stemcellRecord.ID)
			})

			It("deletes the stemcell", func() {
				mockCloud.EXPECT().DeleteStemcell("fake-stemcell-cid")

				err := deployment.Delete(fakeStage)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("when current stemcell has been deleted manually (outside of bosh)", func() {
				It("deletes the stemcell (to ensure related resources are released by the CPI)", func() {
					mockCloud.EXPECT().DeleteStemcell("fake-stemcell-cid")

					err := deployment.Delete(fakeStage)
					Expect(err).ToNot(HaveOccurred())
				})

				It("ignores StemcellNotFound errors", func() {
					mockCloud.EXPECT().DeleteStemcell("fake-stemcell-cid").Return(bicloud.NewCPIError("delete_stemcell", bicloud.CmdError{
						Type:    bicloud.StemcellNotFoundError,
						Message: "fake-stemcell-not-found-message",
					}))

					err := deployment.Delete(fakeStage)
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})
	})
})
