package ui_test

import (
	. "github.com/cloudfoundry/bosh-init/ui"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"bytes"
	"io"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("UI", func() {
	var (
		logOutBuffer, logErrBuffer *bytes.Buffer
		uiOutBuffer, uiErrBuffer   *bytes.Buffer
		uiOut, uiErr               io.Writer
		logger                     boshlog.Logger
		ui                         UI
	)

	BeforeEach(func() {
		uiOutBuffer = bytes.NewBufferString("")
		uiOut = uiOutBuffer
		uiErrBuffer = bytes.NewBufferString("")
		uiErr = uiErrBuffer

		logOutBuffer = bytes.NewBufferString("")
		logErrBuffer = bytes.NewBufferString("")
		logger = boshlog.NewWriterLogger(boshlog.LevelDebug, logOutBuffer, logErrBuffer)
	})

	JustBeforeEach(func() {
		ui = NewWriterUI(uiOut, uiErr, logger)
	})

	Describe("ErrorLinef", func() {
		It("prints to errWriter with a trailing newline", func() {
			ui.ErrorLinef("fake-error-line")
			Expect(uiOutBuffer.String()).To(Equal(""))
			Expect(uiErrBuffer.String()).To(ContainSubstring("fake-error-line\n"))
		})

		Context("when writing errors", func() {
			BeforeEach(func() {
				reader, writer := io.Pipe()
				uiErr = writer
				reader.Close()
			})

			It("logs an error", func() {
				ui.ErrorLinef("fake-error-line")

				Expect(uiOutBuffer.String()).To(Equal(""))
				Expect(uiErrBuffer.String()).To(Equal(""))
				Expect(logOutBuffer.String()).To(Equal(""))
				Expect(logErrBuffer.String()).To(ContainSubstring("UI.ErrorLinef failed (message='fake-error-line')"))
			})
		})
	})

	Describe("PrintLinef", func() {
		It("prints to outWriter with a trailing newline", func() {
			ui.PrintLinef("fake-line")
			Expect(uiOutBuffer.String()).To(ContainSubstring("fake-line\n"))
		})

		Context("when writing errors", func() {
			BeforeEach(func() {
				reader, writer := io.Pipe()
				uiOut = writer
				reader.Close()
			})

			It("logs an error", func() {
				ui.PrintLinef("fake-start")

				Expect(uiOutBuffer.String()).To(Equal(""))
				Expect(uiErrBuffer.String()).To(Equal(""))
				Expect(logOutBuffer.String()).To(Equal(""))
				Expect(logErrBuffer.String()).To(ContainSubstring("UI.PrintLinef failed (message='fake-start')"))
			})
		})
	})

	Describe("BeginLinef", func() {
		It("prints to outWriter", func() {
			ui.BeginLinef("fake-start")
			Expect(uiOutBuffer.String()).To(ContainSubstring("fake-start"))
		})

		Context("when writing errors", func() {
			BeforeEach(func() {
				reader, writer := io.Pipe()
				uiOut = writer
				reader.Close()
			})

			It("logs an error", func() {
				ui.BeginLinef("fake-start")

				Expect(uiOutBuffer.String()).To(Equal(""))
				Expect(uiErrBuffer.String()).To(Equal(""))
				Expect(logOutBuffer.String()).To(Equal(""))
				Expect(logErrBuffer.String()).To(ContainSubstring("UI.BeginLinef failed (message='fake-start')"))
			})
		})
	})

	Describe("EndLinef", func() {
		It("prints to outWriter with a trailing newline", func() {
			ui.EndLinef("fake-end")
			Expect(uiOutBuffer.String()).To(ContainSubstring("fake-end\n"))
		})

		Context("when writing errors", func() {
			BeforeEach(func() {
				reader, writer := io.Pipe()
				uiOut = writer
				reader.Close()
			})

			It("logs an error", func() {
				ui.EndLinef("fake-start")

				Expect(uiOutBuffer.String()).To(Equal(""))
				Expect(uiErrBuffer.String()).To(Equal(""))
				Expect(logOutBuffer.String()).To(Equal(""))
				Expect(logErrBuffer.String()).To(ContainSubstring("UI.EndLinef failed (message='fake-start')"))
			})
		})
	})
})
