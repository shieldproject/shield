package blobstore_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshblob "github.com/cloudfoundry/bosh-utils/blobstore"
	fakeblob "github.com/cloudfoundry/bosh-utils/blobstore/fakes"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("retryableBlobstore", func() {
	var (
		innerBlobstore     *fakeblob.FakeBlobstore
		logger             boshlog.Logger
		retryableBlobstore boshblob.Blobstore
	)

	BeforeEach(func() {
		innerBlobstore = &fakeblob.FakeBlobstore{}
		logger = boshlog.NewLogger(boshlog.LevelNone)
		retryableBlobstore = boshblob.NewRetryableBlobstore(innerBlobstore, 3, logger)
	})

	Describe("Get", func() {
		Context("when inner blobstore succeeds before maximum number of get tries (first time)", func() {
			It("returns path without an error", func() {
				innerBlobstore.GetFileName = "fake-path"

				path, err := retryableBlobstore.Get("fake-blob-id", "fake-fingerprint")
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal("fake-path"))

				Expect(innerBlobstore.GetBlobIDs).To(Equal([]string{"fake-blob-id"}))
				Expect(innerBlobstore.GetFingerprints).To(Equal([]string{"fake-fingerprint"}))
			})
		})

		Context("when inner blobstore succeed exactly at maximum number of get tries", func() {
			It("returns path without an error", func() {
				innerBlobstore.GetFileNames = []string{"", "", "fake-last-path"}
				innerBlobstore.GetErrs = []error{
					errors.New("fake-get-err-1"),
					errors.New("fake-get-err-2"),
					nil,
				}

				path, err := retryableBlobstore.Get("fake-blob-id", "fake-fingerprint")
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal("fake-last-path"))

				Expect(innerBlobstore.GetBlobIDs).To(Equal(
					[]string{"fake-blob-id", "fake-blob-id", "fake-blob-id"},
				))

				Expect(innerBlobstore.GetFingerprints).To(Equal(
					[]string{"fake-fingerprint", "fake-fingerprint", "fake-fingerprint"},
				))
			})
		})

		Context("when inner blobstore does not succeed before maximum number of get tries", func() {
			It("returns last try error from inner blobstore", func() {
				innerBlobstore.GetFileNames = []string{"", "", ""}
				innerBlobstore.GetErrs = []error{
					errors.New("fake-get-err-1"),
					errors.New("fake-get-err-2"),
					errors.New("fake-last-get-err"),
				}

				_, err := retryableBlobstore.Get("fake-blob-id", "fake-fingerprint")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-last-get-err"))

				Expect(innerBlobstore.GetBlobIDs).To(Equal(
					[]string{"fake-blob-id", "fake-blob-id", "fake-blob-id"},
				))

				Expect(innerBlobstore.GetFingerprints).To(Equal(
					[]string{"fake-fingerprint", "fake-fingerprint", "fake-fingerprint"},
				))
			})
		})
	})

	Describe("CleanUp", func() {
		It("delegates to inner blobstore to clean up", func() {
			err := retryableBlobstore.CleanUp("/some/file")
			Expect(err).ToNot(HaveOccurred())

			Expect(innerBlobstore.CleanUpFileName).To(Equal("/some/file"))
		})

		It("returns error if inner blobstore cleaning up fails", func() {
			innerBlobstore.CleanUpErr = errors.New("fake-clean-up-error")

			err := retryableBlobstore.CleanUp("/some/file")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-clean-up-error"))
		})
	})

	Describe("Delete", func() {
		It("delegates to inner blobstore to clean up", func() {
			err := retryableBlobstore.Delete("some-blob")
			Expect(err).ToNot(HaveOccurred())

			Expect(innerBlobstore.DeleteBlobID).To(Equal("some-blob"))
		})

		It("returns error if inner blobstore cleaning up fails", func() {
			innerBlobstore.DeleteErr = errors.New("fake-clean-up-error")

			err := retryableBlobstore.Delete("/some/file")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-clean-up-error"))
		})
	})

	Describe("Create", func() {
		Context("when inner blobstore succeeds before maximum number of create tries (first time)", func() {
			It("returns blobID and fingerprint without an error", func() {
				innerBlobstore.CreateBlobID = "fake-blob-id"
				innerBlobstore.CreateFingerprint = "fake-fingerprint"

				blobID, fingerprint, err := retryableBlobstore.Create("fake-file-name")
				Expect(err).ToNot(HaveOccurred())
				Expect(blobID).To(Equal("fake-blob-id"))
				Expect(fingerprint).To(Equal("fake-fingerprint"))

				Expect(innerBlobstore.CreateFileNames).To(Equal([]string{"fake-file-name"}))
			})
		})

		Context("when inner blobstore succeed exactly at maximum number of create tries", func() {
			It("returns blobID and fingerprint without an error", func() {
				innerBlobstore.CreateBlobIDs = []string{"", "", "fake-last-blob-id"}
				innerBlobstore.CreateFingerprints = []string{"", "", "fake-last-fingerprint"}
				innerBlobstore.CreateErrs = []error{
					errors.New("fake-create-err-1"),
					errors.New("fake-create-err-2"),
					nil,
				}

				blobID, fingerprint, err := retryableBlobstore.Create("fake-file-name")
				Expect(err).ToNot(HaveOccurred())
				Expect(blobID).To(Equal("fake-last-blob-id"))
				Expect(fingerprint).To(Equal("fake-last-fingerprint"))

				Expect(innerBlobstore.CreateFileNames).To(Equal(
					[]string{"fake-file-name", "fake-file-name", "fake-file-name"},
				))
			})
		})

		Context("when inner blobstore does not succeed before maximum number of create tries", func() {
			It("returns last try error from inner blobstore", func() {
				innerBlobstore.CreateErrs = []error{
					errors.New("fake-create-err-1"),
					errors.New("fake-create-err-2"),
					errors.New("fake-last-create-err"),
				}

				_, _, err := retryableBlobstore.Create("fake-blob-id")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-last-create-err"))

				Expect(innerBlobstore.CreateFileNames).To(Equal(
					[]string{"fake-blob-id", "fake-blob-id", "fake-blob-id"},
				))
			})
		})
	})

	Describe("Validate", func() {
		It("returns error if max tries is < 1", func() {
			err := boshblob.NewRetryableBlobstore(innerBlobstore, -1, logger).Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Max tries must be > 0"))

			err = boshblob.NewRetryableBlobstore(innerBlobstore, 0, logger).Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Max tries must be > 0"))
		})

		It("delegates to inner blobstore to validate", func() {
			err := retryableBlobstore.Validate()
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if inner blobstore validation fails", func() {
			innerBlobstore.ValidateError = bosherr.Error("fake-validate-error")

			err := retryableBlobstore.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-validate-error"))
		})
	})
})
