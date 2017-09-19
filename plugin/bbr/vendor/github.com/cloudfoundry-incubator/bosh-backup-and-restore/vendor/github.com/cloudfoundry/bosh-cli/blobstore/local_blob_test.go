package blobstore_test

import (
	. "github.com/cloudfoundry/bosh-cli/blobstore"

	"bytes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakeboshsys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("LocalBlobstore", func() {
	var (
		outBuffer *bytes.Buffer
		errBuffer *bytes.Buffer
		logger    boshlog.Logger
		fs        *fakeboshsys.FakeFileSystem

		localBlobPath string

		localBlob LocalBlob
	)

	BeforeEach(func() {
		outBuffer = bytes.NewBufferString("")
		errBuffer = bytes.NewBufferString("")
		logger = boshlog.NewWriterLogger(boshlog.LevelDebug, outBuffer, errBuffer)

		fs = fakeboshsys.NewFakeFileSystem()

		localBlobPath = "fake-local-blob-path"

		localBlob = NewLocalBlob(localBlobPath, fs, logger)
	})

	Describe("Path", func() {
		It("returns the local blob path", func() {
			Expect(localBlob.Path()).To(Equal(localBlobPath))
		})
	})

	Describe("Delete", func() {
		It("deletes the local blob from the file system", func() {
			err := fs.WriteFileString(localBlobPath, "fake-local-blob-content")
			Expect(err).ToNot(HaveOccurred())

			err = localBlob.Delete()
			Expect(err).ToNot(HaveOccurred())
			Expect(fs.FileExists(localBlobPath)).To(BeFalse())
		})

		Context("when deleting from the file system fails", func() {
			JustBeforeEach(func() {
				fs.RemoveAllStub = func(_ string) error {
					return bosherr.Error("fake-delete-error")
				}
			})

			It("returns an error", func() {
				err := localBlob.Delete()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-delete-error"))
			})
		})
	})

	Describe("DeleteSilently", func() {
		It("deletes the local blob from the file system", func() {
			err := fs.WriteFileString(localBlobPath, "fake-local-blob-content")
			Expect(err).ToNot(HaveOccurred())

			localBlob.DeleteSilently()
			Expect(fs.FileExists(localBlobPath)).To(BeFalse())
		})

		Context("when deleting from the file system fails", func() {
			JustBeforeEach(func() {
				fs.RemoveAllStub = func(_ string) error {
					return bosherr.Error("fake-delete-error")
				}
			})

			It("logs the error", func() {
				localBlob.DeleteSilently()

				errorLogString := errBuffer.String()
				Expect(errorLogString).To(ContainSubstring("Failed to delete local blob"))
				Expect(errorLogString).To(ContainSubstring("fake-delete-error"))
			})
		})
	})
})
