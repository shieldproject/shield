package blobstore_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshblob "github.com/cloudfoundry/bosh-utils/blobstore"
	fakeblob "github.com/cloudfoundry/bosh-utils/blobstore/fakes"
	boshcrypto "github.com/cloudfoundry/bosh-utils/crypto"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"

	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("checksumVerifiableBlobstore", func() {
	const (
		fixturePath = "test_assets/some.config"
		fixtureSHA1 = "da39a3ee5e6b4b0d3255bfef95601890afd80709"
	)

	var (
		innerBlobstore              *fakeblob.FakeBlobstore
		checksumVerifiableBlobstore boshblob.DigestBlobstore
		correctDigest               boshcrypto.Digest
		fs                          *fakesys.FakeFileSystem
	)

	BeforeEach(func() {
		correctDigest = boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, fixtureSHA1)
		innerBlobstore = &fakeblob.FakeBlobstore{}
		fs = fakesys.NewFakeFileSystem()
		createAlgorithms := []boshcrypto.Algorithm{
			boshcrypto.DigestAlgorithmSHA1,
			boshcrypto.DigestAlgorithmSHA256,
		}
		checksumVerifiableBlobstore = boshblob.NewDigestVerifiableBlobstore(innerBlobstore, fs, createAlgorithms)
	})

	Describe("Get", func() {
		It("returns without an error if digest matches", func() {
			innerBlobstore.GetReturns(fixturePath, nil)

			fileName, err := checksumVerifiableBlobstore.Get("fake-blob-id", correctDigest)
			Expect(err).ToNot(HaveOccurred())

			Expect(innerBlobstore.GetArgsForCall(0)).To(Equal("fake-blob-id"))
			Expect(fileName).To(Equal(fixturePath))
		})

		It("returns error if digest does not match", func() {
			innerBlobstore.GetReturns(fixturePath, nil)

			incorrectDigest := boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, "some-incorrect-sha1")

			_, err := checksumVerifiableBlobstore.Get("fake-blob-id", incorrectDigest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Checking downloaded blob 'fake-blob-id'"))
		})

		It("returns error if inner blobstore getting fails", func() {
			innerBlobstore.GetReturns("", errors.New("fake-get-error"))

			_, err := checksumVerifiableBlobstore.Get("fake-blob-id", correctDigest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-get-error"))
		})
	})

	Describe("CleanUp", func() {
		It("delegates to inner blobstore to clean up", func() {
			err := checksumVerifiableBlobstore.CleanUp("/some/file")
			Expect(err).ToNot(HaveOccurred())

			Expect(innerBlobstore.CleanUpArgsForCall(0)).To(Equal("/some/file"))
		})

		It("returns error if inner blobstore cleaning up fails", func() {
			innerBlobstore.CleanUpReturns(errors.New("fake-clean-up-error"))

			err := checksumVerifiableBlobstore.CleanUp("/some/file")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-clean-up-error"))
		})
	})

	Describe("Delete", func() {
		It("delegates to inner blobstore", func() {
			err := checksumVerifiableBlobstore.Delete("some-blob")
			Expect(err).ToNot(HaveOccurred())

			Expect(innerBlobstore.DeleteArgsForCall(0)).To(Equal("some-blob"))
		})

		It("returns error if inner blobstore fails", func() {
			innerBlobstore.DeleteReturns(errors.New("fake-clean-up-error"))

			err := checksumVerifiableBlobstore.Delete("/some/file")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-clean-up-error"))
		})
	})

	Describe("Create", func() {
		BeforeEach(func() {
			fakeFile := fakesys.NewFakeFile(fixturePath, fs)
			fakeFile.Write([]byte("blargityblargblarg"))
			fs.RegisterOpenFile(fixturePath, fakeFile)
		})

		It("delegates to inner blobstore to create blob and returns a multiple digest of returned blob", func() {
			innerBlobstore.CreateReturns("fake-blob-id", nil)

			blobID, multipleDigest, err := checksumVerifiableBlobstore.Create(fixturePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(blobID).To(Equal("fake-blob-id"))

			Expect(innerBlobstore.CreateArgsForCall(0)).To(Equal(fixturePath))
			Expect(multipleDigest.String()).To(Equal("b153af8b5f71cf357896988886a76e9fe59b1e2e;sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"))
		})

		It("returns error if blob cannot be opened by filesystem", func() {
			fs.OpenFileErr = errors.New("no-way")

			_, _, err := checksumVerifiableBlobstore.Create(fixturePath)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("no-way"))
		})

		It("returns an error if creating the digest fails", func() {
			createAlgorithms := []boshcrypto.Algorithm{boshcrypto.NewUnknownAlgorithm("who")}
			checksumVerifiableBlobstore = boshblob.NewDigestVerifiableBlobstore(innerBlobstore, fs, createAlgorithms)

			innerBlobstore.CreateReturns("fake-blob-id", nil)
			_, _, err := checksumVerifiableBlobstore.Create(fixturePath)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Unable to create digest of unknown algorithm 'who'"))
		})

		It("returns error if inner blobstore blob creation fails", func() {
			innerBlobstore.CreateReturns("", errors.New("fake-create-error"))

			_, _, err := checksumVerifiableBlobstore.Create(fixturePath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-create-error"))
		})
	})

	Describe("Validate", func() {
		It("delegates to inner blobstore to validate", func() {
			err := checksumVerifiableBlobstore.Validate()
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if inner blobstore validation fails", func() {
			innerBlobstore.ValidateReturns(bosherr.Error("fake-validate-error"))

			err := checksumVerifiableBlobstore.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-validate-error"))
		})
	})
})
