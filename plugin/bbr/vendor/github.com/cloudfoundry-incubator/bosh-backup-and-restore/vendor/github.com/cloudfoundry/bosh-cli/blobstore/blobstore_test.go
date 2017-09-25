package blobstore_test

import (
	"errors"
	"io/ioutil"
	"strings"

	. "github.com/cloudfoundry/bosh-cli/blobstore"
	fakeboshdavcli "github.com/cloudfoundry/bosh-davcli/client/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Blobstore", func() {
	var (
		fakeDavClient     *fakeboshdavcli.FakeClient
		fakeUUIDGenerator *fakeuuid.FakeGenerator
		fs                *fakesys.FakeFileSystem
		blobstore         Blobstore
	)

	BeforeEach(func() {
		fakeDavClient = fakeboshdavcli.NewFakeClient()
		fakeUUIDGenerator = fakeuuid.NewFakeGenerator()
		fs = fakesys.NewFakeFileSystem()
		logger := boshlog.NewLogger(boshlog.LevelNone)

		blobstore = NewBlobstore(fakeDavClient, fakeUUIDGenerator, fs, logger)
	})

	Describe("Get", func() {
		BeforeEach(func() {
			fakeFile := fakesys.NewFakeFile("fake-destination-path", fs)
			fs.ReturnTempFile = fakeFile
		})

		It("gets the blob from the blobstore", func() {
			fakeDavClient.GetContents = ioutil.NopCloser(strings.NewReader("fake-content"))

			localBlob, err := blobstore.Get("fake-blob-id")
			Expect(err).ToNot(HaveOccurred())
			defer localBlob.DeleteSilently()

			Expect(fakeDavClient.GetPath).To(Equal("fake-blob-id"))
		})

		It("saves the blob to the destination path", func() {
			fakeDavClient.GetContents = ioutil.NopCloser(strings.NewReader("fake-content"))

			localBlob, err := blobstore.Get("fake-blob-id")
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err := localBlob.Delete()
				Expect(err).ToNot(HaveOccurred())
			}()

			Expect(localBlob.Path()).To(Equal("fake-destination-path"))

			contents, err := fs.ReadFileString("fake-destination-path")
			Expect(err).ToNot(HaveOccurred())
			Expect(contents).To(Equal("fake-content"))
		})

		Context("when getting from blobstore fails", func() {
			It("returns an error", func() {
				fakeDavClient.GetErr = errors.New("fake-get-error")

				_, err := blobstore.Get("fake-blob-id")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-get-error"))
			})
		})
	})

	Describe("Add", func() {
		BeforeEach(func() {
			fs.RegisterOpenFile("fake-source-path", &fakesys.FakeFile{
				Contents: []byte("fake-contents"),
			})
		})

		It("adds file to blobstore and returns its blob ID", func() {
			fakeUUIDGenerator.GeneratedUUID = "fake-blob-id"

			blobID, err := blobstore.Add("fake-source-path")
			Expect(err).ToNot(HaveOccurred())
			Expect(blobID).To(Equal("fake-blob-id"))
			Expect(fakeDavClient.PutPath).To(Equal("fake-blob-id"))
			Expect(fakeDavClient.PutContents).To(Equal("fake-contents"))
		})
	})
})
