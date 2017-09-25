package action_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent/action"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"

	fakelogger "github.com/cloudfoundry/bosh-agent/logger/fakes"
	fakeplatform "github.com/cloudfoundry/bosh-agent/platform/fakes"
	fakesettings "github.com/cloudfoundry/bosh-agent/settings/fakes"
	fakeblobstore "github.com/cloudfoundry/bosh-utils/blobstore/fakes"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("SyncDNS", func() {
	var (
		syncDNS             SyncDNS
		fakeBlobstore       *fakeblobstore.FakeBlobstore
		fakeSettingsService *fakesettings.FakeSettingsService
		fakePlatform        *fakeplatform.FakePlatform
		fakeFileSystem      *fakesys.FakeFileSystem
		logger              *fakelogger.FakeLogger
	)

	BeforeEach(func() {
		logger = &fakelogger.FakeLogger{}
		fakeBlobstore = fakeblobstore.NewFakeBlobstore()
		fakeSettingsService = &fakesettings.FakeSettingsService{}
		fakePlatform = fakeplatform.NewFakePlatform()
		fakeFileSystem = fakePlatform.GetFs().(*fakesys.FakeFileSystem)

		syncDNS = NewSyncDNS(fakeBlobstore, fakeSettingsService, fakePlatform, logger)
	})

	It("returns IsAsynchronous false", func() {
		async := syncDNS.IsAsynchronous()
		Expect(async).To(BeFalse())
	})

	It("returns IsPersistent false", func() {
		persistent := syncDNS.IsPersistent()
		Expect(persistent).To(BeFalse())
	})

	It("returns error 'Not supported' when resumed", func() {
		result, err := syncDNS.Resume()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Not supported"))
		Expect(result).To(BeNil())
	})

	It("returns error 'Not supported' when canceled", func() {
		err := syncDNS.Cancel()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Not supported"))
	})

	Context("when sync_dns is recieved", func() {
		Context("when blobstore contains DNS records", func() {
			BeforeEach(func() {
				fakeDNSRecordsString := `
				{
					"records": [
						["fake-ip0", "fake-name0"],
						["fake-ip1", "fake-name1"]
					]
				}`

				err := fakeFileSystem.WriteFileString("fake-blobstore-file-path", fakeDNSRecordsString)
				Expect(err).ToNot(HaveOccurred())

				fakeBlobstore.GetFileName = "fake-blobstore-file-path"
			})

			It("accesses the blobstore and fetches DNS records", func() {
				response, err := syncDNS.Run("fake-blobstore-id", "fake-fingerprint")
				Expect(err).ToNot(HaveOccurred())
				Expect(response).To(Equal("synced"))

				Expect(fakeBlobstore.GetBlobIDs).To(ContainElement("fake-blobstore-id"))
				Expect(fakeBlobstore.GetFingerprints).To(ContainElement("fake-fingerprint"))

				Expect(fakeBlobstore.GetError).ToNot(HaveOccurred())
				Expect(fakeBlobstore.GetFileName).ToNot(Equal(""))
			})

			It("reads the DNS records from the blobstore file", func() {
				response, err := syncDNS.Run("fake-blobstore-id", "fake-fingerprint")
				Expect(err).ToNot(HaveOccurred())
				Expect(response).To(Equal("synced"))

				Expect(fakeBlobstore.GetError).ToNot(HaveOccurred())
				Expect(fakeBlobstore.GetFileName).To(Equal("fake-blobstore-file-path"))
				Expect(fakeFileSystem.ReadFileError).ToNot(HaveOccurred())
			})

			It("fails reading the DNS records from the blobstore file", func() {
				fakeFileSystem.RegisterReadFileError("fake-blobstore-file-path", errors.New("fake-error"))

				response, err := syncDNS.Run("fake-blobstore-id", "fake-fingerprint")
				Expect(err).To(HaveOccurred())
				Expect(response).To(Equal(""))
				Expect(err.Error()).To(ContainSubstring("Reading fileName"))
			})

			It("deletes the file once read", func() {
				_, err := syncDNS.Run("fake-blobstore-id", "fake-fingerprint")
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeFileSystem.FileExists("fake-blobstore-file-path")).To(BeFalse())
			})

			It("logs when the dns blob file can't be deleted", func() {
				fakeFileSystem.RegisterRemoveAllError("fake-blobstore-file-path", errors.New("fake-file-path-error"))
				_, err := syncDNS.Run("fake-blobstore-id", "fake-fingerprint")
				Expect(err).ToNot(HaveOccurred())

				tag, message, _ := logger.InfoArgsForCall(0)
				Expect(tag).To(Equal("Sync DNS action"))
				Expect(message).To(Equal("Failed to remove dns blob file at path 'fake-blobstore-file-path'"))
			})

			It("saves DNS records to the platform", func() {
				response, err := syncDNS.Run("fake-blobstore-id", "fake-fingerprint")
				Expect(err).ToNot(HaveOccurred())
				Expect(response).To(Equal("synced"))

				Expect(fakePlatform.SaveDNSRecordsError).To(BeNil())
				Expect(fakePlatform.SaveDNSRecordsDNSRecords).To(Equal(boshsettings.DNSRecords{
					Records: [][2]string{
						{"fake-ip0", "fake-name0"},
						{"fake-ip1", "fake-name1"},
					},
				}))
			})

			Context("when DNS records is invalid", func() {
				BeforeEach(func() {
					err := fakeFileSystem.WriteFileString("fake-blobstore-file-path", "")
					Expect(err).ToNot(HaveOccurred())
				})

				It("fails unmarshalling the DNS records from the file", func() {
					_, err := syncDNS.Run("fake-blobstore-id", "fake-fingerprint")
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Unmarshalling DNS records"))
				})
			})

			Context("when platform fails to save DNS records", func() {
				BeforeEach(func() {
					fakePlatform.SaveDNSRecordsError = errors.New("fake-error")
				})

				It("fails to save DNS records on the platform", func() {
					_, err := syncDNS.Run("fake-blobstore-id", "fake-fingerprint")
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Saving DNS records in platform"))
				})
			})
		})

		Context("when blobstore does not contain DNS records", func() {
			It("fails getting the DNS records", func() {
				_, err := syncDNS.Run("fake-blobstore-id", "fake-fingerprint")
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
