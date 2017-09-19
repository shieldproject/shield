package disk_test

import (
	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	bicloud "github.com/cloudfoundry/bosh-cli/cloud"
	biconfig "github.com/cloudfoundry/bosh-cli/config"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"

	fakebicloud "github.com/cloudfoundry/bosh-cli/cloud/fakes"

	. "github.com/cloudfoundry/bosh-cli/deployment/disk"
)

var _ = Describe("Disk", func() {
	var (
		disk                Disk
		diskCloudProperties biproperty.Map
		fakeCloud           *fakebicloud.FakeCloud
		diskRepo            biconfig.DiskRepo
		fakeUUIDGenerator   *fakeuuid.FakeGenerator
	)

	BeforeEach(func() {
		diskCloudProperties = biproperty.Map{
			"fake-cloud-property-key": "fake-cloud-property-value",
		}
		fakeCloud = fakebicloud.NewFakeCloud()

		diskRecord := biconfig.DiskRecord{
			CID:             "fake-disk-cid",
			Size:            1024,
			CloudProperties: diskCloudProperties,
		}

		fs := fakesys.NewFakeFileSystem()
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fakeUUIDGenerator = &fakeuuid.FakeGenerator{}
		//		todo: come back to this?
		deploymentStateService := biconfig.NewFileSystemDeploymentStateService(fs, fakeUUIDGenerator, logger, "/fake/path")
		diskRepo = biconfig.NewDiskRepo(deploymentStateService, fakeUUIDGenerator)

		disk = NewDisk(diskRecord, fakeCloud, diskRepo)
	})

	Describe("NeedsMigration", func() {
		Context("when size is different", func() {
			It("returns true", func() {
				needsMigration := disk.NeedsMigration(2048, diskCloudProperties)
				Expect(needsMigration).To(BeTrue())
			})
		})

		Context("when cloud properties are different", func() {
			It("returns true", func() {
				newDiskCloudProperties := biproperty.Map{
					"fake-cloud-property-key": "new-fake-cloud-property-value",
				}

				needsMigration := disk.NeedsMigration(1024, newDiskCloudProperties)
				Expect(needsMigration).To(BeTrue())
			})
		})

		Context("when cloud properties are nil", func() {
			It("returns true", func() {
				needsMigration := disk.NeedsMigration(1024, nil)
				Expect(needsMigration).To(BeTrue())
			})
		})

		Context("when size and cloud properties are the same", func() {
			It("returns false", func() {
				needsMigration := disk.NeedsMigration(1024, diskCloudProperties)
				Expect(needsMigration).To(BeFalse())
			})
		})
	})

	Describe("Delete", func() {
		It("deletes disk from cloud", func() {
			err := disk.Delete()
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeCloud.DeleteDiskInputs).To(Equal([]fakebicloud.DeleteDiskInput{
				{
					DiskCID: "fake-disk-cid",
				},
			}))
		})

		It("deletes disk from repo", func() {
			_, err := diskRepo.Save("fake-disk-cid", 1024, diskCloudProperties)
			Expect(err).ToNot(HaveOccurred())

			err = disk.Delete()
			Expect(err).ToNot(HaveOccurred())
			diskRecords, err := diskRepo.All()
			Expect(diskRecords).To(BeEmpty())
		})

		Context("when deleted disk is the current disk", func() {
			BeforeEach(func() {
				diskRecord, err := diskRepo.Save("fake-disk-cid", 1024, diskCloudProperties)
				Expect(err).ToNot(HaveOccurred())

				err = diskRepo.UpdateCurrent(diskRecord.ID)
				Expect(err).ToNot(HaveOccurred())
			})

			It("clears current disk in the disk repo", func() {
				err := disk.Delete()
				Expect(err).ToNot(HaveOccurred())

				_, found, err := diskRepo.FindCurrent()
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeFalse())
			})
		})

		Context("when deleting disk in the cloud fails", func() {
			BeforeEach(func() {
				fakeCloud.DeleteDiskErr = errors.New("fake-delete-disk-error")
			})

			It("returns an error", func() {
				err := disk.Delete()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-delete-disk-error"))
			})
		})

		Context("when deleting disk in the cloud fails with DiskNotFoundError", func() {
			var deleteErr = bicloud.NewCPIError("delete_vm", bicloud.CmdError{
				Type:    bicloud.DiskNotFoundError,
				Message: "fake-disk-not-found-message",
			})

			BeforeEach(func() {
				diskRecord, err := diskRepo.Save("fake-disk-cid", 1024, diskCloudProperties)
				Expect(err).ToNot(HaveOccurred())

				err = diskRepo.UpdateCurrent(diskRecord.ID)
				Expect(err).ToNot(HaveOccurred())

				fakeCloud.DeleteDiskErr = deleteErr
			})

			It("deletes disk in the cloud", func() {
				err := disk.Delete()
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(deleteErr))

				Expect(fakeCloud.DeleteDiskInputs).To(Equal([]fakebicloud.DeleteDiskInput{
					{
						DiskCID: "fake-disk-cid",
					},
				}))
			})

			It("deletes disk in the disk repo", func() {
				err := disk.Delete()
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(deleteErr))

				diskRecords, err := diskRepo.All()
				Expect(diskRecords).To(BeEmpty())
			})

			It("clears current disk in the disk repo", func() {
				err := disk.Delete()
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(deleteErr))

				_, found, err := diskRepo.FindCurrent()
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeFalse())
			})
		})
	})
})
