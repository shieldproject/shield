package instance_test

import (
	. "github.com/cloudfoundry/bosh-cli/deployment/instance"

	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	mock_agentclient "github.com/cloudfoundry/bosh-cli/agentclient/mocks"
	mock_blobstore "github.com/cloudfoundry/bosh-cli/blobstore/mocks"
	mock_instance_state "github.com/cloudfoundry/bosh-cli/deployment/instance/state/mocks"
	"github.com/golang/mock/gomock"

	bias "github.com/cloudfoundry/bosh-agent/agentclient/applyspec"
	bidisk "github.com/cloudfoundry/bosh-cli/deployment/disk"
	bideplmanifest "github.com/cloudfoundry/bosh-cli/deployment/manifest"
	bisshtunnel "github.com/cloudfoundry/bosh-cli/deployment/sshtunnel"
	biinstallmanifest "github.com/cloudfoundry/bosh-cli/installation/manifest"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"

	"github.com/cloudfoundry/bosh-agent/agentclient"
	fakebicloud "github.com/cloudfoundry/bosh-cli/cloud/fakes"
	fakebidisk "github.com/cloudfoundry/bosh-cli/deployment/disk/fakes"
	fakebisshtunnel "github.com/cloudfoundry/bosh-cli/deployment/sshtunnel/fakes"
	fakebivm "github.com/cloudfoundry/bosh-cli/deployment/vm/fakes"
	fakebistemcell "github.com/cloudfoundry/bosh-cli/stemcell/stemcellfakes"
	fakebiui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("Manager", func() {
	var mockCtrl *gomock.Controller

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	var (
		fakeCloud *fakebicloud.FakeCloud

		mockStateBuilderFactory *mock_instance_state.MockBuilderFactory
		mockStateBuilder        *mock_instance_state.MockBuilder
		mockState               *mock_instance_state.MockState

		mockBlobstore *mock_blobstore.MockBlobstore

		fakeVMManager        *fakebivm.FakeManager
		fakeSSHTunnelFactory *fakebisshtunnel.FakeFactory
		fakeSSHTunnel        *fakebisshtunnel.FakeTunnel
		instanceFactory      Factory
		logger               boshlog.Logger
		fakeStage            *fakebiui.FakeStage

		manager Manager
	)

	BeforeEach(func() {
		fakeCloud = fakebicloud.NewFakeCloud()

		fakeVMManager = fakebivm.NewFakeManager()

		fakeSSHTunnelFactory = fakebisshtunnel.NewFakeFactory()
		fakeSSHTunnel = fakebisshtunnel.NewFakeTunnel()
		fakeSSHTunnel.SetStartBehavior(nil, nil)
		fakeSSHTunnelFactory.SSHTunnel = fakeSSHTunnel

		mockStateBuilderFactory = mock_instance_state.NewMockBuilderFactory(mockCtrl)
		mockStateBuilder = mock_instance_state.NewMockBuilder(mockCtrl)
		mockState = mock_instance_state.NewMockState(mockCtrl)

		instanceFactory = NewFactory(mockStateBuilderFactory)

		mockBlobstore = mock_blobstore.NewMockBlobstore(mockCtrl)

		logger = boshlog.NewLogger(boshlog.LevelNone)

		fakeStage = fakebiui.NewFakeStage()

		manager = NewManager(
			fakeCloud,
			fakeVMManager,
			mockBlobstore,
			fakeSSHTunnelFactory,
			instanceFactory,
			logger,
		)
	})

	Describe("Create", func() {
		var (
			mockAgentClient    *mock_agentclient.MockAgentClient
			fakeVM             *fakebivm.FakeVM
			diskPool           bideplmanifest.DiskPool
			deploymentManifest bideplmanifest.Manifest
			fakeCloudStemcell  *fakebistemcell.FakeCloudStemcell
			registry           biinstallmanifest.Registry

			expectedInstance Instance
			expectedDisk     *fakebidisk.FakeDisk
		)

		var allowApplySpecToBeCreated = func() {
			jobName := "cpi"
			jobIndex := 0

			applySpec := bias.ApplySpec{
				Deployment: "test-release",
				Index:      jobIndex,
				Packages:   map[string]bias.Blob{},
				Networks: map[string]interface{}{
					"network-1": biproperty.Map{
						"cloud_properties": biproperty.Map{},
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

			fakeAgentState := agentclient.AgentState{}
			fakeVM.GetStateResult = fakeAgentState

			mockStateBuilderFactory.EXPECT().NewBuilder(mockBlobstore, mockAgentClient).Return(mockStateBuilder).AnyTimes()
			mockStateBuilder.EXPECT().Build(jobName, jobIndex, deploymentManifest, fakeStage, fakeAgentState).Return(mockState, nil).AnyTimes()
			mockState.EXPECT().ToApplySpec().Return(applySpec).AnyTimes()
		}

		BeforeEach(func() {
			diskPool = bideplmanifest.DiskPool{
				Name:     "fake-persistent-disk-pool-name",
				DiskSize: 1024,
				CloudProperties: biproperty.Map{
					"fake-disk-pool-cloud-property-key": "fake-disk-pool-cloud-property-value",
				},
			}

			deploymentManifest = bideplmanifest.Manifest{
				Update: bideplmanifest.Update{
					UpdateWatchTime: bideplmanifest.WatchTime{
						Start: 0,
						End:   5478,
					},
				},
				DiskPools: []bideplmanifest.DiskPool{
					diskPool,
				},
				Jobs: []bideplmanifest.Job{
					{
						Name:               "fake-job-name",
						PersistentDiskPool: "fake-persistent-disk-pool-name",
						Instances:          1,
					},
				},
			}

			fakeCloudStemcell = fakebistemcell.NewFakeCloudStemcell("fake-stemcell-cid", "fake-stemcell-name", "fake-stemcell-version")
			registry = biinstallmanifest.Registry{}

			fakeVM = fakebivm.NewFakeVM("fake-vm-cid")
			fakeVMManager.CreateVM = fakeVM

			mockAgentClient = mock_agentclient.NewMockAgentClient(mockCtrl)
			fakeVM.AgentClientReturn = mockAgentClient

			expectedInstance = NewInstance(
				"fake-job-name",
				0,
				fakeVM,
				fakeVMManager,
				fakeSSHTunnelFactory,
				mockStateBuilder,
				logger,
			)

			expectedDisk = fakebidisk.NewFakeDisk("fake-disk-cid")
			fakeVM.UpdateDisksDisks = []bidisk.Disk{expectedDisk}
		})

		JustBeforeEach(func() {
			allowApplySpecToBeCreated()
		})

		It("returns an Instance that wraps a newly created VM", func() {
			instance, _, err := manager.Create(
				"fake-job-name",
				0,
				deploymentManifest,
				fakeCloudStemcell,
				registry,
				fakeStage,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(instance).To(Equal(expectedInstance))

			Expect(fakeVMManager.CreateInput).To(Equal(fakebivm.CreateInput{
				Stemcell: fakeCloudStemcell,
				Manifest: deploymentManifest,
			}))
		})

		It("updates the current stemcell", func() {
			_, _, err := manager.Create(
				"fake-job-name",
				0,
				deploymentManifest,
				fakeCloudStemcell,
				registry,
				fakeStage,
			)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeCloudStemcell.PromoteAsCurrentCalledTimes).To(Equal(1))
		})

		It("logs instance update stages", func() {
			_, _, err := manager.Create(
				"fake-job-name",
				0,
				deploymentManifest,
				fakeCloudStemcell,
				registry,
				fakeStage,
			)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeStage.PerformCalls).To(Equal([]*fakebiui.PerformCall{
				{Name: "Creating VM for instance 'fake-job-name/0' from stemcell 'fake-stemcell-cid'"},
				{Name: "Waiting for the agent on VM 'fake-vm-cid' to be ready"},
			}))
		})

		Context("when registry settings are empty", func() {
			BeforeEach(func() {
				registry = biinstallmanifest.Registry{}
			})

			It("does not start the registry", func() {
				_, _, err := manager.Create(
					"fake-job-name",
					0,
					deploymentManifest,
					fakeCloudStemcell,
					registry,
					fakeStage,
				)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		It("waits for the vm", func() {
			_, _, err := manager.Create(
				"fake-job-name",
				0,
				deploymentManifest,
				fakeCloudStemcell,
				registry,
				fakeStage,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeVM.WaitUntilReadyInputs).To(Equal([]fakebivm.WaitUntilReadyInput{
				{
					Timeout: 10 * time.Minute,
					Delay:   500 * time.Millisecond,
				},
			}))

			Expect(fakeStage.PerformCalls).To(Equal([]*fakebiui.PerformCall{
				{Name: "Creating VM for instance 'fake-job-name/0' from stemcell 'fake-stemcell-cid'"},
				{Name: "Waiting for the agent on VM 'fake-vm-cid' to be ready"},
			}))
		})

		It("returns the 'updated' disks", func() {
			_, disks, err := manager.Create(
				"fake-job-name",
				0,
				deploymentManifest,
				fakeCloudStemcell,
				registry,
				fakeStage,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(disks).To(Equal([]bidisk.Disk{expectedDisk}))

			Expect(fakeVM.UpdateDisksInputs).To(Equal([]fakebivm.UpdateDisksInput{
				{
					DiskPool: diskPool,
					Stage:    fakeStage,
				},
			}))
		})

		Context("when registry or sshTunnelConfig are not empty", func() {
			BeforeEach(func() {
				registry = biinstallmanifest.Registry{
					Username: "fake-registry-username",
					Password: "fake-registry-password",
					Host:     "fake-registry-host",
					Port:     124,
					SSHTunnel: biinstallmanifest.SSHTunnel{
						User:       "fake-ssh-user",
						Host:       "fake-ssh-host",
						Port:       123,
						Password:   "fake-ssh-password",
						PrivateKey: "---BEGIN PRIVATE KEY--- im a real key ---END PRIVATE KEY---",
					},
				}
			})

			It("starts & stops the ssh tunnel", func() {
				_, _, err := manager.Create(
					"fake-job-name",
					0,
					deploymentManifest,
					fakeCloudStemcell,
					registry,
					fakeStage,
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeSSHTunnelFactory.NewSSHTunnelOptions).To(Equal(bisshtunnel.Options{
					User:              "fake-ssh-user",
					Host:              "fake-ssh-host",
					Port:              123,
					Password:          "fake-ssh-password",
					PrivateKey:        "---BEGIN PRIVATE KEY--- im a real key ---END PRIVATE KEY---",
					LocalForwardPort:  124,
					RemoteForwardPort: 124,
				}))
				Expect(fakeSSHTunnel.Started).To(BeTrue())
			})

			Context("when starting the ssh tunnel fails", func() {
				BeforeEach(func() {
					fakeSSHTunnel.SetStartBehavior(errors.New("fake-ssh-tunnel-start-error"), nil)
				})

				It("returns an error", func() {
					_, _, err := manager.Create(
						"fake-job-name",
						0,
						deploymentManifest,
						fakeCloudStemcell,
						registry,
						fakeStage,
					)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-ssh-tunnel-start-error"))
				})
			})
		})

		Context("when ssh tunnel conifg is empty", func() {
			It("does not start the ssh tunnel", func() {
				_, _, err := manager.Create(
					"fake-job-name",
					0,
					deploymentManifest,
					fakeCloudStemcell,
					registry,
					fakeStage,
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeSSHTunnel.Started).To(BeFalse())
			})
		})

		Context("when creating VM fails", func() {
			BeforeEach(func() {
				fakeVMManager.CreateErr = errors.New("fake-create-vm-error")
			})

			It("returns an error", func() {
				_, _, err := manager.Create(
					"fake-job-name",
					0,
					deploymentManifest,
					fakeCloudStemcell,
					registry,
					fakeStage,
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-create-vm-error"))
			})

			It("logs start and stop events to the eventLogger", func() {
				_, _, err := manager.Create(
					"fake-job-name",
					0,
					deploymentManifest,
					fakeCloudStemcell,
					registry,
					fakeStage,
				)
				Expect(err).To(HaveOccurred())

				Expect(fakeStage.PerformCalls[0].Name).To(Equal("Creating VM for instance 'fake-job-name/0' from stemcell 'fake-stemcell-cid'"))
				Expect(fakeStage.PerformCalls[0].Error).To(HaveOccurred())
				Expect(fakeStage.PerformCalls[0].Error.Error()).To(Equal("Creating VM: fake-create-vm-error"))
			})
		})
	})
})
