package vm_test

import (
	"errors"

	fakebiagentclient "github.com/cloudfoundry/bosh-agent/agentclient/fakes"
	"github.com/cloudfoundry/bosh-init/cloud"
	fakebicloud "github.com/cloudfoundry/bosh-init/cloud/fakes"
	biconfig "github.com/cloudfoundry/bosh-init/config"
	fakebiconfig "github.com/cloudfoundry/bosh-init/config/fakes"
	bideplmanifest "github.com/cloudfoundry/bosh-init/deployment/manifest"
	. "github.com/cloudfoundry/bosh-init/deployment/vm"
	fakebivm "github.com/cloudfoundry/bosh-init/deployment/vm/fakes"
	bistemcell "github.com/cloudfoundry/bosh-init/stemcell"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manager", func() {
	var (
		fakeCloud                 *fakebicloud.FakeCloud
		manager                   Manager
		logger                    boshlog.Logger
		expectedNetworkInterfaces map[string]biproperty.Map
		expectedCloudProperties   biproperty.Map
		expectedEnv               biproperty.Map
		deploymentManifest        bideplmanifest.Manifest
		fakeVMRepo                *fakebiconfig.FakeVMRepo
		stemcellRepo              biconfig.StemcellRepo
		fakeDiskDeployer          *fakebivm.FakeDiskDeployer
		fakeAgentClient           *fakebiagentclient.FakeAgentClient
		stemcell                  bistemcell.CloudStemcell
		fs                        *fakesys.FakeFileSystem
	)

	BeforeEach(func() {
		logger = boshlog.NewLogger(boshlog.LevelNone)
		fs = fakesys.NewFakeFileSystem()
		fakeCloud = fakebicloud.NewFakeCloud()
		fakeAgentClient = &fakebiagentclient.FakeAgentClient{}
		fakeVMRepo = fakebiconfig.NewFakeVMRepo()

		fakeUUIDGenerator := &fakeuuid.FakeGenerator{}
		deploymentStateService := biconfig.NewFileSystemDeploymentStateService(fs, fakeUUIDGenerator, logger, "/fake/path")
		stemcellRepo = biconfig.NewStemcellRepo(deploymentStateService, fakeUUIDGenerator)

		fakeDiskDeployer = fakebivm.NewFakeDiskDeployer()

		manager = NewManagerFactory(
			fakeVMRepo,
			stemcellRepo,
			fakeDiskDeployer,
			fakeUUIDGenerator,
			fs,
			logger,
		).NewManager(fakeCloud, fakeAgentClient)

		fakeCloud.CreateVMCID = "fake-vm-cid"
		expectedNetworkInterfaces = map[string]biproperty.Map{
			"fake-network-name": biproperty.Map{
				"type":             "dynamic",
				"ip":               "fake-ip",
				"cloud_properties": biproperty.Map{},
				"default":          []bideplmanifest.NetworkDefault{"dns", "gateway"},
			},
		}
		expectedCloudProperties = biproperty.Map{
			"fake-cloud-property-key": "fake-cloud-property-value",
		}
		expectedEnv = biproperty.Map{
			"fake-env-key": "fake-env-value",
		}
		deploymentManifest = bideplmanifest.Manifest{
			Name: "fake-deployment",
			Networks: []bideplmanifest.Network{
				{
					Name:            "fake-network-name",
					Type:            "dynamic",
					CloudProperties: biproperty.Map{},
				},
			},
			ResourcePools: []bideplmanifest.ResourcePool{
				{
					Name: "fake-resource-pool-name",
					CloudProperties: biproperty.Map{
						"fake-cloud-property-key": "fake-cloud-property-value",
					},
					Env: biproperty.Map{
						"fake-env-key": "fake-env-value",
					},
				},
			},
			Jobs: []bideplmanifest.Job{
				{
					Name: "fake-job",
					Networks: []bideplmanifest.JobNetwork{
						{
							Name:      "fake-network-name",
							StaticIPs: []string{"fake-ip"},
						},
					},
					ResourcePool: "fake-resource-pool-name",
				},
			},
		}

		stemcellRecord := biconfig.StemcellRecord{CID: "fake-stemcell-cid"}
		stemcell = bistemcell.NewCloudStemcell(stemcellRecord, stemcellRepo, fakeCloud)
	})

	Describe("Create", func() {
		It("creates a VM", func() {
			vm, err := manager.Create(stemcell, deploymentManifest)
			Expect(err).ToNot(HaveOccurred())
			expectedVM := NewVM(
				"fake-vm-cid",
				fakeVMRepo,
				stemcellRepo,
				fakeDiskDeployer,
				fakeAgentClient,
				fakeCloud,
				fs,
				logger,
			)
			Expect(vm).To(Equal(expectedVM))

			Expect(fakeCloud.CreateVMInput).To(Equal(
				fakebicloud.CreateVMInput{
					AgentID:            "fake-uuid-0",
					StemcellCID:        "fake-stemcell-cid",
					CloudProperties:    expectedCloudProperties,
					NetworksInterfaces: expectedNetworkInterfaces,
					Env:                expectedEnv,
				},
			))
		})

		It("sets the vm metadata", func() {
			_, err := manager.Create(stemcell, deploymentManifest)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeCloud.SetVMMetadataCid).To(Equal("fake-vm-cid"))
			Expect(fakeCloud.SetVMMetadataMetadata).To(Equal(cloud.VMMetadata{
				Deployment: "fake-deployment",
				Job:        "fake-job",
				Index:      "0",
				Director:   "bosh-init",
			}))
		})

		It("updates the current vm record", func() {
			_, err := manager.Create(stemcell, deploymentManifest)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeVMRepo.UpdateCurrentCID).To(Equal("fake-vm-cid"))
		})

		Context("when setting vm metadata fails", func() {
			BeforeEach(func() {
				fakeCloud.SetVMMetadataError = errors.New("fake-set-metadata-error")
			})

			It("returns an error", func() {
				_, err := manager.Create(stemcell, deploymentManifest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-set-metadata-error"))
			})

			It("still updates the current vm record", func() {
				_, err := manager.Create(stemcell, deploymentManifest)
				Expect(err).To(HaveOccurred())
				Expect(fakeVMRepo.UpdateCurrentCID).To(Equal("fake-vm-cid"))
			})

			It("ignores not implemented error", func() {
				notImplementedCloudError := cloud.NewCPIError("set_vm_metadata", cloud.CmdError{
					Type:      "Bosh::Clouds::NotImplemented",
					Message:   "set_vm_metadata is not implemented by VCloudCloud::Cloud",
					OkToRetry: false,
				})
				fakeCloud.SetVMMetadataError = notImplementedCloudError

				_, err := manager.Create(stemcell, deploymentManifest)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when creating the vm fails", func() {
			BeforeEach(func() {
				fakeCloud.CreateVMErr = errors.New("fake-create-error")
			})

			It("returns an error", func() {
				_, err := manager.Create(stemcell, deploymentManifest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-create-error"))
			})
		})
	})
})
