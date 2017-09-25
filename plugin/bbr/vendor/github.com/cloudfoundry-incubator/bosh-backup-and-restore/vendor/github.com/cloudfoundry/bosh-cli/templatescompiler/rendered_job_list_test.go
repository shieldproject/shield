package templatescompiler_test

import (
	"bytes"
	"os"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakeboshsys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	bireljob "github.com/cloudfoundry/bosh-cli/release/job"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
	. "github.com/cloudfoundry/bosh-cli/templatescompiler"
)

var _ = Describe("RenderedJobList", func() {
	var (
		outBuffer *bytes.Buffer
		errBuffer *bytes.Buffer
		logger    boshlog.Logger
		fs        *fakeboshsys.FakeFileSystem

		renderedJobList RenderedJobList
	)

	BeforeEach(func() {
		outBuffer = bytes.NewBufferString("")
		errBuffer = bytes.NewBufferString("")
		logger = boshlog.NewWriterLogger(boshlog.LevelDebug, outBuffer, errBuffer)
		fs = fakeboshsys.NewFakeFileSystem()
		renderedJobList = NewRenderedJobList()
	})

	Describe("All", func() {
		It("returns the added rendered jobs", func() {
			job0 := bireljob.NewJob(NewResource("fake-job-0", "", nil))
			renderedJob0 := NewRenderedJob(*job0, "fake-path-0", fs, logger)
			job1 := bireljob.NewJob(NewResource("fake-job-1", "", nil))
			renderedJob1 := NewRenderedJob(*job1, "fake-path-1", fs, logger)
			renderedJobList.Add(renderedJob0)
			renderedJobList.Add(renderedJob1)

			Expect(renderedJobList.All()).To(Equal([]RenderedJob{
				renderedJob0,
				renderedJob1,
			}))
		})
	})

	Describe("Delete", func() {
		It("deletes the rendered jobs", func() {
			err := fs.MkdirAll("fake-path-0", os.ModePerm)
			Expect(err).ToNot(HaveOccurred())

			err = fs.MkdirAll("fake-path-1", os.ModePerm)
			Expect(err).ToNot(HaveOccurred())

			job0 := bireljob.NewJob(NewResource("fake-job-0", "", nil))
			renderedJob0 := NewRenderedJob(*job0, "fake-path-0", fs, logger)
			job1 := bireljob.NewJob(NewResource("fake-job-0", "", nil))
			renderedJob1 := NewRenderedJob(*job1, "fake-path-1", fs, logger)
			renderedJobList.Add(renderedJob0)
			renderedJobList.Add(renderedJob1)

			err = renderedJobList.Delete()
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.FileExists("fake-path-0")).To(BeFalse())
			Expect(fs.FileExists("fake-path-1")).To(BeFalse())
		})

		Context("when deleting from the file system fails", func() {
			JustBeforeEach(func() {
				fs.RemoveAllStub = func(_ string) error {
					return bosherr.Error("fake-delete-error")
				}
			})

			It("returns an error", func() {
				job0 := bireljob.NewJob(NewResource("fake-job-0", "", nil))
				renderedJob0 := NewRenderedJob(*job0, "fake-path-0", fs, logger)
				renderedJobList.Add(renderedJob0)

				err := renderedJobList.Delete()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-delete-error"))
			})
		})
	})

	Describe("DeleteSilently", func() {
		It("deletes the rendered jobs", func() {
			err := fs.MkdirAll("fake-path-0", os.ModePerm)
			Expect(err).ToNot(HaveOccurred())

			err = fs.MkdirAll("fake-path-1", os.ModePerm)
			Expect(err).ToNot(HaveOccurred())

			job0 := bireljob.NewJob(NewResource("fake-job-0", "", nil))
			renderedJob0 := NewRenderedJob(*job0, "fake-path-0", fs, logger)
			job1 := bireljob.NewJob(NewResource("fake-job-1", "", nil))
			renderedJob1 := NewRenderedJob(*job1, "fake-path-1", fs, logger)
			renderedJobList.Add(renderedJob0)
			renderedJobList.Add(renderedJob1)

			renderedJobList.DeleteSilently()

			Expect(fs.FileExists("fake-path-0")).To(BeFalse())
			Expect(fs.FileExists("fake-path-1")).To(BeFalse())
		})

		Context("when deleting from the file system fails", func() {
			JustBeforeEach(func() {
				fs.RemoveAllStub = func(_ string) error {
					return bosherr.Error("fake-delete-error")
				}
			})

			It("logs all the errors", func() {
				job0 := bireljob.NewJob(NewResource("fake-job-0", "", nil))
				renderedJob0 := NewRenderedJob(*job0, "fake-path-0", fs, logger)
				job1 := bireljob.NewJob(NewResource("fake-job-1", "", nil))
				renderedJob1 := NewRenderedJob(*job1, "fake-path-1", fs, logger)
				renderedJobList.Add(renderedJob0)
				renderedJobList.Add(renderedJob1)

				renderedJobList.DeleteSilently()

				errorLogString := errBuffer.String()
				Expect(errorLogString).To(MatchRegexp("Failed to delete rendered job: .*fake-path-0.*fake-delete-error"))
				Expect(errorLogString).To(MatchRegexp("Failed to delete rendered job: .*fake-path-1.*fake-delete-error"))
			})
		})
	})
})
