package disk_test

import (
	"errors"

	fakebicloud "github.com/cloudfoundry/bosh-cli/cloud/fakes"
	biconfig "github.com/cloudfoundry/bosh-cli/config"
	. "github.com/cloudfoundry/bosh-cli/deployment/disk"
	bidisk "github.com/cloudfoundry/bosh-cli/deployment/disk"
	bideplmanifest "github.com/cloudfoundry/bosh-cli/deployment/manifest"
	fakebiui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manager", func() {
	var (
		manager           Manager
		fakeCloud         *fakebicloud.FakeCloud
		fakeFs            *fakesys.FakeFileSystem
		fakeUUIDGenerator *fakeuuid.FakeGenerator
		diskRepo          biconfig.DiskRepo
	)

	BeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fakeFs = fakesys.NewFakeFileSystem()
		fakeUUIDGenerator = &fakeuuid.FakeGenerator{}
		deploymentStateService := biconfig.NewFileSystemDeploymentStateService(fakeFs, fakeUUIDGenerator, logger, "/fake/path")
		diskRepo = biconfig.NewDiskRepo(deploymentStateService, fakeUUIDGenerator)
		managerFactory := NewManagerFactory(diskRepo, logger)
		fakeCloud = fakebicloud.NewFakeCloud()
		manager = managerFactory.NewManager(fakeCloud)
		fakeUUIDGenerator.GeneratedUUID = "fake-uuid"
	})

	Describe("Create", func() {
		var (
			diskPool bideplmanifest.DiskPool
		)

		BeforeEach(func() {

			diskPool = bideplmanifest.DiskPool{
				Name:     "fake-disk-pool-name",
				DiskSize: 1024,
				CloudProperties: biproperty.Map{
					"fake-cloud-property-key": "fake-cloud-property-value",
				},
			}
		})

		Context("when creating disk succeeds", func() {
			BeforeEach(func() {
				fakeCloud.CreateDiskCID = "fake-disk-cid"
			})

			It("returns a disk", func() {
				disk, err := manager.Create(diskPool, "fake-vm-cid")
				Expect(err).ToNot(HaveOccurred())
				Expect(disk.CID()).To(Equal("fake-disk-cid"))
			})

			It("saves the disk record", func() {
				_, err := manager.Create(diskPool, "fake-vm-cid")
				Expect(err).ToNot(HaveOccurred())

				diskRecord, found, err := diskRepo.Find("fake-disk-cid")
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())

				Expect(diskRecord).To(Equal(biconfig.DiskRecord{
					ID:   "fake-uuid",
					CID:  "fake-disk-cid",
					Size: 1024,
					CloudProperties: biproperty.Map{
						"fake-cloud-property-key": "fake-cloud-property-value",
					},
				}))
			})
		})

		Context("when creating disk fails", func() {
			BeforeEach(func() {
				fakeCloud.CreateDiskErr = errors.New("fake-create-error")
			})

			It("returns an error", func() {
				_, err := manager.Create(diskPool, "fake-vm-cid")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-create-error"))
			})
		})

		Context("when updating disk record fails", func() {
			BeforeEach(func() {
				fakeFs.WriteFileError = errors.New("fake-write-error")
			})

			It("returns an error", func() {
				_, err := manager.Create(diskPool, "fake-vm-cid")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-write-error"))
			})
		})
	})

	Describe("FindCurrent", func() {
		Context("when disk already exists in disk repo", func() {
			BeforeEach(func() {
				diskRecord, err := diskRepo.Save("fake-existing-disk-cid", 1024, biproperty.Map{})
				Expect(err).ToNot(HaveOccurred())

				err = diskRepo.UpdateCurrent(diskRecord.ID)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns the existing disk", func() {
				disks, err := manager.FindCurrent()
				Expect(err).ToNot(HaveOccurred())
				Expect(disks).To(HaveLen(1))
				Expect(disks[0].CID()).To(Equal("fake-existing-disk-cid"))
			})
		})

		Context("when disk does not exists in disk repo", func() {
			It("returns an empty array", func() {
				disks, err := manager.FindCurrent()
				Expect(err).ToNot(HaveOccurred())
				Expect(disks).To(BeEmpty())
			})
		})

		Context("when reading disk repo fails", func() {
			BeforeEach(func() {
				fakeFs.WriteFileString("/fake/path", "{}")
				fakeFs.ReadFileError = errors.New("fake-read-error")
			})

			It("returns an error", func() {
				_, err := manager.FindCurrent()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-read-error"))
			})
		})
	})

	Describe("FindUnused", func() {
		var (
			firstDisk bidisk.Disk
			thirdDisk bidisk.Disk
		)

		BeforeEach(func() {
			fakeUUIDGenerator.GeneratedUUID = "fake-guid-1"
			firstDiskRecord, err := diskRepo.Save("fake-disk-cid-1", 1024, biproperty.Map{})
			Expect(err).ToNot(HaveOccurred())
			firstDisk = NewDisk(firstDiskRecord, fakeCloud, diskRepo)

			fakeUUIDGenerator.GeneratedUUID = "fake-guid-2"
			_, err = diskRepo.Save("fake-disk-cid-2", 1024, biproperty.Map{})
			Expect(err).ToNot(HaveOccurred())
			err = diskRepo.UpdateCurrent("fake-guid-2")
			Expect(err).ToNot(HaveOccurred())

			fakeUUIDGenerator.GeneratedUUID = "fake-guid-3"
			thirdDiskRecord, err := diskRepo.Save("fake-disk-cid-3", 1024, biproperty.Map{})
			Expect(err).ToNot(HaveOccurred())
			thirdDisk = NewDisk(thirdDiskRecord, fakeCloud, diskRepo)
		})

		It("returns unused disks from repo", func() {
			disks, err := manager.FindUnused()
			Expect(err).ToNot(HaveOccurred())

			Expect(disks).To(Equal([]bidisk.Disk{
				firstDisk,
				thirdDisk,
			}))
		})
	})

	Describe("DeleteUnused", func() {
		var (
			secondDiskRecord biconfig.DiskRecord
			fakeStage        *fakebiui.FakeStage
		)
		BeforeEach(func() {
			fakeStage = fakebiui.NewFakeStage()

			fakeUUIDGenerator.GeneratedUUID = "fake-disk-id-1"
			_, err := diskRepo.Save("fake-disk-cid-1", 100, nil)
			Expect(err).ToNot(HaveOccurred())

			fakeUUIDGenerator.GeneratedUUID = "fake-disk-id-2"
			secondDiskRecord, err = diskRepo.Save("fake-disk-cid-2", 100, nil)
			Expect(err).ToNot(HaveOccurred())
			err = diskRepo.UpdateCurrent(secondDiskRecord.ID)
			Expect(err).ToNot(HaveOccurred())

			fakeUUIDGenerator.GeneratedUUID = "fake-disk-id-3"
			_, err = diskRepo.Save("fake-disk-cid-3", 100, nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("deletes unused disks", func() {
			err := manager.DeleteUnused(fakeStage)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeCloud.DeleteDiskInputs).To(Equal([]fakebicloud.DeleteDiskInput{
				{DiskCID: "fake-disk-cid-1"},
				{DiskCID: "fake-disk-cid-3"},
			}))

			Expect(fakeStage.PerformCalls).To(Equal([]*fakebiui.PerformCall{
				{Name: "Deleting unused disk 'fake-disk-cid-1'"},
				{Name: "Deleting unused disk 'fake-disk-cid-3'"},
			}))

			currentRecord, found, err := diskRepo.FindCurrent()
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(currentRecord).To(Equal(secondDiskRecord))

			records, err := diskRepo.All()
			Expect(err).ToNot(HaveOccurred())
			Expect(records).To(Equal([]biconfig.DiskRecord{
				secondDiskRecord,
			}))
		})
	})
})
