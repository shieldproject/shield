package deployment_test

import (
	"errors"
	"time"

	. "github.com/cloudfoundry/bosh-cli/deployment"

	mock_httpagent "github.com/cloudfoundry/bosh-agent/agentclient/http/mocks"
	mock_agentclient "github.com/cloudfoundry/bosh-cli/agentclient/mocks"
	mock_blobstore "github.com/cloudfoundry/bosh-cli/blobstore/mocks"
	mock_instance_state "github.com/cloudfoundry/bosh-cli/deployment/instance/state/mocks"
	mock_vm "github.com/cloudfoundry/bosh-cli/deployment/vm/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	bias "github.com/cloudfoundry/bosh-agent/agentclient/applyspec"
	biconfig "github.com/cloudfoundry/bosh-cli/config"
	biinstance "github.com/cloudfoundry/bosh-cli/deployment/instance"
	bideplmanifest "github.com/cloudfoundry/bosh-cli/deployment/manifest"
	bisshtunnel "github.com/cloudfoundry/bosh-cli/deployment/sshtunnel"
	biinstallmanifest "github.com/cloudfoundry/bosh-cli/installation/manifest"
	bistemcell "github.com/cloudfoundry/bosh-cli/stemcell"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"

	"github.com/cloudfoundry/bosh-agent/agentclient"
	fakebicloud "github.com/cloudfoundry/bosh-cli/cloud/fakes"
	fakebiconfig "github.com/cloudfoundry/bosh-cli/config/fakes"
	fakebisshtunnel "github.com/cloudfoundry/bosh-cli/deployment/sshtunnel/fakes"
	fakebivm "github.com/cloudfoundry/bosh-cli/deployment/vm/fakes"
	fakebiui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("Deployer", func() {
	var mockCtrl *gomock.Controller

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	var (
		deployer               Deployer
		mockVMManagerFactory   *mock_vm.MockManagerFactory
		fakeVMManager          *fakebivm.FakeManager
		mockAgentClient        *mock_agentclient.MockAgentClient
		mockAgentClientFactory *mock_httpagent.MockAgentClientFactory
		fakeSSHTunnelFactory   *fakebisshtunnel.FakeFactory
		fakeSSHTunnel          *fakebisshtunnel.FakeTunnel
		cloud                  *fakebicloud.FakeCloud
		deploymentManifest     bideplmanifest.Manifest
		diskPool               bideplmanifest.DiskPool
		registryConfig         biinstallmanifest.Registry
		fakeStage              *fakebiui.FakeStage
		fakeVM                 *fakebivm.FakeVM

		cloudStemcell bistemcell.CloudStemcell

		applySpec bias.ApplySpec

		mockStateBuilderFactory *mock_instance_state.MockBuilderFactory
		mockStateBuilder        *mock_instance_state.MockBuilder
		mockState               *mock_instance_state.MockState

		mockBlobstore *mock_blobstore.MockBlobstore
	)

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
		registryConfig = biinstallmanifest.Registry{
			SSHTunnel: biinstallmanifest.SSHTunnel{
				User:       "fake-ssh-username",
				PrivateKey: "---BEGIN PRIVATE KEY--- qwerty ---END PRIVATE KEY---",
				Password:   "fake-password",
				Host:       "fake-ssh-host",
				Port:       124,
			},
		}

		cloud = fakebicloud.NewFakeCloud()

		mockAgentClientFactory = mock_httpagent.NewMockAgentClientFactory(mockCtrl)
		mockAgentClient = mock_agentclient.NewMockAgentClient(mockCtrl)
		mockAgentClientFactory.EXPECT().NewAgentClient(gomock.Any(), gomock.Any()).Return(mockAgentClient).AnyTimes()

		mockVMManagerFactory = mock_vm.NewMockManagerFactory(mockCtrl)
		fakeVMManager = fakebivm.NewFakeManager()
		mockVMManagerFactory.EXPECT().NewManager(cloud, mockAgentClient).Return(fakeVMManager).AnyTimes()

		fakeSSHTunnelFactory = fakebisshtunnel.NewFakeFactory()
		fakeSSHTunnel = fakebisshtunnel.NewFakeTunnel()
		fakeSSHTunnelFactory.SSHTunnel = fakeSSHTunnel
		fakeSSHTunnel.SetStartBehavior(nil, nil)

		fakeVM = fakebivm.NewFakeVM("fake-vm-cid")
		fakeVMManager.CreateVM = fakeVM

		fakeVM.AgentClientReturn = mockAgentClient

		logger := boshlog.NewLogger(boshlog.LevelNone)
		fakeStage = fakebiui.NewFakeStage()

		fakeStemcellRepo := fakebiconfig.NewFakeStemcellRepo()
		stemcellRecord := biconfig.StemcellRecord{
			ID:      "fake-stemcell-id",
			Name:    "fake-stemcell-name",
			Version: "fake-stemcell-version",
			CID:     "fake-stemcell-cid",
		}
		err := fakeStemcellRepo.SetFindBehavior("fake-stemcell-name", "fake-stemcell-version", stemcellRecord, true, nil)
		Expect(err).ToNot(HaveOccurred())

		cloudStemcell = bistemcell.NewCloudStemcell(stemcellRecord, fakeStemcellRepo, cloud)

		mockStateBuilderFactory = mock_instance_state.NewMockBuilderFactory(mockCtrl)
		mockStateBuilder = mock_instance_state.NewMockBuilder(mockCtrl)
		mockState = mock_instance_state.NewMockState(mockCtrl)

		instanceFactory := biinstance.NewFactory(mockStateBuilderFactory)
		instanceManagerFactory := biinstance.NewManagerFactory(fakeSSHTunnelFactory, instanceFactory, logger)

		mockBlobstore = mock_blobstore.NewMockBlobstore(mockCtrl)

		pingTimeout := 10 * time.Second
		pingDelay := 500 * time.Millisecond
		deploymentFactory := NewFactory(pingTimeout, pingDelay)

		deployer = NewDeployer(
			mockVMManagerFactory,
			instanceManagerFactory,
			deploymentFactory,
			logger,
		)
	})

	JustBeforeEach(func() {
		jobName := "fake-job-name"
		jobIndex := 0

		// since we're just passing this from State.ToApplySpec() to VM.Apply(), it doesn't need to be filled out
		applySpec = bias.ApplySpec{
			Deployment: "fake-deployment-name",
		}

		fakeAgentState := agentclient.AgentState{}
		fakeVM.GetStateResult = fakeAgentState

		mockStateBuilderFactory.EXPECT().NewBuilder(mockBlobstore, mockAgentClient).Return(mockStateBuilder).AnyTimes()
		mockStateBuilder.EXPECT().Build(jobName, jobIndex, deploymentManifest, fakeStage, fakeAgentState).Return(mockState, nil).AnyTimes()
		mockStateBuilder.EXPECT().BuildInitialState(jobName, jobIndex, deploymentManifest).Return(mockState, nil).AnyTimes()
		mockState.EXPECT().ToApplySpec().Return(applySpec).AnyTimes()
	})

	Context("when a previous instance exists", func() {
		var fakeExistingVM *fakebivm.FakeVM

		BeforeEach(func() {
			fakeExistingVM = fakebivm.NewFakeVM("existing-vm-cid")
			fakeVMManager.SetFindCurrentBehavior(fakeExistingVM, true, nil)
			fakeExistingVM.AgentClientReturn = mockAgentClient
		})

		It("deletes existing vm", func() {
			_, err := deployer.Deploy(cloud, deploymentManifest, cloudStemcell, registryConfig, fakeVMManager, mockBlobstore, fakeStage)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeExistingVM.DeleteCalled).To(Equal(1))

			Expect(fakeStage.PerformCalls[:3]).To(Equal([]*fakebiui.PerformCall{
				{Name: "Waiting for the agent on VM 'existing-vm-cid'"},
				{Name: "Stopping jobs on instance 'unknown/0'"},
				{Name: "Deleting VM 'existing-vm-cid'"},
			}))
		})
	})

	It("creates a vm", func() {
		_, err := deployer.Deploy(cloud, deploymentManifest, cloudStemcell, registryConfig, fakeVMManager, mockBlobstore, fakeStage)
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeVMManager.CreateInput).To(Equal(fakebivm.CreateInput{
			Stemcell: cloudStemcell,
			Manifest: deploymentManifest,
		}))
	})

	Context("when registry & ssh tunnel configs are not empty", func() {
		BeforeEach(func() {
			registryConfig = biinstallmanifest.Registry{
				Username: "fake-username",
				Password: "fake-password",
				Host:     "fake-host",
				Port:     123,
				SSHTunnel: biinstallmanifest.SSHTunnel{
					User:       "fake-ssh-username",
					PrivateKey: "---BEGIN PRIVATE KEY--- huzzah! ---END PRIVATE KEY---",
					Password:   "fake-password",
					Host:       "fake-ssh-host",
					Port:       124,
				},
			}
		})

		It("starts the SSH tunnel", func() {
			_, err := deployer.Deploy(cloud, deploymentManifest, cloudStemcell, registryConfig, fakeVMManager, mockBlobstore, fakeStage)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeSSHTunnel.Started).To(BeTrue())
			Expect(fakeSSHTunnelFactory.NewSSHTunnelOptions).To(Equal(bisshtunnel.Options{
				User:              "fake-ssh-username",
				PrivateKey:        "---BEGIN PRIVATE KEY--- huzzah! ---END PRIVATE KEY---",
				Password:          "fake-password",
				Host:              "fake-ssh-host",
				Port:              124,
				LocalForwardPort:  123,
				RemoteForwardPort: 123,
			}))
		})

		Context("when starting SSH tunnel fails", func() {
			BeforeEach(func() {
				fakeSSHTunnel.SetStartBehavior(errors.New("fake-ssh-tunnel-start-error"), nil)
			})

			It("returns an error", func() {
				_, err := deployer.Deploy(cloud, deploymentManifest, cloudStemcell, registryConfig, fakeVMManager, mockBlobstore, fakeStage)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-ssh-tunnel-start-error"))
			})
		})
	})

	It("waits for the vm", func() {
		_, err := deployer.Deploy(cloud, deploymentManifest, cloudStemcell, registryConfig, fakeVMManager, mockBlobstore, fakeStage)
		Expect(err).NotTo(HaveOccurred())
		Expect(fakeVM.WaitUntilReadyInputs).To(ContainElement(fakebivm.WaitUntilReadyInput{
			Timeout: 10 * time.Minute,
			Delay:   500 * time.Millisecond,
		}))
	})

	It("logs start and stop events to the eventLogger", func() {
		_, err := deployer.Deploy(cloud, deploymentManifest, cloudStemcell, registryConfig, fakeVMManager, mockBlobstore, fakeStage)
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeStage.PerformCalls[1]).To(Equal(&fakebiui.PerformCall{
			Name: "Waiting for the agent on VM 'fake-vm-cid' to be ready",
		}))
	})

	Context("when waiting for the agent fails", func() {
		var (
			waitError = bosherr.Error("fake-wait-error")
		)

		BeforeEach(func() {
			fakeVM.WaitUntilReadyErr = waitError
		})

		It("logs start and stop events to the eventLogger", func() {
			_, err := deployer.Deploy(cloud, deploymentManifest, cloudStemcell, registryConfig, fakeVMManager, mockBlobstore, fakeStage)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-wait-error"))

			Expect(fakeStage.PerformCalls[1]).To(Equal(&fakebiui.PerformCall{
				Name:  "Waiting for the agent on VM 'fake-vm-cid' to be ready",
				Error: waitError,
			}))
		})
	})

	It("updates the vm", func() {
		_, err := deployer.Deploy(cloud, deploymentManifest, cloudStemcell, registryConfig, fakeVMManager, mockBlobstore, fakeStage)
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeVM.ApplyInputs).To(Equal([]fakebivm.ApplyInput{
			{ApplySpec: applySpec},
			{ApplySpec: applySpec},
		}))
	})

	It("starts the agent", func() {
		_, err := deployer.Deploy(cloud, deploymentManifest, cloudStemcell, registryConfig, fakeVMManager, mockBlobstore, fakeStage)
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeVM.StartCalled).To(Equal(1))
	})

	It("waits until agent reports state as running", func() {
		_, err := deployer.Deploy(cloud, deploymentManifest, cloudStemcell, registryConfig, fakeVMManager, mockBlobstore, fakeStage)
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeVM.WaitToBeRunningInputs).To(ContainElement(fakebivm.WaitInput{
			MaxAttempts: 5,
			Delay:       1 * time.Second,
		}))
	})

	Context("when the deployment has an invalid disk pool specification", func() {
		BeforeEach(func() {
			deploymentManifest.Jobs[0].PersistentDiskPool = "fake-non-existent-persistent-disk-pool-name"
		})

		It("returns an error", func() {
			_, err := deployer.Deploy(cloud, deploymentManifest, cloudStemcell, registryConfig, fakeVMManager, mockBlobstore, fakeStage)
			Expect(err).To(HaveOccurred())
		})
	})

	It("logs instance update ui stages", func() {
		_, err := deployer.Deploy(cloud, deploymentManifest, cloudStemcell, registryConfig, fakeVMManager, mockBlobstore, fakeStage)
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeStage.PerformCalls[2:4]).To(Equal([]*fakebiui.PerformCall{
			{Name: "Updating instance 'fake-job-name/0'"},
			{Name: "Waiting for instance 'fake-job-name/0' to be running"},
		}))
	})

	Context("when applying instance spec fails", func() {
		BeforeEach(func() {
			fakeVM.ApplyErr = bosherr.Error("fake-apply-error")
		})

		It("fails with descriptive error", func() {
			_, err := deployer.Deploy(cloud, deploymentManifest, cloudStemcell, registryConfig, fakeVMManager, mockBlobstore, fakeStage)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Applying the initial agent state: fake-apply-error"))
		})
	})

	Context("when starting agent services fails", func() {
		BeforeEach(func() {
			fakeVM.StartErr = bosherr.Error("fake-start-error")
		})

		It("logs start and stop events to the eventLogger", func() {
			_, err := deployer.Deploy(cloud, deploymentManifest, cloudStemcell, registryConfig, fakeVMManager, mockBlobstore, fakeStage)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-start-error"))

			Expect(fakeStage.PerformCalls[2].Name).To(Equal("Updating instance 'fake-job-name/0'"))
			Expect(fakeStage.PerformCalls[2].Error).To(HaveOccurred())
			Expect(fakeStage.PerformCalls[2].Error.Error()).To(Equal("Starting the agent: fake-start-error"))
		})
	})

	Context("when waiting for running state fails", func() {
		var (
			waitError = bosherr.Error("fake-wait-running-error")
		)

		BeforeEach(func() {
			fakeVM.WaitToBeRunningErr = waitError
		})

		It("logs start and stop events to the eventLogger", func() {
			_, err := deployer.Deploy(cloud, deploymentManifest, cloudStemcell, registryConfig, fakeVMManager, mockBlobstore, fakeStage)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-wait-running-error"))

			Expect(fakeStage.PerformCalls[3]).To(Equal(&fakebiui.PerformCall{
				Name:  "Waiting for instance 'fake-job-name/0' to be running",
				Error: waitError,
			}))
		})
	})
})
