package blobstore_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-utils/blobstore"
	boshcrypto "github.com/cloudfoundry/bosh-utils/crypto"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
)

var _ = Describe("Provider", func() {
	var (
		fs       *fakesys.FakeFileSystem
		runner   *fakesys.FakeCmdRunner
		logger   boshlog.Logger
		provider Provider
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		runner = fakesys.NewFakeCmdRunner()
		logger = boshlog.NewLogger(boshlog.LevelNone)
		provider = NewProvider(fs, runner, "/var/vcap/config", logger)
	})

	Describe("Get", func() {
		It("get dummy", func() {
			blobstore, err := provider.Get(BlobstoreTypeDummy, map[string]interface{}{})
			Expect(err).ToNot(HaveOccurred())
			Expect(blobstore).ToNot(BeNil())
		})

		It("get external when external command in path", func() {
			options := map[string]interface{}{"key": "value"}
			runner.CommandExistsValue = true

			externalBlobstore := NewExternalBlobstore(
				"fake-external-type",
				options,
				fs,
				runner,
				boshuuid.NewGenerator(),
				"/var/vcap/config/blobstore-fake-external-type.json",
			)

			expectedAlgos := []boshcrypto.Algorithm{
				boshcrypto.DigestAlgorithmSHA1,
			}

			expectedBlobstore := NewDigestVerifiableBlobstore(externalBlobstore, fs, expectedAlgos)
			expectedBlobstore = NewRetryableBlobstore(expectedBlobstore, 3, logger)

			blobstore, err := provider.Get("fake-external-type", options)
			Expect(err).ToNot(HaveOccurred())
			Expect(blobstore).To(Equal(expectedBlobstore))

			err = expectedBlobstore.Validate()
			Expect(err).ToNot(HaveOccurred())
		})

		It("get external errs when external command not in path", func() {
			options := map[string]interface{}{"key": "value"}
			runner.CommandExistsValue = false

			_, err := provider.Get("fake-external-type", options)
			Expect(err).To(HaveOccurred())
		})
	})
})
