package action_test

import (
	"errors"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent/action"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshcrypto "github.com/cloudfoundry/bosh-utils/crypto"

	fakelogger "github.com/cloudfoundry/bosh-agent/logger/fakes"
	fakeplatform "github.com/cloudfoundry/bosh-agent/platform/fakes"
	fakesettings "github.com/cloudfoundry/bosh-agent/settings/fakes"
	fakeblobstore "github.com/cloudfoundry/bosh-utils/blobstore/fakes"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("SyncDNS", func() {
	var (
		action               SyncDNS
		fakeBlobstore        *fakeblobstore.FakeDigestBlobstore
		fakeSettingsService  *fakesettings.FakeSettingsService
		fakePlatform         *fakeplatform.FakePlatform
		fakeFileSystem       *fakesys.FakeFileSystem
		logger               *fakelogger.FakeLogger
		fakeDNSRecordsString string
	)

	BeforeEach(func() {
		logger = &fakelogger.FakeLogger{}
		fakeBlobstore = &fakeblobstore.FakeDigestBlobstore{}
		fakeSettingsService = &fakesettings.FakeSettingsService{}
		fakePlatform = fakeplatform.NewFakePlatform()
		fakeFileSystem = fakePlatform.GetFs().(*fakesys.FakeFileSystem)

		action = NewSyncDNS(fakeBlobstore, fakeSettingsService, fakePlatform, logger)
	})

	AssertActionIsNotAsynchronous(action)
	AssertActionIsNotPersistent(action)
	AssertActionIsLoggable(action)

	AssertActionIsNotResumable(action)
	AssertActionIsNotCancelable(action)

	Context("#Run", func() {
		var (
			stateFilePath string
			multiDigest   boshcrypto.MultipleDigest
		)

		BeforeEach(func() {
			fakeDNSRecordsString = `
							{
								"version": 2,
								"records": [
									["fake-ip0", "fake-name0"],
									["fake-ip1", "fake-name1"]
								],
								"record_keys": ["id", "instance_group", "az", "network", "deployment", "ip"],
								"record_infos": [
									["id-1", "instance-group-1", "az1", "network1", "deployment1", "ip1"]
								]
							}`
			multiDigest = boshcrypto.MustNewMultipleDigest(boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, "fake-fingerprint"))
			err := fakeFileSystem.WriteFileString("fake-blobstore-file-path", fakeDNSRecordsString)
			Expect(err).ToNot(HaveOccurred())

			fakeBlobstore.GetReturns("fake-blobstore-file-path", nil)
			stateFilePath = filepath.Join(fakePlatform.GetDirProvider().InstanceDNSDir(), "records.json")
		})

		Context("when local DNS state version is >= Run's version", func() {
			BeforeEach(func() {
				fakeBlobstore.GetReturns("", errors.New("fake-blobstore-get-error"))
			})

			Context("when the version equals the Run's version", func() {
				BeforeEach(func() {
					err := fakeFileSystem.WriteFileString(stateFilePath, `{"version": 2}`)
					Expect(err).ToNot(HaveOccurred())

					fakeFileSystem.WriteFileError = errors.New("fake-write-error")
				})

				It("returns with no error and does no writes and no gets to blobstore", func() {
					_, err := action.Run("fake-blobstore-id", multiDigest, 2)
					Expect(err).ToNot(HaveOccurred())
				})
			})

			Context("when the version > the Run's version", func() {
				BeforeEach(func() {
					err := fakeFileSystem.WriteFileString(stateFilePath, `{"version": 3}`)
					Expect(err).ToNot(HaveOccurred())

					fakeFileSystem.WriteFileError = errors.New("fake-write-error")
				})

				It("returns error", func() {
					_, err := action.Run("fake-blobstore-id", multiDigest, 2)
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

		Context("when the version in the blob does not match the version director supplied", func() {
			It("returns an error", func() {
				_, err := action.Run("fake-blobstore-id", multiDigest, 3)
				Expect(err).To(MatchError("version from unpacked dns blob does not match version supplied by director"))
			})
		})

		Context("when local DNS state version is < Run's version", func() {
			BeforeEach(func() {
				err := fakeFileSystem.WriteFileString(stateFilePath, `{"version": 1}`)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("when blobstore contains DNS records", func() {
				It("accesses the blobstore and fetches DNS records", func() {
					response, err := action.Run("fake-blobstore-id", multiDigest, 2)
					Expect(err).ToNot(HaveOccurred())
					Expect(response).To(Equal("synced"))

					Expect(fakeBlobstore.GetCallCount()).To(Equal(1))
					blobID, fingerPrint := fakeBlobstore.GetArgsForCall(0)
					Expect(blobID).To(Equal("fake-blobstore-id"))
					Expect(fingerPrint).To(Equal(multiDigest))
				})

				It("reads the DNS records from the blobstore file", func() {
					response, err := action.Run("fake-blobstore-id", multiDigest, 2)
					Expect(err).ToNot(HaveOccurred())
					Expect(response).To(Equal("synced"))

					Expect(fakeFileSystem.ReadFileError).ToNot(HaveOccurred())
				})

				It("fails reading the DNS records from the blobstore file", func() {
					fakeFileSystem.RegisterReadFileError("fake-blobstore-file-path", errors.New("fake-error"))

					response, err := action.Run("fake-blobstore-id", multiDigest, 2)
					Expect(err).To(HaveOccurred())
					Expect(response).To(Equal(""))
					Expect(err.Error()).To(ContainSubstring("reading fake-blobstore-file-path from blobstore"))
					Expect(fakeFileSystem.FileExists("fake-blobstore-file-path")).To(BeFalse())
				})

				It("deletes the file once read", func() {
					_, err := action.Run("fake-blobstore-id", multiDigest, 2)
					Expect(err).ToNot(HaveOccurred())

					Expect(fakeFileSystem.FileExists("fake-blobstore-file-path")).To(BeFalse())
				})

				It("logs when the dns blob file can't be deleted", func() {
					fakeFileSystem.RemoveAllStub = func(path string) error {
						if path == "fake-blobstore-file-path" {
							return errors.New("fake-file-path-error")
						}
						return nil
					}
					_, err := action.Run("fake-blobstore-id", multiDigest, 2)
					Expect(err).ToNot(HaveOccurred())

					tag, message, _ := logger.ErrorArgsForCall(0)
					Expect(tag).To(Equal("Sync DNS action"))
					Expect(message).To(Equal("Failed to remove dns blob file at path 'fake-blobstore-file-path'"))
				})

				It("saves DNS records to the platform", func() {
					response, err := action.Run("fake-blobstore-id", multiDigest, 2)
					Expect(err).ToNot(HaveOccurred())
					Expect(response).To(Equal("synced"))

					Expect(fakePlatform.SaveDNSRecordsError).To(BeNil())
					Expect(fakePlatform.SaveDNSRecordsDNSRecords).To(Equal(boshsettings.DNSRecords{
						Version: 2,
						Records: [][2]string{
							{"fake-ip0", "fake-name0"},
							{"fake-ip1", "fake-name1"},
						},
					}))
				})

				Context("when there is no local DNS state", func() {
					BeforeEach(func() {
						err := fakeFileSystem.RemoveAll(stateFilePath)
						Expect(err).NotTo(HaveOccurred())
					})

					It("saves DNS records to the platform", func() {
						Expect(fakeFileSystem.FileExists(stateFilePath)).To(BeFalse())

						response, err := action.Run("fake-blobstore-id", multiDigest, 2)
						Expect(err).ToNot(HaveOccurred())
						Expect(response).To(Equal("synced"))

						Expect(fakePlatform.SaveDNSRecordsError).To(BeNil())
						Expect(fakePlatform.SaveDNSRecordsDNSRecords).To(Equal(boshsettings.DNSRecords{
							Version: 2,
							Records: [][2]string{
								{"fake-ip0", "fake-name0"},
								{"fake-ip1", "fake-name1"},
							},
						}))
					})
				})

				Context("when there is an error reading the local dns state", func() {
					BeforeEach(func() {
						err := fakeFileSystem.WriteFileString(stateFilePath, `{"version": 1}`)
						Expect(err).ToNot(HaveOccurred())

						fakeFileSystem.RegisterReadFileError(stateFilePath, errors.New("fake-read-error"))
					})

					It("saves DNS records to the platform", func() {
						response, err := action.Run("fake-blobstore-id", multiDigest, 2)
						Expect(err).ToNot(HaveOccurred())
						Expect(response).To(Equal("synced"))

						Expect(fakePlatform.SaveDNSRecordsError).To(BeNil())
						Expect(fakePlatform.SaveDNSRecordsDNSRecords).To(Equal(boshsettings.DNSRecords{
							Version: 2,
							Records: [][2]string{
								{"fake-ip0", "fake-name0"},
								{"fake-ip1", "fake-name1"},
							},
						}))
					})
				})

				Context("when the the local dns state is corrupt", func() {
					BeforeEach(func() {
						err := fakeFileSystem.WriteFileString(stateFilePath, "hot-trash")
						Expect(err).ToNot(HaveOccurred())
					})

					It("saves DNS records to the platform", func() {
						response, err := action.Run("fake-blobstore-id", multiDigest, 2)
						Expect(err).ToNot(HaveOccurred())
						Expect(response).To(Equal("synced"))

						Expect(fakePlatform.SaveDNSRecordsError).To(BeNil())
						Expect(fakePlatform.SaveDNSRecordsDNSRecords).To(Equal(boshsettings.DNSRecords{
							Version: 2,
							Records: [][2]string{
								{"fake-ip0", "fake-name0"},
								{"fake-ip1", "fake-name1"},
							},
						}))
					})
				})

				Context("local DNS state operations", func() {
					Context("when there is no local DNS state", func() {
						BeforeEach(func() {
							err := fakeFileSystem.RemoveAll(stateFilePath)
							Expect(err).NotTo(HaveOccurred())
						})

						It("runs successfully and creates a new state file", func() {
							Expect(fakeFileSystem.FileExists(stateFilePath)).To(BeFalse())

							response, err := action.Run("fake-blobstore-id", multiDigest, 2)
							Expect(err).ToNot(HaveOccurred())
							Expect(response).To(Equal("synced"))

							contents, err := fakeFileSystem.ReadFile(stateFilePath)
							Expect(err).ToNot(HaveOccurred())
							Expect(contents).To(MatchJSON(`
							{
								"version": 2,
								"records": [
									["fake-ip0", "fake-name0"],
									["fake-ip1", "fake-name1"]
								],
								"record_keys": ["id", "instance_group", "az", "network", "deployment", "ip"],
								"record_infos": [
									["id-1", "instance-group-1", "az1", "network1", "deployment1", "ip1"]
								]
							}`))
						})
					})

					Context("when saving fails", func() {
						BeforeEach(func() {
							fakeFileSystem.WriteFileError = errors.New("fake-write-error")
						})

						It("returns an error", func() {
							_, err := action.Run("fake-blobstore-id", multiDigest, 2)
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(ContainSubstring("saving local DNS state"))
						})
					})
				})

				Context("when DNS records is invalid", func() {
					BeforeEach(func() {
						err := fakeFileSystem.WriteFileString("fake-blobstore-file-path", "")
						Expect(err).ToNot(HaveOccurred())
					})

					It("fails unmarshalling the DNS records from the file", func() {
						_, err := action.Run("fake-blobstore-id", multiDigest, 2)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("unmarshalling DNS records"))
					})
				})

				Context("when platform fails to save DNS records", func() {
					BeforeEach(func() {
						fakePlatform.SaveDNSRecordsError = errors.New("fake-error")

						err := fakeFileSystem.WriteFileString(stateFilePath, `{"version": 1}`)
						Expect(err).ToNot(HaveOccurred())
					})

					It("fails to save DNS records on the platform", func() {
						_, err := action.Run("fake-blobstore-id", multiDigest, 2)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("saving DNS records"))
					})

					It("should not update the records.json", func() {
						_, err := action.Run("fake-blobstore-id", multiDigest, 2)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("saving DNS records"))

						contents, err := fakeFileSystem.ReadFile(stateFilePath)
						Expect(err).ToNot(HaveOccurred())
						Expect(contents).To(MatchJSON(`{"version": 1}`))
					})
				})
			})

			Context("when blobstore does not contain DNS records", func() {
				BeforeEach(func() {
					fakeBlobstore.GetReturns("fake-blobstore-file-path-does-not-exist", nil)
				})

				Context("when blobstore returns an error", func() {
					It("fails with an wrapped error", func() {
						_, err := action.Run("fake-blobstore-id", multiDigest, 2)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("reading fake-blobstore-file-path-does-not-exist from blobstore"))
					})
				})

				Context("when blobstore returns a file that cannot be read", func() {
					BeforeEach(func() {
						fakeBlobstore.GetReturns("fake-blobstore-file-path", nil)

						fakeFileSystem.RemoveAllStub = func(path string) error {
							if path == "fake-blobstore-file-path" {
								return errors.New("fake-remove-all-error")
							}
							return nil
						}
					})

					Context("when file removal failed", func() {
						It("logs error", func() {
							_, _ = action.Run("fake-blobstore-id", multiDigest, 2)
							tag, message, _ := logger.ErrorArgsForCall(0)
							Expect(tag).To(Equal("Sync DNS action"))
							Expect(message).To(Equal("Failed to remove dns blob file at path 'fake-blobstore-file-path'"))
						})
					})
				})
			})
		})
	})
})
