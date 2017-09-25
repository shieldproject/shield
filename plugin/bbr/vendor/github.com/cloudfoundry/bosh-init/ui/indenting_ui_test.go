package ui_test

import (
	. "github.com/cloudfoundry/bosh-init/ui"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"bytes"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("IndentingUI", func() {
	var (
		uiOut, uiErr *bytes.Buffer
		ui           UI
	)

	BeforeEach(func() {
		uiOut = bytes.NewBufferString("")
		uiErr = bytes.NewBufferString("")

		logger := boshlog.NewLogger(boshlog.LevelNone)
		ui = NewIndentingUI(NewWriterUI(uiOut, uiErr, logger))
	})

	Describe("ErrorLinef", func() {
		It("delegates to the parent UI.ErrorLinef with an indent", func() {
			ui.ErrorLinef("fake-error-line")
			Expect(uiErr.String()).To(ContainSubstring("  fake-error-line\n"))
			Expect(uiOut.String()).To(BeEmpty())
		})
	})

	Describe("PrintLinef", func() {
		It("delegates to the parent UI.PrintLinef with an indent", func() {
			ui.PrintLinef("fake-line")
			Expect(uiOut.String()).To(ContainSubstring("  fake-line\n"))
			Expect(uiErr.String()).To(BeEmpty())
		})
	})

	Describe("BeginLinef", func() {
		It("delegates to the parent UI.BeginLinef with an indent", func() {
			ui.BeginLinef("fake-start")
			Expect(uiOut.String()).To(ContainSubstring("  fake-start"))
			Expect(uiErr.String()).To(BeEmpty())
		})
	})

	Describe("EndLinef", func() {
		It("delegates to the UI.EndLinef", func() {
			ui.EndLinef("fake-end")
			Expect(uiOut.String()).To(ContainSubstring("fake-end\n"))
			Expect(uiErr.String()).To(BeEmpty())
		})
	})
})
