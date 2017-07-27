package config_test

import (
	. "github.com/cloudfoundry/bosh-cli/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
)

var _ = Describe("DiskRepo", func() {
	var (
		deploymentStateService DeploymentStateService
		repo                   DiskRepo
		fs                     *fakesys.FakeFileSystem
		fakeUUIDGenerator      *fakeuuid.FakeGenerator
		cloudProperties        biproperty.Map
	)

	BeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fs = fakesys.NewFakeFileSystem()
		fakeUUIDGenerator = &fakeuuid.FakeGenerator{}
		deploymentStateService = NewFileSystemDeploymentStateService(fs, fakeUUIDGenerator, logger, "/fake/path")
		repo = NewDiskRepo(deploymentStateService, fakeUUIDGenerator)
		cloudProperties = biproperty.Map{
			"fake-cloud_property-key": "fake-cloud-property-value",
		}
	})

	Describe("Save", func() {
		It("saves the disk record using the config service", func() {
			record, err := repo.Save("fake-cid", 1024, cloudProperties)
			Expect(err).ToNot(HaveOccurred())
			Expect(record).To(Equal(DiskRecord{
				ID:              "fake-uuid-1",
				CID:             "fake-cid",
				Size:            1024,
				CloudProperties: cloudProperties,
			}))

			deploymentState, err := deploymentStateService.Load()
			Expect(err).ToNot(HaveOccurred())

			expectedConfig := DeploymentState{
				DirectorID: "fake-uuid-0",
				Disks: []DiskRecord{
					{
						ID:              "fake-uuid-1",
						CID:             "fake-cid",
						Size:            1024,
						CloudProperties: cloudProperties,
					},
				},
			}
			Expect(deploymentState).To(Equal(expectedConfig))
		})
	})

	Describe("Find", func() {
		It("finds existing disk records", func() {
			savedRecord, err := repo.Save("fake-cid", 1024, cloudProperties)
			Expect(err).ToNot(HaveOccurred())

			foundRecord, found, err := repo.Find("fake-cid")
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(foundRecord).To(Equal(savedRecord))
		})

		It("when the disk is not in the records, returns not found", func() {
			_, err := repo.Save("other-cid", 1024, cloudProperties)
			Expect(err).ToNot(HaveOccurred())

			_, found, err := repo.Find("fake-cid")
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeFalse())
		})
	})

	Describe("UpdateCurrent", func() {
		Context("when a disk record exists with the same ID", func() {
			var (
				recordID string
			)

			BeforeEach(func() {
				record, err := repo.Save("fake-cid", 1024, cloudProperties)
				Expect(err).ToNot(HaveOccurred())
				recordID = record.ID
			})

			It("saves the disk record as current stemcell", func() {
				err := repo.UpdateCurrent(recordID)
				Expect(err).ToNot(HaveOccurred())

				deploymentState, err := deploymentStateService.Load()
				Expect(err).ToNot(HaveOccurred())

				Expect(deploymentState.CurrentDiskID).To(Equal(recordID))
			})
		})

		Context("when a disk record does not exists with the same ID", func() {
			BeforeEach(func() {
				_, err := repo.Save("fake-cid", 1024, cloudProperties)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				err := repo.UpdateCurrent("fake-unknown-id")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Verifying disk record exists with id 'fake-unknown-id'"))
			})
		})
	})

	Describe("FindCurrent", func() {
		Context("when current disk exists", func() {
			var (
				diskID2 string
			)
			BeforeEach(func() {
				_, err := repo.Save("fake-cid-1", 1024, cloudProperties)
				Expect(err).ToNot(HaveOccurred())

				record, err := repo.Save("fake-cid-2", 1024, cloudProperties)
				Expect(err).ToNot(HaveOccurred())
				diskID2 = record.ID

				repo.UpdateCurrent(record.ID)
			})

			It("returns existing disk", func() {
				record, found, err := repo.FindCurrent()
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())
				Expect(record).To(Equal(DiskRecord{
					ID:              diskID2,
					CID:             "fake-cid-2",
					Size:            1024,
					CloudProperties: cloudProperties,
				}))
			})
		})

		Context("when current disk does not exist", func() {
			BeforeEach(func() {
				_, err := repo.Save("fake-cid", 1024, cloudProperties)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns not found", func() {
				_, found, err := repo.FindCurrent()
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeFalse())
			})
		})

		Context("when there are no disks", func() {
			It("returns not found", func() {
				_, found, err := repo.FindCurrent()
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeFalse())
			})
		})
	})

	Describe("All", func() {
		var (
			firstDisk  DiskRecord
			secondDisk DiskRecord
		)

		BeforeEach(func() {
			var err error
			firstDisk, err = repo.Save("fake-cid-1", 1024, cloudProperties)
			Expect(err).ToNot(HaveOccurred())

			secondDisk, err = repo.Save("fake-cid-2", 2048, cloudProperties)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns all disks", func() {
			disks, err := repo.All()
			Expect(err).ToNot(HaveOccurred())
			Expect(disks).To(Equal([]DiskRecord{
				firstDisk,
				secondDisk,
			}))
		})
	})

	Describe("Delete", func() {
		var (
			firstDisk  DiskRecord
			secondDisk DiskRecord
		)

		BeforeEach(func() {
			var err error

			firstDisk, err = repo.Save("fake-cid-1", 1024, cloudProperties)
			Expect(err).ToNot(HaveOccurred())

			secondDisk, err = repo.Save("fake-cid-2", 2048, cloudProperties)
			Expect(err).ToNot(HaveOccurred())
		})

		It("removes the disk record from the repo", func() {
			err := repo.Delete(firstDisk)
			Expect(err).ToNot(HaveOccurred())

			disks, err := repo.All()
			Expect(err).ToNot(HaveOccurred())
			Expect(disks).To(Equal([]DiskRecord{
				secondDisk,
			}))
		})

		Context("when the disk to be deleted is also the current disk", func() {
			BeforeEach(func() {
				err := repo.UpdateCurrent(firstDisk.ID)
				Expect(err).ToNot(HaveOccurred())
			})

			It("clears the current disk", func() {
				err := repo.Delete(firstDisk)
				Expect(err).ToNot(HaveOccurred())

				disks, err := repo.All()
				Expect(err).ToNot(HaveOccurred())
				Expect(disks).To(Equal([]DiskRecord{
					secondDisk,
				}))

				_, found, err := repo.FindCurrent()
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeFalse())
			})
		})
	})

	Describe("ClearCurrent", func() {
		It("updates disk cid", func() {
			err := repo.ClearCurrent()
			Expect(err).ToNot(HaveOccurred())

			deploymentState, err := deploymentStateService.Load()
			Expect(err).ToNot(HaveOccurred())

			expectedConfig := DeploymentState{
				DirectorID:    "fake-uuid-0",
				CurrentDiskID: "",
			}
			Expect(deploymentState).To(Equal(expectedConfig))

			_, found, err := repo.FindCurrent()
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeFalse())
		})
	})
})
