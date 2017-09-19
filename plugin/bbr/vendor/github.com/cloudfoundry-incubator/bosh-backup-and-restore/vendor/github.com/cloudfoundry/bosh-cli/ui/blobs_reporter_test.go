package ui_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/ui"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("BlobsReporter", func() {
	var (
		ui       *fakeui.FakeUI
		reporter BlobsReporter
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		reporter = NewBlobsReporter(ui)
	})

	Describe("BlobDownloadStarted", func() {
		It("prints download msg", func() {
			reporter.BlobDownloadStarted("path", 100, "blob-id", "blob-sha1")
			Expect(ui.Said).To(Equal([]string{
				"Blob download 'path' (100 B) (id: blob-id sha1: blob-sha1) started\n"}))
		})
	})

	Describe("BlobDownloadFinished", func() {
		It("prints failed download msg", func() {
			reporter.BlobDownloadFinished("path", "blob-id", errors.New("err"))
			Expect(ui.Errors).To(Equal([]string{"Blob download 'path' (id: blob-id) failed"}))
		})

		It("prints finished download msg", func() {
			reporter.BlobDownloadFinished("path", "blob-id", nil)
			Expect(ui.Said).To(Equal([]string{"Blob download 'path' (id: blob-id) finished\n"}))
		})
	})

	Describe("BlobUploadStarted", func() {
		It("prints upload msg", func() {
			reporter.BlobUploadStarted("path", 100, "blob-sha1")
			Expect(ui.Said).To(Equal([]string{"Blob upload 'path' (100 B) (sha1: blob-sha1) started\n"}))
		})
	})

	Describe("BlobUploadFinished", func() {
		It("prints failed upload msg", func() {
			reporter.BlobUploadFinished("path", "", errors.New("err"))
			Expect(ui.Errors).To(Equal([]string{"Blob upload 'path' failed"}))
		})

		It("prints finished upload msg", func() {
			reporter.BlobUploadFinished("path", "blob-id", nil)
			Expect(ui.Said).To(Equal([]string{"Blob upload 'path' (id: blob-id) finished\n"}))
		})
	})
})
