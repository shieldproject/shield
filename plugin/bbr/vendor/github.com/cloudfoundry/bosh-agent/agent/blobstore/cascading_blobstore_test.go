package blobstore_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"errors"
	"github.com/cloudfoundry/bosh-agent/agent/blobstore"
	boshblob "github.com/cloudfoundry/bosh-utils/blobstore"
	fakeblob "github.com/cloudfoundry/bosh-utils/blobstore/fakes"
	boshcrypto "github.com/cloudfoundry/bosh-utils/crypto"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("cascadingBlobstore", func() {
	var (
		innerBlobstore     *fakeblob.FakeDigestBlobstore
		blobManager        *fakeblob.FakeBlobManagerInterface
		cascadingBlobstore boshblob.DigestBlobstore
	)

	BeforeEach(func() {
		innerBlobstore = &fakeblob.FakeDigestBlobstore{}
		blobManager = &fakeblob.FakeBlobManagerInterface{}
		logger := boshlog.NewLogger(boshlog.LevelNone)

		cascadingBlobstore = blobstore.NewCascadingBlobstore(innerBlobstore, blobManager, logger)
	})

	Describe("Get", func() {

		Context("when blobManager does contain the blob", func() {

			BeforeEach(func() {
				blobManager.BlobExistsReturns(true)
			})

			It("returns the path provided by the blobManager", func() {
				blobManager.GetPathReturns("/path/to-copy/of-blob", nil)
				digest := boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, "fake-checksum")
				filename, err := cascadingBlobstore.Get("blobID", digest)

				Expect(err).To(BeNil())
				Expect(filename).To(Equal("/path/to-copy/of-blob"))

				Expect(blobManager.GetPathCallCount()).To(Equal(1))

				receivedBlobID, receivedDigest := blobManager.GetPathArgsForCall(0)
				Expect(receivedBlobID).To(Equal("blobID"))
				Expect(receivedDigest).To(Equal(digest))

				Expect(innerBlobstore.GetCallCount()).To(Equal(0))
			})

			Context("when blobManager returns an error", func() {
				It("returns that error to the caller", func() {
					blobManager.GetPathReturns("", errors.New("some-error"))
					digest := boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, "fake-checksum")

					filename, err := cascadingBlobstore.Get("blobID", digest)

					Expect(filename).To(BeEmpty())
					Expect(err.Error()).To(Equal("some-error"))

					Expect(blobManager.GetPathCallCount()).To(Equal(1))

					receivedBlobID, receivedDigest := blobManager.GetPathArgsForCall(0)
					Expect(receivedBlobID).To(Equal("blobID"))
					Expect(receivedDigest).To(Equal(digest))

					Expect(innerBlobstore.GetCallCount()).To(Equal(0))
				})
			})
		})

		Context("when blobManager does NOT contain the blob", func() {

			BeforeEach(func() {
				blobManager.BlobExistsReturns(false)
			})

			It("delegates the action of getting the blob to inner blobstore", func() {
				blobID := "smurf-4"
				digest := boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, "smurf-4-sha")

				blobManager.GetPathReturns("", errors.New("broken"))

				innerBlobstore.GetReturns("/smurf-file/path", nil)
				innerBlobstore.CreateReturns("createdBlobID", boshcrypto.MultipleDigest{}, nil)

				filename, err := cascadingBlobstore.Get(blobID, digest)

				Expect(filename).To(Equal("/smurf-file/path"))
				Expect(err).To(BeNil())

				Expect(blobManager.GetPathCallCount()).To(Equal(0))

				Expect(innerBlobstore.GetCallCount()).To(Equal(1))
				receivedBlobID, receivedDigest := innerBlobstore.GetArgsForCall(0)
				Expect(receivedBlobID).To(Equal(blobID))
				Expect(receivedDigest).To(Equal(digest))
			})

			Context("when inner blobstore returns an error", func() {

				It("returns that error to the caller", func() {
					blobID := "smurf-5"
					sha1 := boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, "smurf-5-sha")

					blobManager.GetPathReturns("", errors.New("broken"))

					innerBlobstore.GetReturns("/smurf-file/path", errors.New("inner blobstore GET is broken"))

					_, err := cascadingBlobstore.Get(blobID, boshcrypto.MustNewMultipleDigest(sha1))

					Expect(err.Error()).To(Equal("inner blobstore GET is broken"))
					Expect(blobManager.GetPathCallCount()).To(Equal(0))
				})
			})
		})
	})

	Describe("CleanUp", func() {
		It("delegates the action to the inner blobstore", func() {
			err := cascadingBlobstore.CleanUp("fileToDelete")
			Expect(err).To(BeNil())
			Expect(innerBlobstore.CleanUpArgsForCall(0)).To(Equal("fileToDelete"))
		})

		It("returns an error if the inner blobstore fails to clean up", func() {
			innerBlobstore.CleanUpReturns(errors.New("error cleaning up"))

			err := cascadingBlobstore.CleanUp("randomFile")
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("error cleaning up"))
		})
	})

	Describe("Create", func() {
		It("delegates the action to the inner blobstore", func() {
			innerBlobstore.CreateReturns("createBlobId", boshcrypto.MultipleDigest{}, nil)

			createdBlobID, _, err := cascadingBlobstore.Create("createdFile")

			Expect(err).To(BeNil())

			Expect(createdBlobID).To(Equal("createBlobId"))

			Expect(innerBlobstore.CreateCallCount()).To(Equal(1))
			Expect(innerBlobstore.CreateArgsForCall(0)).To(Equal("createdFile"))
		})

		It("returns an error if the inner blobstore fails to create", func() {
			innerBlobstore.CreateReturns("", boshcrypto.MultipleDigest{}, errors.New("error creating"))

			_, _, err := cascadingBlobstore.Create("createdFile")

			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("error creating"))
		})
	})

	Describe("Validate", func() {
		It("delegates the action to the inner blobstore", func() {
			err := cascadingBlobstore.Validate()

			Expect(err).To(BeNil())
		})

		It("returns an error if the inner blobstore fails to validate", func() {
			innerBlobstore.ValidateReturns(errors.New("error validating"))

			err := cascadingBlobstore.Validate()

			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("error validating"))
		})
	})

	Describe("Delete", func() {
		It("deletes the blob from the blobManager, and calls Delete on inner blobstore", func() {
			blobID := "smurf-25"

			blobManager.DeleteReturns(nil)

			err := cascadingBlobstore.Delete(blobID)

			Expect(err).To(BeNil())

			Expect(blobManager.DeleteCallCount()).To(Equal(1))
			Expect(blobManager.DeleteArgsForCall(0)).To(Equal(blobID))

			Expect(innerBlobstore.DeleteArgsForCall(0)).To(Equal(blobID))
		})

		It("returns an error if blobManager returns an error when deleting", func() {
			blobID := "smurf-28"

			blobManager.DeleteReturns(errors.New("error deleting in blobManager"))

			err := cascadingBlobstore.Delete(blobID)

			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("error deleting in blobManager"))

			Expect(blobManager.DeleteCallCount()).To(Equal(1))
			Expect(blobManager.DeleteArgsForCall(0)).To(Equal(blobID))

			Expect(innerBlobstore.DeleteCallCount()).To(Equal(0))
		})

		It("returns an error if inner blobStore returns an error when deleting", func() {
			blobID := "smurf-29"

			blobManager.DeleteReturns(nil)
			innerBlobstore.DeleteReturns(errors.New("error deleting in innerBlobStore"))

			err := cascadingBlobstore.Delete(blobID)

			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("error deleting in innerBlobStore"))

			Expect(blobManager.DeleteCallCount()).To(Equal(1))
			Expect(blobManager.DeleteArgsForCall(0)).To(Equal(blobID))

			Expect(innerBlobstore.DeleteArgsForCall(0)).To(Equal(blobID))
		})
	})
})
