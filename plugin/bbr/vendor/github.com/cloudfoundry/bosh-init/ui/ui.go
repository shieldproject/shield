package ui

import (
	"fmt"
	"io"
	"os"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type UI interface {
	ErrorLinef(pattern string, args ...interface{})
	PrintLinef(pattern string, args ...interface{})
	BeginLinef(pattern string, args ...interface{})
	EndLinef(pattern string, args ...interface{})
}

type ui struct {
	outWriter io.Writer
	errWriter io.Writer
	logger    boshlog.Logger
	logTag    string
}

func NewConsoleUI(logger boshlog.Logger) UI {
	return NewWriterUI(os.Stdout, os.Stderr, logger)
}

func NewWriterUI(outWriter, errWriter io.Writer, logger boshlog.Logger) UI {
	return &ui{
		outWriter: outWriter,
		errWriter: errWriter,
		logger:    logger,
		logTag:    "ui",
	}
}

// ErrorLinef starts and ends a text error line
func (ui *ui) ErrorLinef(pattern string, args ...interface{}) {
	message := fmt.Sprintf(pattern, args...)
	_, err := fmt.Fprintln(ui.errWriter, message)
	if err != nil {
		ui.logger.Error(ui.logTag, "UI.ErrorLinef failed (message='%s'): %s", message, err)
	}
}

// Printlnf starts and ends a text line
func (ui *ui) PrintLinef(pattern string, args ...interface{}) {
	message := fmt.Sprintf(pattern, args...)
	_, err := fmt.Fprintln(ui.outWriter, message)
	if err != nil {
		ui.logger.Error(ui.logTag, "UI.PrintLinef failed (message='%s'): %s", message, err)
	}
}

// PrintBeginf starts a text line
func (ui *ui) BeginLinef(pattern string, args ...interface{}) {
	message := fmt.Sprintf(pattern, args...)
	_, err := fmt.Fprint(ui.outWriter, message)
	if err != nil {
		ui.logger.Error(ui.logTag, "UI.BeginLinef failed (message='%s'): %s", message, err)
	}
}

// PrintEndf ends a text line
func (ui *ui) EndLinef(pattern string, args ...interface{}) {
	message := fmt.Sprintf(pattern, args...)
	_, err := fmt.Fprintln(ui.outWriter, message)
	if err != nil {
		ui.logger.Error(ui.logTag, "UI.EndLinef failed (message='%s'): %s", message, err)
	}
}
