package ui_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/ui"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("IndexReporter", func() {
	var (
		ui       *fakeui.FakeUI
		reporter IndexReporter
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		reporter = NewIndexReporter(ui)
	})

	Describe("IndexEntryStartedAdding", func() {
		It("prints download msg", func() {
			reporter.IndexEntryStartedAdding("type", "desc")
			Expect(ui.Said).To(Equal([]string{"Adding type 'desc'...\n"}))
		})
	})

	Describe("IndexEntryFinishedAdding", func() {
		It("prints failed download msg", func() {
			reporter.IndexEntryFinishedAdding("type", "desc", errors.New("err"))
			Expect(ui.Errors).To(Equal([]string{"Failed adding type 'desc'\n"}))
		})

		It("prints finished download msg", func() {
			reporter.IndexEntryFinishedAdding("type", "desc", nil)
			Expect(ui.Said).To(Equal([]string{"Added type 'desc'\n"}))
		})
	})

	Describe("IndexEntryDownloadStarted", func() {
		It("prints download msg", func() {
			reporter.IndexEntryDownloadStarted("type", "desc")
			Expect(ui.Said).To(Equal([]string{"-- Started downloading 'type' (desc)\n"}))
		})
	})

	Describe("IndexEntryDownloadFinished", func() {
		It("prints failed download msg", func() {
			reporter.IndexEntryDownloadFinished("type", "desc", errors.New("err"))
			Expect(ui.Errors).To(Equal([]string{"-- Failed downloading 'type' (desc)\n"}))
		})

		It("prints finished download msg", func() {
			reporter.IndexEntryDownloadFinished("type", "desc", nil)
			Expect(ui.Said).To(Equal([]string{"-- Finished downloading 'type' (desc)\n"}))
		})
	})

	Describe("IndexEntryUploadStarted", func() {
		It("prints upload msg", func() {
			reporter.IndexEntryUploadStarted("type", "desc")
			Expect(ui.Said).To(Equal([]string{"-- Started uploading 'type' (desc)\n"}))
		})
	})

	Describe("IndexEntryUploadFinished", func() {
		It("prints failed upload msg", func() {
			reporter.IndexEntryUploadFinished("type", "desc", errors.New("err"))
			Expect(ui.Errors).To(Equal([]string{"-- Failed uploading 'type' (desc)\n"}))
		})

		It("prints finished upload msg", func() {
			reporter.IndexEntryUploadFinished("type", "desc", nil)
			Expect(ui.Said).To(Equal([]string{"-- Finished uploading 'type' (desc)\n"}))
		})
	})
})
