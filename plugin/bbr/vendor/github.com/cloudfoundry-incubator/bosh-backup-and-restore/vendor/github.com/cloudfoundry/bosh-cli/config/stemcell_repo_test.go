package config_test

import (
	. "github.com/cloudfoundry/bosh-cli/config"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StemcellRepo", func() {
	var (
		repo                   StemcellRepo
		deploymentStateService DeploymentStateService
		fs                     *fakesys.FakeFileSystem
		fakeUUIDGenerator      *fakeuuid.FakeGenerator
	)

	BeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fs = fakesys.NewFakeFileSystem()
		fakeUUIDGenerator = &fakeuuid.FakeGenerator{}
		deploymentStateService = NewFileSystemDeploymentStateService(fs, fakeUUIDGenerator, logger, "/fake/path")
		repo = NewStemcellRepo(deploymentStateService, fakeUUIDGenerator)
	})

	Describe("Save", func() {
		It("saves the stemcell record using the config service", func() {
			_, err := repo.Save("fake-name", "fake-version", "fake-cid")
			Expect(err).ToNot(HaveOccurred())

			deploymentState, err := deploymentStateService.Load()
			Expect(err).ToNot(HaveOccurred())

			expectedConfig := DeploymentState{
				DirectorID: "fake-uuid-0",
				Stemcells: []StemcellRecord{
					{
						ID:      "fake-uuid-1",
						Name:    "fake-name",
						Version: "fake-version",
						CID:     "fake-cid",
					},
				},
			}
			Expect(deploymentState).To(Equal(expectedConfig))
		})

		It("returns the stemcell record with a new uuid", func() {
			fakeUUIDGenerator.GeneratedUUID = "fake-uuid-1"
			record, err := repo.Save("fake-name", "fake-version-1", "fake-cid-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(record).To(Equal(StemcellRecord{
				ID:      "fake-uuid-1",
				Name:    "fake-name",
				Version: "fake-version-1",
				CID:     "fake-cid-1",
			}))

			fakeUUIDGenerator.GeneratedUUID = "fake-uuid-2"
			record, err = repo.Save("fake-name", "fake-version-2", "fake-cid-2")
			Expect(err).ToNot(HaveOccurred())
			Expect(record).To(Equal(StemcellRecord{
				ID:      "fake-uuid-2",
				Name:    "fake-name",
				Version: "fake-version-2",
				CID:     "fake-cid-2",
			}))
		})

		Context("when a stemcell record with the same name and version exists", func() {
			BeforeEach(func() {
				_, err := repo.Save("fake-name", "fake-version", "fake-cid")
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				_, err := repo.Save("fake-name", "fake-version", "fake-cid-2")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("duplicate name/version"))
			})
		})

		Context("when there stemcell record with the same cid exists (cpi does not garentee cid uniqueness)", func() {
			BeforeEach(func() {
				_, err := repo.Save("fake-name-1", "fake-version-1", "fake-cid-1")
				Expect(err).ToNot(HaveOccurred())
			})

			It("saves the stemcell record using the config service", func() {
				_, err := repo.Save("fake-name-2", "fake-version-2", "fake-cid-1")
				Expect(err).ToNot(HaveOccurred())

				deploymentState, err := deploymentStateService.Load()
				Expect(err).ToNot(HaveOccurred())

				expectedConfig := DeploymentState{
					DirectorID: "fake-uuid-0",
					Stemcells: []StemcellRecord{
						{
							ID:      "fake-uuid-1",
							Name:    "fake-name-1",
							Version: "fake-version-1",
							CID:     "fake-cid-1",
						},
						{
							ID:      "fake-uuid-2",
							Name:    "fake-name-2",
							Version: "fake-version-2",
							CID:     "fake-cid-1",
						},
					},
				}
				Expect(deploymentState).To(Equal(expectedConfig))
			})

			It("returns the stemcell record with a new uuid", func() {
				record, err := repo.Save("fake-name-2", "fake-version-2", "fake-cid-1")
				Expect(err).ToNot(HaveOccurred())
				Expect(record).To(Equal(StemcellRecord{
					ID:      "fake-uuid-2",
					Name:    "fake-name-2",
					Version: "fake-version-2",
					CID:     "fake-cid-1",
				}))
			})
		})
	})

	Describe("Find", func() {
		Context("when a stemcell record with the same name and version exists", func() {
			BeforeEach(func() {
				_, err := repo.Save("fake-name", "fake-version", "fake-cid")
				Expect(err).ToNot(HaveOccurred())
			})

			It("finds existing stemcell records", func() {
				foundStemcellRecord, found, err := repo.Find("fake-name", "fake-version")
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())
				Expect(foundStemcellRecord).To(Equal(StemcellRecord{
					ID:      "fake-uuid-1",
					Name:    "fake-name",
					Version: "fake-version",
					CID:     "fake-cid",
				}))
			})
		})

		Context("when a stemcell record with the same name and version does not exist", func() {
			It("finds existing stemcell records", func() {
				_, found, err := repo.Find("fake-name", "fake-version")
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeFalse())
			})
		})
	})

	Describe("UpdateCurrent", func() {
		Context("when a stemcell record exists with the same ID", func() {
			BeforeEach(func() {
				fakeUUIDGenerator.GeneratedUUID = "fake-uuid-1"
				_, err := repo.Save("fake-name", "fake-version", "fake-cid")
				Expect(err).ToNot(HaveOccurred())
			})

			It("saves the stemcell record as current stemcell", func() {
				err := repo.UpdateCurrent("fake-uuid-1")
				Expect(err).ToNot(HaveOccurred())

				deploymentState, err := deploymentStateService.Load()
				Expect(err).ToNot(HaveOccurred())

				Expect(deploymentState.CurrentStemcellID).To(Equal("fake-uuid-1"))
			})
		})

		Context("when a stemcell record does not exists with the same ID", func() {
			BeforeEach(func() {
				fakeUUIDGenerator.GeneratedUUID = "fake-uuid-1"
				_, err := repo.Save("fake-name", "fake-version", "fake-cid")
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				err := repo.UpdateCurrent("fake-uuid-2")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Verifying stemcell record exists with id 'fake-uuid-2'"))
			})
		})
	})

	Describe("ClearCurrent", func() {
		Context("when a stemcell record exists with the same ID", func() {
			BeforeEach(func() {
				fakeUUIDGenerator.GeneratedUUID = "fake-uuid-1"
				_, err := repo.Save("fake-name", "fake-version", "fake-cid")
				Expect(err).ToNot(HaveOccurred())

				err = repo.UpdateCurrent("fake-uuid-1")
				Expect(err).ToNot(HaveOccurred())
			})

			It("clears current stemcell id", func() {
				err := repo.ClearCurrent()
				Expect(err).ToNot(HaveOccurred())

				deploymentState, err := deploymentStateService.Load()
				Expect(err).ToNot(HaveOccurred())

				Expect(deploymentState.CurrentStemcellID).To(Equal(""))
			})
		})
	})

	Describe("Delete", func() {
		var (
			firstStemcellRecord  StemcellRecord
			secondStemcellRecord StemcellRecord
			thirdStemcellRecord  StemcellRecord
		)

		BeforeEach(func() {
			var err error
			fakeUUIDGenerator.GeneratedUUID = "fake-uuid-1"
			firstStemcellRecord, err = repo.Save("fake-name1", "fake-version1", "fake-cid1")
			Expect(err).ToNot(HaveOccurred())
			fakeUUIDGenerator.GeneratedUUID = "fake-uuid-2"
			secondStemcellRecord, err = repo.Save("fake-name2", "fake-version2", "fake-cid2")
			Expect(err).ToNot(HaveOccurred())
			fakeUUIDGenerator.GeneratedUUID = "fake-uuid-3"
			thirdStemcellRecord, err = repo.Save("fake-name3", "fake-version3", "fake-cid3")
			Expect(err).ToNot(HaveOccurred())
		})

		It("deletes stemcell record from repo", func() {
			err := repo.Delete(secondStemcellRecord)
			Expect(err).ToNot(HaveOccurred())

			stemcellRecords, err := repo.All()
			Expect(err).ToNot(HaveOccurred())

			Expect(stemcellRecords).To(Equal([]StemcellRecord{
				firstStemcellRecord,
				thirdStemcellRecord,
			}))
		})

		Context("when the stemcell to be deleted is also the current stemcell", func() {
			BeforeEach(func() {
				err := repo.UpdateCurrent(secondStemcellRecord.ID)
				Expect(err).ToNot(HaveOccurred())
			})

			It("clears the current stemcell", func() {
				err := repo.Delete(secondStemcellRecord)
				Expect(err).ToNot(HaveOccurred())

				disks, err := repo.All()
				Expect(err).ToNot(HaveOccurred())
				Expect(disks).To(Equal([]StemcellRecord{
					firstStemcellRecord,
					thirdStemcellRecord,
				}))

				_, found, err := repo.FindCurrent()
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeFalse())
			})
		})
	})

	Describe("FindCurrent", func() {
		Context("when current stemcell exists", func() {
			BeforeEach(func() {
				fakeUUIDGenerator.GeneratedUUID = "fake-guid-1"
				_, err := repo.Save("fake-name", "fake-version-1", "fake-cid-1")
				Expect(err).ToNot(HaveOccurred())

				fakeUUIDGenerator.GeneratedUUID = "fake-guid-2"
				record, err := repo.Save("fake-name", "fake-version-2", "fake-cid-2")
				Expect(err).ToNot(HaveOccurred())

				repo.UpdateCurrent(record.ID)
			})

			It("returns existing stemcell", func() {
				record, found, err := repo.FindCurrent()
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())
				Expect(record).To(Equal(StemcellRecord{
					ID:      "fake-guid-2",
					Name:    "fake-name",
					Version: "fake-version-2",
					CID:     "fake-cid-2",
				}))
			})
		})

		Context("when current stemcell does not exist", func() {
			BeforeEach(func() {
				fakeUUIDGenerator.GeneratedUUID = "fake-guid-1"
				_, err := repo.Save("fake-name", "fake-version", "fake-cid")
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns not found", func() {
				_, found, err := repo.FindCurrent()
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeFalse())
			})
		})

		Context("when there are no stemcells", func() {
			It("returns not found", func() {
				_, found, err := repo.FindCurrent()
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeFalse())
			})
		})
	})
})
