package blobstore_test

import (
	. "github.com/cloudfoundry/bosh-cli/blobstore"

	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshdavcli "github.com/cloudfoundry/bosh-davcli/client"
	boshdavcliconf "github.com/cloudfoundry/bosh-davcli/config"
	bihttpclient "github.com/cloudfoundry/bosh-utils/httpclient"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
)

var _ = Describe("BlobstoreFactory", func() {
	var (
		fakeUUIDGenerator *fakeuuid.FakeGenerator
		httpClient        *http.Client
		fs                *fakesys.FakeFileSystem
		logger            boshlog.Logger
		blobstoreFactory  Factory
	)

	BeforeEach(func() {
		fakeUUIDGenerator = fakeuuid.NewFakeGenerator()
		fs = fakesys.NewFakeFileSystem()
		logger = boshlog.NewLogger(boshlog.LevelNone)
		httpClient = bihttpclient.DefaultClient
		blobstoreFactory = NewBlobstoreFactory(fakeUUIDGenerator, fs, logger)
	})

	Describe("Create", func() {
		Context("when username and password are provided", func() {
			It("returns the blobstore", func() {
				blobstore, err := blobstoreFactory.Create("https://fake-user:fake-password@fake-host:1234", httpClient)
				Expect(err).ToNot(HaveOccurred())
				davClient := boshdavcli.NewClient(boshdavcliconf.Config{
					Endpoint: "https://fake-host:1234/blobs",
					User:     "fake-user",
					Password: "fake-password",
				}, httpClient, logger)
				expectedBlobstore := NewBlobstore(davClient, fakeUUIDGenerator, fs, logger)
				Expect(blobstore).To(Equal(expectedBlobstore))
			})
		})

		Context("when URL does not have username and password", func() {
			// This test was added because parsing password is failing when userInfo is missing in URL
			It("returns the blobstore", func() {
				davClient := boshdavcli.NewClient(boshdavcliconf.Config{
					Endpoint: "https://fake-host:1234/blobs",
					User:     "",
					Password: "",
				}, httpClient, logger)
				expectedBlobstore := NewBlobstore(davClient, fakeUUIDGenerator, fs, logger)

				blobstore, err := blobstoreFactory.Create("https://fake-host:1234", httpClient)
				Expect(err).ToNot(HaveOccurred())
				Expect(blobstore).To(Equal(expectedBlobstore))
			})
		})
	})
})
