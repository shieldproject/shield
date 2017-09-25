package config_test

import (
	. "github.com/cloudfoundry/bosh-cli/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	"errors"
	"path/filepath"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
)

var _ = Describe("fileSystemDeploymentStateService", func() {
	var (
		service             DeploymentStateService
		deploymentStatePath string
		fakeFs              *fakesys.FakeFileSystem
		fakeUUIDGenerator   *fakeuuid.FakeGenerator
	)

	BeforeEach(func() {
		fakeFs = fakesys.NewFakeFileSystem()
		deploymentStatePath = "/some/deployment.json"
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fakeUUIDGenerator = fakeuuid.NewFakeGenerator()
		service = NewFileSystemDeploymentStateService(fakeFs, fakeUUIDGenerator, logger, deploymentStatePath)
	})

	Describe("DeploymentStatePath", func() {
		Context("when statePath is NOT specified", func() {
			It("is based on the manifest path and name", func() {
				Expect(DeploymentStatePath("/path/to/some-manifest.yml", "")).To(Equal(filepath.Join("/", "path", "to", "some-manifest-state.json")))
				Expect(DeploymentStatePath("/path/to/some-manifesty.yaml", "")).To(Equal(filepath.Join("/", "path", "to", "some-manifesty-state.json")))
				Expect(DeploymentStatePath("/path/to/some-manifest", "")).To(Equal(filepath.Join("/", "path", "to", "some-manifest-state.json")))
			})
		})

		Describe("statePath is specified", func() {
			Context("and is a file", func() {
				It("is based on the manifest path and name", func() {
					Expect(DeploymentStatePath("/path/to/some-manifest.yml", "/whatever/they/pass")).To(Equal("/whatever/they/pass"))
				})
			})
		})
	})

	Describe("Exists", func() {
		It("returns true if the config file exists", func() {
			fakeFs.WriteFileString(deploymentStatePath, "")
			Expect(service.Exists()).To(BeTrue())
		})

		It("returns false if the config file does not exist", func() {
			Expect(service.Exists()).To(BeFalse())
		})
	})

	Describe("Load", func() {
		It("reads the given config file", func() {
			stemcells := []StemcellRecord{
				StemcellRecord{
					Name:    "fake-stemcell-name-1",
					Version: "fake-stemcell-version-1",
					CID:     "fake-stemcell-cid-1",
				},
				StemcellRecord{
					Name:    "fake-stemcell-name-2",
					Version: "fake-stemcell-version-2",
					CID:     "fake-stemcell-cid-2",
				},
			}
			disks := []DiskRecord{
				{
					ID:   "fake-disk-id",
					CID:  "fake-disk-cid",
					Size: 1024,
					CloudProperties: biproperty.Map{
						"fake-disk-property-key": "fake-disk-property-value",
					},
				},
			}
			deploymentStateFileContents, err := json.Marshal(biproperty.Map{
				"director_id":     "fake-director-id",
				"deployment_id":   "fake-deployment-id",
				"stemcells":       stemcells,
				"current_vm_cid":  "fake-vm-cid",
				"current_disk_id": "fake-disk-id",
				"disks":           disks,
			})
			fakeFs.WriteFile(deploymentStatePath, deploymentStateFileContents)

			deploymentState, err := service.Load()
			Expect(err).NotTo(HaveOccurred())
			Expect(deploymentState.DirectorID).To(Equal("fake-director-id"))
			Expect(deploymentState.Stemcells).To(Equal(stemcells))
			Expect(deploymentState.CurrentVMCID).To(Equal("fake-vm-cid"))
			Expect(deploymentState.CurrentDiskID).To(Equal("fake-disk-id"))
			Expect(deploymentState.Disks).To(Equal(disks))
		})

		Context("when the config does not exist", func() {
			It("returns a new DeploymentState with generated defaults", func() {
				deploymentState, err := service.Load()
				Expect(err).NotTo(HaveOccurred())
				Expect(deploymentState).To(Equal(DeploymentState{
					DirectorID: "fake-uuid-0",
				}))

				Expect(fakeFs.FileExists(deploymentStatePath)).To(BeTrue())
			})
		})

		Context("when reading config file fails", func() {
			BeforeEach(func() {
				fakeFs.WriteFileString(deploymentStatePath, "{}")
				fakeFs.ReadFileError = errors.New("fake-read-error")
			})

			It("returns an error", func() {
				_, err := service.Load()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-read-error"))
			})
		})

		Context("when the config is invalid", func() {
			It("returns an empty DeploymentState and an error", func() {
				fakeFs.WriteFileString(deploymentStatePath, "some invalid content")
				deploymentState, err := service.Load()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Unmarshalling deployment state file '/some/deployment.json'"))
				Expect(deploymentState).To(Equal(DeploymentState{}))
			})
		})
	})

	Describe("Save", func() {
		It("writes the deployment state to the deployment file", func() {
			config := DeploymentState{
				DirectorID: "deadbeef",
				Stemcells: []StemcellRecord{
					{
						Name:    "fake-stemcell-name",
						Version: "fake-stemcell-version",
						CID:     "fake-stemcell-cid",
					},
				},
				CurrentVMCID: "fake-vm-cid",
				Disks: []DiskRecord{
					{
						CID:  "fake-disk-cid",
						Size: 1024,
						CloudProperties: biproperty.Map{
							"fake-disk-property-key": "fake-disk-property-value",
						},
					},
				},
			}

			err := service.Save(config)
			Expect(err).NotTo(HaveOccurred())

			deploymentStateFileContents, err := fakeFs.ReadFileString(deploymentStatePath)
			deploymentState := DeploymentState{
				DirectorID: "deadbeef",
				Stemcells: []StemcellRecord{
					{
						Name:    "fake-stemcell-name",
						Version: "fake-stemcell-version",
						CID:     "fake-stemcell-cid",
					},
				},
				CurrentVMCID: "fake-vm-cid",
				Disks: []DiskRecord{
					{
						CID:  "fake-disk-cid",
						Size: 1024,
						CloudProperties: biproperty.Map{
							"fake-disk-property-key": "fake-disk-property-value",
						},
					},
				},
			}
			expectedDeploymentStateFileContents, err := json.MarshalIndent(deploymentState, "", "    ")
			Expect(deploymentStateFileContents).To(Equal(string(expectedDeploymentStateFileContents)))
		})

		Context("when the deployment file cannot be written", func() {
			BeforeEach(func() {
				fakeFs.WriteFileError = errors.New("")
			})

			It("returns an error when it cannot write the config file", func() {
				config := DeploymentState{
					Stemcells: []StemcellRecord{},
				}
				err := service.Save(config)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Writing deployment state file '/some/deployment.json'"))
			})
		})
	})

	Describe("Cleanup", func() {
		It("returns true if deployment state file deleted", func() {
			fakeFs.WriteFileString(deploymentStatePath, "")
			Expect(service.Exists()).To(BeTrue())

			err := service.Cleanup()

			Expect(err).ToNot(HaveOccurred())

			Expect(service.Exists()).To(BeFalse())

		})

		It("returns error if delete opertation fails to remove file", func() {
			fakeFs.RemoveAllStub = func(_ string) error {
				return errors.New("Could not do that Dave")
			}

			Expect(service.Exists()).To(BeFalse())

			err := service.Cleanup()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Could not do that Dave"))
		})
	})
})
