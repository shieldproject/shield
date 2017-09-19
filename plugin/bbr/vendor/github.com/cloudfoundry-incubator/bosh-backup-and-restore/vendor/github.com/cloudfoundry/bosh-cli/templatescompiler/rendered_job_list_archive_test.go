package templatescompiler_test

import (
	. "github.com/cloudfoundry/bosh-cli/templatescompiler"

	"bytes"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	mock_template "github.com/cloudfoundry/bosh-cli/templatescompiler/mocks"
	"github.com/golang/mock/gomock"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakeboshsys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("RenderedJobListArchive", func() {
	var mockCtrl *gomock.Controller

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	var (
		outBuffer *bytes.Buffer
		errBuffer *bytes.Buffer
		logger    boshlog.Logger
		fs        *fakeboshsys.FakeFileSystem

		mockRenderedJobList *mock_template.MockRenderedJobList

		renderedJobListArchivePath string
		renderedJobListArchiveSHA1 string

		renderedJobListArchive RenderedJobListArchive
	)

	BeforeEach(func() {
		outBuffer = bytes.NewBufferString("")
		errBuffer = bytes.NewBufferString("")
		logger = boshlog.NewWriterLogger(boshlog.LevelDebug, outBuffer, errBuffer)

		fs = fakeboshsys.NewFakeFileSystem()

		mockRenderedJobList = mock_template.NewMockRenderedJobList(mockCtrl)

		renderedJobListArchivePath = "fake-archive-path"
		renderedJobListArchiveSHA1 = "fake-sha1"

		renderedJobListArchive = NewRenderedJobListArchive(
			mockRenderedJobList,
			renderedJobListArchivePath,
			renderedJobListArchiveSHA1,
			fs, logger)
	})

	Describe("List", func() {
		It("returns the rendered job list", func() {
			Expect(renderedJobListArchive.List()).To(Equal(mockRenderedJobList))
		})
	})

	Describe("Path", func() {
		It("returns the rendered job list archive path", func() {
			Expect(renderedJobListArchive.Path()).To(Equal(renderedJobListArchivePath))
		})
	})

	Describe("SHA1", func() {
		It("returns the rendered job list archive sha1", func() {
			Expect(renderedJobListArchive.SHA1()).To(Equal(renderedJobListArchiveSHA1))
		})
	})

	Describe("Delete", func() {
		It("deletes the rendered job list archive from the file system", func() {
			err := fs.MkdirAll(renderedJobListArchivePath, os.ModePerm)
			Expect(err).ToNot(HaveOccurred())

			err = renderedJobListArchive.Delete()
			Expect(err).ToNot(HaveOccurred())
			Expect(fs.FileExists(renderedJobListArchivePath)).To(BeFalse())
		})

		Context("when deleting from the file system fails", func() {
			JustBeforeEach(func() {
				fs.RemoveAllStub = func(_ string) error {
					return bosherr.Error("fake-delete-error")
				}
			})

			It("returns an error", func() {
				err := renderedJobListArchive.Delete()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-delete-error"))
			})
		})
	})

	Describe("DeleteSilently", func() {
		It("deletes the rendered job path from the file system", func() {
			err := fs.MkdirAll(renderedJobListArchivePath, os.ModePerm)
			Expect(err).ToNot(HaveOccurred())

			renderedJobListArchive.DeleteSilently()
			Expect(fs.FileExists(renderedJobListArchivePath)).To(BeFalse())
		})

		Context("when deleting from the file system fails", func() {
			JustBeforeEach(func() {
				fs.RemoveAllStub = func(_ string) error {
					return bosherr.Error("fake-delete-error")
				}
			})

			It("logs the error", func() {
				renderedJobListArchive.DeleteSilently()

				errorLogString := errBuffer.String()
				Expect(errorLogString).To(ContainSubstring("Failed to delete rendered job list archive"))
				Expect(errorLogString).To(ContainSubstring("fake-delete-error"))
			})
		})
	})
})
