package ui_test

import (
	"bytes"
	"io"
	"os"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/ui"
	. "github.com/cloudfoundry/bosh-cli/ui/table"
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

		Context("when writing fails", func() {
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

		Context("when writing fails", func() {
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

		Context("when writing fails", func() {
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

		Context("when writing fails", func() {
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

	Describe("PrintBlock", func() {
		It("prints to outWriter as is", func() {
			ui.PrintBlock("block")
			Expect(uiOutBuffer.String()).To(Equal("block"))
			Expect(uiErrBuffer.String()).To(Equal(""))
		})

		Context("when writing fails", func() {
			BeforeEach(func() {
				reader, writer := io.Pipe()
				uiOut = writer
				reader.Close()
			})

			It("logs an error", func() {
				ui.PrintBlock("block")
				Expect(uiOutBuffer.String()).To(Equal(""))
				Expect(uiErrBuffer.String()).To(Equal(""))
				Expect(logOutBuffer.String()).To(Equal(""))
				Expect(logErrBuffer.String()).To(ContainSubstring("UI.PrintBlock failed (message='block')"))
			})
		})
	})

	Describe("PrintErrorBlock", func() {
		It("prints to outWriter as is", func() {
			ui.PrintErrorBlock("block")
			Expect(uiOutBuffer.String()).To(Equal("block"))
			Expect(uiErrBuffer.String()).To(Equal(""))
		})

		Context("when writing fails", func() {
			BeforeEach(func() {
				reader, writer := io.Pipe()
				uiOut = writer
				reader.Close()
			})

			It("logs an error", func() {
				ui.PrintErrorBlock("block")
				Expect(uiOutBuffer.String()).To(Equal(""))
				Expect(uiErrBuffer.String()).To(Equal(""))
				Expect(logOutBuffer.String()).To(Equal(""))
				Expect(logErrBuffer.String()).To(ContainSubstring("UI.PrintErrorBlock failed (message='block')"))
			})
		})
	})

	Describe("PrintTable", func() {
		It("prints table", func() {
			table := Table{
				Title:   "Title",
				Content: "things",
				Header:  []Header{NewHeader("Header1"), NewHeader("Header2")},

				Rows: [][]Value{
					{ValueString{"r1c1"}, ValueString{"r1c2"}},
					{ValueString{"r2c1"}, ValueString{"r2c2"}},
				},

				Notes:         []string{"note1", "note2"},
				BackgroundStr: ".",
				BorderStr:     "|",
			}
			ui.PrintTable(table)
			Expect("\n" + uiOutBuffer.String()).To(Equal(`
Title

Header1|Header2|
r1c1...|r1c2...|
r2c1...|r2c2...|

note1
note2

2 things
`))
		})
	})

	Describe("IsInteractive", func() {
		It("returns true", func() {
			Expect(ui.IsInteractive()).To(BeTrue())
		})
	})

	Describe("Flush", func() {
		It("does nothing", func() {
			Expect(func() { ui.Flush() }).ToNot(Panic())
		})
	})

	Describe("AskForText", func() {
		It("allows empty and non-empty text input", func() {
			r, w, err := os.Pipe()
			Expect(err).ToNot(HaveOccurred())

			os.Stdin = r

			_, err = w.Write([]byte("\ntest\n"))
			Expect(err).ToNot(HaveOccurred())

			err = w.Close()
			Expect(err).ToNot(HaveOccurred())

			text, err := ui.AskForText("ask-test")
			Expect(err).ToNot(HaveOccurred())
			Expect(text).To(Equal(""))

			text, err = ui.AskForText("ask-test2")
			Expect(err).ToNot(HaveOccurred())
			Expect(text).To(Equal("test"))
		})
	})

	Describe("AskForPassword", func() {
		It("allows empty and non-empty password input", func() {
			r, w, err := os.Pipe()
			Expect(err).ToNot(HaveOccurred())

			os.Stdin = r

			_, err = w.Write([]byte("\npassword\n"))
			Expect(err).ToNot(HaveOccurred())

			err = w.Close()
			Expect(err).ToNot(HaveOccurred())

			text, err := ui.AskForPassword("ask-test")
			Expect(err).ToNot(HaveOccurred())
			Expect(text).To(Equal(""))

			text, err = ui.AskForPassword("ask-test2")
			Expect(err).ToNot(HaveOccurred())
			Expect(text).To(Equal("password"))
		})
	})
})
