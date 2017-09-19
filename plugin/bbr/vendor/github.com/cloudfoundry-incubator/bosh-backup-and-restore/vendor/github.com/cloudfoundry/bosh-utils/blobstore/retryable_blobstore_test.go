package blobstore_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshblob "github.com/cloudfoundry/bosh-utils/blobstore"
	fakeblob "github.com/cloudfoundry/bosh-utils/blobstore/fakes"
	boshcrypto "github.com/cloudfoundry/bosh-utils/crypto"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("retryableBlobstore", func() {
	var (
		innerBlobstore     *fakeblob.FakeDigestBlobstore
		logger             boshlog.Logger
		retryableBlobstore boshblob.DigestBlobstore
	)

	BeforeEach(func() {
		innerBlobstore = &fakeblob.FakeDigestBlobstore{}
		logger = boshlog.NewLogger(boshlog.LevelNone)
		retryableBlobstore = boshblob.NewRetryableBlobstore(innerBlobstore, 3, logger)
	})

	Describe("Get", func() {
		Context("when inner blobstore succeeds before maximum number of get tries (first time)", func() {
			It("returns path without an error", func() {
				innerBlobstore.GetReturns("fake-path", nil)
				digest := boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, "fingerprint")
				path, err := retryableBlobstore.Get("fake-blob-id", digest)
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal("fake-path"))
				actualBlobID, actualDigest := innerBlobstore.GetArgsForCall(0)
				Expect(actualBlobID).To(Equal("fake-blob-id"))
				Expect(actualDigest).To(Equal(digest))
			})
		})

		Context("when inner blobstore succeed exactly at maximum number of get tries", func() {
			It("returns path without an error", func() {
				tries := 0
				getFileNames := []string{"", "", "fake-last-path"}
				getErrs := []error{
					errors.New("fake-get-err-1"),
					errors.New("fake-get-err-2"),
					nil,
				}
				innerBlobstore.GetStub = func(blobID string, digest boshcrypto.Digest) (fileName string, err error) {
					defer func() { tries += 1 }()
					return getFileNames[tries], getErrs[tries]
				}

				digest := boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, "fingerprint")
				path, err := retryableBlobstore.Get("fake-blob-id", digest)
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal("fake-last-path"))

				blobIDs := []string{"fake-blob-id", "fake-blob-id", "fake-blob-id"}
				for call, expectedBlobID := range blobIDs {
					blobID, actualDigest := innerBlobstore.GetArgsForCall(call)
					Expect(blobID).To(Equal(expectedBlobID))
					Expect(actualDigest).To(Equal(digest))
				}
			})
		})

		Context("when inner blobstore does not succeed before maximum number of get tries", func() {
			It("returns last try error from inner blobstore", func() {
				tries := 0
				getFileNames := []string{"", "", ""}
				getErrs := []error{
					errors.New("fake-get-err-1"),
					errors.New("fake-get-err-2"),
					errors.New("fake-last-get-err"),
				}
				innerBlobstore.GetStub = func(blobID string, digest boshcrypto.Digest) (fileName string, err error) {
					defer func() { tries += 1 }()
					return getFileNames[tries], getErrs[tries]
				}

				digest := boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, "fingerprint")
				_, err := retryableBlobstore.Get("fake-blob-id", digest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-last-get-err"))

				blobIDs := []string{"fake-blob-id", "fake-blob-id", "fake-blob-id"}
				for call, blobID := range blobIDs {
					actualBlobID, actualDigest := innerBlobstore.GetArgsForCall(call)
					Expect(actualBlobID).To(Equal(blobID))
					Expect(actualDigest).To(Equal(digest))
				}
			})
		})
	})

	Describe("CleanUp", func() {
		It("delegates to inner blobstore to clean up", func() {
			err := retryableBlobstore.CleanUp("/some/file")
			Expect(err).ToNot(HaveOccurred())
			Expect(innerBlobstore.CleanUpArgsForCall(0)).To(Equal("/some/file"))
		})

		It("returns error if inner blobstore cleaning up fails", func() {
			innerBlobstore.CleanUpReturns(errors.New("fake-clean-up-error"))

			err := retryableBlobstore.CleanUp("/some/file")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-clean-up-error"))
		})
	})

	Describe("Delete", func() {
		It("delegates to inner blobstore", func() {
			err := retryableBlobstore.Delete("some-blob")
			Expect(err).ToNot(HaveOccurred())

			Expect(innerBlobstore.DeleteArgsForCall(0)).To(Equal("some-blob"))
		})

		It("returns error if inner blobstore fails", func() {
			innerBlobstore.DeleteReturns(errors.New("fake-delete-error"))

			err := retryableBlobstore.Delete("/some/file")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-delete-error"))
		})
	})

	Describe("Create", func() {
		Context("when inner blobstore succeeds before maximum number of create tries (first time)", func() {
			It("returns blobID and fingerprint without an error", func() {
				expectedDigest := boshcrypto.MultipleDigest{}
				innerBlobstore.CreateReturns("fake-blob-id", expectedDigest, nil)

				blobID, actualDigest, err := retryableBlobstore.Create("fake-file-name")

				Expect(err).ToNot(HaveOccurred())
				Expect(blobID).To(Equal("fake-blob-id"))
				Expect(actualDigest).To(Equal(expectedDigest))
				Expect(innerBlobstore.CreateArgsForCall(0)).To(Equal("fake-file-name"))
			})
		})

		Context("when inner blobstore succeed exactly at maximum number of create tries", func() {
			It("returns blobID and fingerprint without an error", func() {

				tries := 0
				expectedDigest := boshcrypto.MustParseMultipleDigest("someshasum")
				createBlobIDs := []string{"", "", "fake-last-blob-id"}

				createDigests := []boshcrypto.MultipleDigest{
					boshcrypto.MultipleDigest{},
					boshcrypto.MultipleDigest{},
					expectedDigest,
				}
				createErrs := []error{
					errors.New("fake-create-err-1"),
					errors.New("fake-create-err-2"),
					nil,
				}

				innerBlobstore.CreateStub = func(blobID string) (fileName string, digests boshcrypto.MultipleDigest, err error) {
					defer func() { tries += 1 }()
					return createBlobIDs[tries], createDigests[tries], createErrs[tries]
				}

				blobID, digest, err := retryableBlobstore.Create("fake-file-name")
				Expect(err).ToNot(HaveOccurred())

				Expect(digest).To(Equal(expectedDigest))
				Expect(blobID).To(Equal("fake-last-blob-id"))

				createFileNames := []string{"fake-file-name", "fake-file-name", "fake-file-name"}
				for call, createFileName := range createFileNames {
					Expect(innerBlobstore.CreateArgsForCall(call)).To(Equal(createFileName))
				}
			})
		})

		Context("when inner blobstore does not succeed before maximum number of create tries", func() {
			It("returns last try error from inner blobstore", func() {
				createErrs := []error{
					errors.New("fake-create-err-1"),
					errors.New("fake-create-err-2"),
					errors.New("fake-last-create-err"),
				}

				tries := 0
				innerBlobstore.CreateStub = func(blobID string) (fileName string, digests boshcrypto.MultipleDigest, err error) {
					defer func() { tries += 1 }()
					return "", boshcrypto.MultipleDigest{}, createErrs[tries]
				}
				_, _, err := retryableBlobstore.Create("fake-file-name")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-last-create-err"))

				createFileNames := []string{"fake-file-name", "fake-file-name", "fake-file-name"}
				for call, createFileName := range createFileNames {
					Expect(innerBlobstore.CreateArgsForCall(call)).To(Equal(createFileName))
				}
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
			innerBlobstore.ValidateReturns(bosherr.Error("fake-validate-error"))

			err := retryableBlobstore.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-validate-error"))
		})
	})
})
