package stemcell_test

import (
	. "github.com/cloudfoundry/bosh-cli/stemcell"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"errors"

	bicloud "github.com/cloudfoundry/bosh-cli/cloud"
	fakebicloud "github.com/cloudfoundry/bosh-cli/cloud/fakes"
	biconfig "github.com/cloudfoundry/bosh-cli/config"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
)

var _ = Describe("CloudStemcell", func() {
	var (
		stemcellRepo      biconfig.StemcellRepo
		fakeUUIDGenerator *fakeuuid.FakeGenerator
		fakeCloud         *fakebicloud.FakeCloud
		cloudStemcell     CloudStemcell
	)

	BeforeEach(func() {
		stemcellRecord := biconfig.StemcellRecord{
			CID:     "fake-stemcell-cid",
			Name:    "fake-stemcell-name",
			Version: "fake-stemcell-version",
		}
		fs := fakesys.NewFakeFileSystem()
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fakeUUIDGenerator = &fakeuuid.FakeGenerator{}
		deploymentStateService := biconfig.NewFileSystemDeploymentStateService(fs, fakeUUIDGenerator, logger, "/fake/path")
		stemcellRepo = biconfig.NewStemcellRepo(deploymentStateService, fakeUUIDGenerator)
		fakeCloud = fakebicloud.NewFakeCloud()
		cloudStemcell = NewCloudStemcell(stemcellRecord, stemcellRepo, fakeCloud)
	})

	Describe("PromoteAsCurrent", func() {
		Context("when stemcell is in the repo", func() {
			BeforeEach(func() {
				fakeUUIDGenerator.GeneratedUUID = "fake-stemcell-id"
				_, err := stemcellRepo.Save("fake-stemcell-name", "fake-stemcell-version", "fake-stemcell-cid")
				Expect(err).ToNot(HaveOccurred())
			})

			It("sets stemcell as current in the repo", func() {
				err := cloudStemcell.PromoteAsCurrent()
				Expect(err).ToNot(HaveOccurred())

				currentStemcell, found, err := stemcellRepo.FindCurrent()
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())
				Expect(currentStemcell).To(Equal(biconfig.StemcellRecord{
					ID:      "fake-stemcell-id",
					CID:     "fake-stemcell-cid",
					Name:    "fake-stemcell-name",
					Version: "fake-stemcell-version",
				}))
			})
		})

		Context("when stemcell is not in the repo", func() {
			It("returns an error", func() {
				err := cloudStemcell.PromoteAsCurrent()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Stemcell does not exist in repo"))
			})
		})
	})

	Describe("Delete", func() {
		It("deletes stemcell from cloud", func() {
			err := cloudStemcell.Delete()
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeCloud.DeleteStemcellInputs).To(Equal([]fakebicloud.DeleteStemcellInput{
				{
					StemcellCID: "fake-stemcell-cid",
				},
			}))
		})

		It("deletes stemcell from repo", func() {
			_, err := stemcellRepo.Save("fake-stemcell-name", "fake-stemcell-version", "fake-stemcell-cid")
			Expect(err).ToNot(HaveOccurred())

			err = cloudStemcell.Delete()
			Expect(err).ToNot(HaveOccurred())
			stemcellRecords, err := stemcellRepo.All()
			Expect(stemcellRecords).To(BeEmpty())
		})

		Context("when deleted stemcell is the current stemcell", func() {
			BeforeEach(func() {
				stemcellRecord, err := stemcellRepo.Save("fake-stemcell-name", "fake-stemcell-version", "fake-stemcell-cid")
				Expect(err).ToNot(HaveOccurred())

				err = stemcellRepo.UpdateCurrent(stemcellRecord.ID)
				Expect(err).ToNot(HaveOccurred())
			})

			It("clears current stemcell in the repo", func() {
				err := cloudStemcell.Delete()
				Expect(err).ToNot(HaveOccurred())

				_, found, err := stemcellRepo.FindCurrent()
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeFalse())
			})
		})

		Context("when deleting stemcell in the cloud fails", func() {
			BeforeEach(func() {
				fakeCloud.DeleteStemcellErr = errors.New("fake-delete-stemcell-error")
			})

			It("returns an error", func() {
				err := cloudStemcell.Delete()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-delete-stemcell-error"))
			})
		})

		Context("when deleting stemcell in the cloud fails with StemcellNotFoundError", func() {
			var deleteErr = bicloud.NewCPIError("delete_stemcell", bicloud.CmdError{
				Type:    bicloud.StemcellNotFoundError,
				Message: "fake-stemcell-not-found-message",
			})

			BeforeEach(func() {
				stemcellRecord, err := stemcellRepo.Save("fake-stemcell-name", "fake-stemcell-version", "fake-stemcell-cid")
				Expect(err).ToNot(HaveOccurred())

				err = stemcellRepo.UpdateCurrent(stemcellRecord.ID)
				Expect(err).ToNot(HaveOccurred())

				fakeCloud.DeleteStemcellErr = deleteErr
			})

			It("deletes stemcell in the cloud", func() {
				err := cloudStemcell.Delete()
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(deleteErr))

				Expect(fakeCloud.DeleteStemcellInputs).To(Equal([]fakebicloud.DeleteStemcellInput{
					{
						StemcellCID: "fake-stemcell-cid",
					},
				}))
			})

			It("deletes stemcell in the disk repo", func() {
				err := cloudStemcell.Delete()
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(deleteErr))

				stemcellRecords, err := stemcellRepo.All()
				Expect(stemcellRecords).To(BeEmpty())
			})

			It("clears current stemcell in the stemcell repo", func() {
				err := cloudStemcell.Delete()
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(deleteErr))

				_, found, err := stemcellRepo.FindCurrent()
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeFalse())
			})
		})
	})
})
