package stemcell_test

import (
	. "github.com/cloudfoundry/bosh-cli/stemcell"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"errors"

	biconfig "github.com/cloudfoundry/bosh-cli/config"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"

	fakebicloud "github.com/cloudfoundry/bosh-cli/cloud/fakes"
	fakebistemcell "github.com/cloudfoundry/bosh-cli/stemcell/stemcellfakes"
	fakebiui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("Manager", func() {
	var (
		stemcellRepo        biconfig.StemcellRepo
		fakeUUIDGenerator   *fakeuuid.FakeGenerator
		manager             Manager
		fs                  *fakesys.FakeFileSystem
		reader              *fakebistemcell.FakeStemcellReader
		fakeCloud           *fakebicloud.FakeCloud
		fakeStage           *fakebiui.FakeStage
		stemcellTarballPath string
		tempExtractionDir   string

		expectedExtractedStemcell ExtractedStemcell
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		reader = fakebistemcell.NewFakeReader()
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fakeUUIDGenerator = &fakeuuid.FakeGenerator{}
		deploymentStateService := biconfig.NewFileSystemDeploymentStateService(fs, fakeUUIDGenerator, logger, "/fake/path")
		fakeUUIDGenerator.GeneratedUUID = "fake-stemcell-id-1"
		stemcellRepo = biconfig.NewStemcellRepo(deploymentStateService, fakeUUIDGenerator)
		fakeStage = fakebiui.NewFakeStage()
		fakeCloud = fakebicloud.NewFakeCloud()
		manager = NewManager(stemcellRepo, fakeCloud)
		stemcellTarballPath = "/stemcell/tarball/path"
		tempExtractionDir = "/path/to/dest"
		fs.TempDirDir = tempExtractionDir

		expectedExtractedStemcell = NewExtractedStemcell(
			Manifest{
				Name:    "fake-stemcell-name",
				Version: "fake-stemcell-version",
				CloudProperties: biproperty.Map{
					"fake-prop-key": "fake-prop-value",
				},
			},
			tempExtractionDir,
			nil,
			fs,
		)
		reader.SetReadBehavior(stemcellTarballPath, tempExtractionDir, expectedExtractedStemcell, nil)
	})

	Describe("Upload", func() {
		var (
			expectedCloudStemcell CloudStemcell
		)

		BeforeEach(func() {
			fakeCloud.CreateStemcellCID = "fake-stemcell-cid"
			stemcellRecord := biconfig.StemcellRecord{
				CID:     "fake-stemcell-cid",
				Name:    "fake-stemcell-name",
				Version: "fake-stemcell-version",
			}
			expectedCloudStemcell = NewCloudStemcell(stemcellRecord, stemcellRepo, fakeCloud)
		})

		It("uploads the stemcell to the infrastructure and returns the cid", func() {
			cloudStemcell, err := manager.Upload(expectedExtractedStemcell, fakeStage)
			Expect(err).ToNot(HaveOccurred())
			Expect(cloudStemcell).To(Equal(expectedCloudStemcell))

			Expect(fakeCloud.CreateStemcellInputs).To(Equal([]fakebicloud.CreateStemcellInput{
				{
					ImagePath: tempExtractionDir + "/image",
					CloudProperties: biproperty.Map{
						"fake-prop-key": "fake-prop-value",
					},
				},
			}))
		})

		It("saves the stemcell record in the stemcellRepo", func() {
			cloudStemcell, err := manager.Upload(expectedExtractedStemcell, fakeStage)
			Expect(err).ToNot(HaveOccurred())
			Expect(cloudStemcell).To(Equal(expectedCloudStemcell))

			stemcellRecords, err := stemcellRepo.All()
			Expect(stemcellRecords).To(Equal([]biconfig.StemcellRecord{
				{
					ID:      "fake-stemcell-id-1",
					Name:    "fake-stemcell-name",
					Version: "fake-stemcell-version",
					CID:     "fake-stemcell-cid",
				},
			}))
		})

		It("prints uploading ui stage", func() {
			_, err := manager.Upload(expectedExtractedStemcell, fakeStage)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeStage.PerformCalls).To(Equal([]*fakebiui.PerformCall{
				{Name: "Uploading stemcell 'fake-stemcell-name/fake-stemcell-version'"},
			}))
		})

		It("when the upload fails, prints failed uploading ui stage", func() {
			fakeCloud.CreateStemcellErr = errors.New("fake-create-error")
			_, err := manager.Upload(expectedExtractedStemcell, fakeStage)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-create-error"))

			Expect(fakeStage.PerformCalls[0].Name).To(Equal("Uploading stemcell 'fake-stemcell-name/fake-stemcell-version'"))
			Expect(fakeStage.PerformCalls[0].Error).To(HaveOccurred())
			Expect(fakeStage.PerformCalls[0].Error.Error()).To(Equal("creating stemcell (fake-stemcell-name fake-stemcell-version): fake-create-error"))
		})

		It("when the stemcellRepo save fails, logs uploading start and failure events to the eventLogger", func() {
			fs.WriteFileError = errors.New("fake-save-error")
			_, err := manager.Upload(expectedExtractedStemcell, fakeStage)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-save-error"))

			Expect(fakeStage.PerformCalls[0].Name).To(Equal("Uploading stemcell 'fake-stemcell-name/fake-stemcell-version'"))
			Expect(fakeStage.PerformCalls[0].Error).To(HaveOccurred())
			Expect(fakeStage.PerformCalls[0].Error.Error()).To(MatchRegexp("Finding existing stemcell record in repo: .*fake-save-error.*"))
		})

		Context("when the stemcell record exists in the stemcellRepo (having been previously uploaded)", func() {
			var (
				foundStemcellRecord biconfig.StemcellRecord
			)

			BeforeEach(func() {
				var err error
				foundStemcellRecord, err = stemcellRepo.Save("fake-stemcell-name", "fake-stemcell-version", "fake-existing-cid")
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns the existing cloud stemcell", func() {
				stemcell, err := manager.Upload(expectedExtractedStemcell, fakeStage)
				Expect(err).ToNot(HaveOccurred())
				foundStemcell := NewCloudStemcell(foundStemcellRecord, stemcellRepo, fakeCloud)
				Expect(stemcell).To(Equal(foundStemcell))
			})

			It("does not re-upload the stemcell to the infrastructure", func() {
				_, err := manager.Upload(expectedExtractedStemcell, fakeStage)
				Expect(err).ToNot(HaveOccurred())
				Expect(fakeCloud.CreateStemcellInputs).To(HaveLen(0))
			})

			It("logs skipping uploading events to the eventLogger", func() {
				_, err := manager.Upload(expectedExtractedStemcell, fakeStage)
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeStage.PerformCalls[0].Name).To(Equal("Uploading stemcell 'fake-stemcell-name/fake-stemcell-version'"))
				Expect(fakeStage.PerformCalls[0].SkipError).To(HaveOccurred())
				Expect(fakeStage.PerformCalls[0].SkipError.Error()).To(MatchRegexp("Stemcell already uploaded: Found stemcell: .*fake-existing-cid.*"))
			})
		})
	})

	Describe("FindCurrent", func() {
		Context("when stemcell already exists in stemcell repo", func() {
			BeforeEach(func() {
				stemcellRecord, err := stemcellRepo.Save("fake-stemcell-name", "fake-stemcell-version", "fake-existing-stemcell-cid")
				Expect(err).ToNot(HaveOccurred())

				err = stemcellRepo.UpdateCurrent(stemcellRecord.ID)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns the existing stemcell", func() {
				stemcells, err := manager.FindCurrent()
				Expect(err).ToNot(HaveOccurred())
				Expect(stemcells).To(HaveLen(1))
				Expect(stemcells[0].CID()).To(Equal("fake-existing-stemcell-cid"))
			})
		})

		Context("when stemcell does not exists in stemcell repo", func() {
			It("returns false", func() {
				stemcells, err := manager.FindCurrent()
				Expect(err).ToNot(HaveOccurred())
				Expect(stemcells).To(BeEmpty())
			})
		})

		Context("when reading stemcell repo fails", func() {
			BeforeEach(func() {
				fs.WriteFileString("/fake/path", "{}")
				fs.ReadFileError = errors.New("fake-read-error")
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
			firstStemcell  CloudStemcell
			secondStemcell CloudStemcell
		)

		BeforeEach(func() {
			fakeUUIDGenerator.GeneratedUUID = "fake-stemcell-id-1"
			firstStemcellRecord, err := stemcellRepo.Save("fake-stemcell-name-1", "fake-stemcell-version-1", "fake-stemcell-cid-1")
			Expect(err).ToNot(HaveOccurred())
			firstStemcell = NewCloudStemcell(firstStemcellRecord, stemcellRepo, fakeCloud)

			fakeUUIDGenerator.GeneratedUUID = "fake-stemcell-id-2"
			_, err = stemcellRepo.Save("fake-stemcell-name-2", "fake-stemcell-version-2", "fake-stemcell-cid-2")
			Expect(err).ToNot(HaveOccurred())
			err = stemcellRepo.UpdateCurrent("fake-stemcell-id-2")
			Expect(err).ToNot(HaveOccurred())

			fakeUUIDGenerator.GeneratedUUID = "fake-stemcell-id-3"
			secondStemcellRecord, err := stemcellRepo.Save("fake-stemcell-name-3", "fake-stemcell-version-3", "fake-stemcell-cid-3")
			Expect(err).ToNot(HaveOccurred())
			secondStemcell = NewCloudStemcell(secondStemcellRecord, stemcellRepo, fakeCloud)
		})

		It("returns unused stemcells", func() {
			stemcells, err := manager.FindUnused()
			Expect(err).ToNot(HaveOccurred())
			Expect(stemcells).To(Equal([]CloudStemcell{
				firstStemcell,
				secondStemcell,
			}))
		})
	})

	Describe("DeleteUnused", func() {
		var (
			secondStemcellRecord biconfig.StemcellRecord
		)
		BeforeEach(func() {
			fakeUUIDGenerator.GeneratedUUID = "fake-stemcell-id-1"
			_, err := stemcellRepo.Save("fake-stemcell-name-1", "fake-stemcell-version-1", "fake-stemcell-cid-1")
			Expect(err).ToNot(HaveOccurred())

			fakeUUIDGenerator.GeneratedUUID = "fake-stemcell-id-2"
			secondStemcellRecord, err = stemcellRepo.Save("fake-stemcell-name-2", "fake-stemcell-version-2", "fake-stemcell-cid-2")
			Expect(err).ToNot(HaveOccurred())
			err = stemcellRepo.UpdateCurrent(secondStemcellRecord.ID)
			Expect(err).ToNot(HaveOccurred())

			fakeUUIDGenerator.GeneratedUUID = "fake-stemcell-id-3"
			_, err = stemcellRepo.Save("fake-stemcell-name-3", "fake-stemcell-version-3", "fake-stemcell-cid-3")
			Expect(err).ToNot(HaveOccurred())
		})

		It("deletes unused stemcells", func() {
			err := manager.DeleteUnused(fakeStage)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeCloud.DeleteStemcellInputs).To(Equal([]fakebicloud.DeleteStemcellInput{
				{StemcellCID: "fake-stemcell-cid-1"},
				{StemcellCID: "fake-stemcell-cid-3"},
			}))

			Expect(fakeStage.PerformCalls).To(Equal([]*fakebiui.PerformCall{
				{Name: "Deleting unused stemcell 'fake-stemcell-cid-1'"},
				{Name: "Deleting unused stemcell 'fake-stemcell-cid-3'"},
			}))

			currentRecord, found, err := stemcellRepo.FindCurrent()
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(currentRecord).To(Equal(secondStemcellRecord))

			records, err := stemcellRepo.All()
			Expect(err).ToNot(HaveOccurred())
			Expect(records).To(Equal([]biconfig.StemcellRecord{
				secondStemcellRecord,
			}))
		})
	})
})
