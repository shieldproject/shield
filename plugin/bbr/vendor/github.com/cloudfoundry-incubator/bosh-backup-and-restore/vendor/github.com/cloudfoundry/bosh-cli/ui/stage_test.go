package ui_test

import (
	"bytes"
	"strings"
	"time"

	. "github.com/cloudfoundry/bosh-cli/ui"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/pivotal-golang/clock/fakeclock"
)

var _ = Describe("Stage", func() {
	var (
		logOutBuffer, logErrBuffer *bytes.Buffer
		logger                     boshlog.Logger

		stage           Stage
		ui              UI
		fakeTimeService *fakeclock.FakeClock

		uiOut, uiErr *bytes.Buffer
	)

	BeforeEach(func() {
		uiOut = bytes.NewBufferString("")
		uiErr = bytes.NewBufferString("")

		logOutBuffer = bytes.NewBufferString("")
		logErrBuffer = bytes.NewBufferString("")
		logger = boshlog.NewWriterLogger(boshlog.LevelDebug, logOutBuffer, logErrBuffer)

		ui = NewWriterUI(uiOut, uiErr, logger)
		fakeTimeService = fakeclock.NewFakeClock(time.Now())

		stage = NewStage(ui, fakeTimeService, logger)
	})

	Describe("Perform", func() {
		It("prints a single-line stage", func() {
			actionsPerformed := []string{}

			err := stage.Perform("Simple stage 1", func() error {
				actionsPerformed = append(actionsPerformed, "1")
				fakeTimeService.Increment(time.Minute)
				return nil
			})

			Expect(err).To(BeNil())

			expectedOutput := "Simple stage 1... Finished (00:01:00)\n"
			Expect(uiOut.String()).To(Equal(expectedOutput))
			Expect(actionsPerformed).To(Equal([]string{"1"}))
		})

		It("fails on error", func() {
			actionsPerformed := []string{}
			stageError := bosherr.Error("fake-stage-1-error")

			err := stage.Perform("Simple stage 1", func() error {
				actionsPerformed = append(actionsPerformed, "1")
				fakeTimeService.Increment(time.Minute)
				return stageError
			})

			Expect(err).To(Equal(stageError))

			expectedOutput := "Simple stage 1... Failed (00:01:00)\n"
			Expect(uiOut.String()).To(Equal(expectedOutput))
			Expect(actionsPerformed).To(Equal([]string{"1"}))
		})

		It("logs skip errors", func() {
			actionsPerformed := []string{}

			err := stage.Perform("Simple stage 1", func() error {
				actionsPerformed = append(actionsPerformed, "1")
				cause := bosherr.Error("fake-skip-error")
				fakeTimeService.Increment(time.Minute)
				return NewSkipStageError(cause, "fake-skip-message")
			})

			Expect(err).ToNot(HaveOccurred())

			expectedOutput := "Simple stage 1... Skipped [fake-skip-message] (00:01:00)\n"
			Expect(uiOut.String()).To(Equal(expectedOutput))
			Expect(logOutBuffer.String()).To(ContainSubstring("fake-skip-message: fake-skip-error"))
			Expect(actionsPerformed).To(Equal([]string{"1"}))
		})
	})

	Describe("PerformComplex", func() {
		It("prints a multi-line stage (depth: 1)", func() {
			actionsPerformed := []string{}

			err := stage.PerformComplex("Complex stage 1", func(stage Stage) error {
				err := stage.Perform("Simple stage A", func() error {
					actionsPerformed = append(actionsPerformed, "A")
					fakeTimeService.Increment(time.Minute)
					return nil
				})
				if err != nil {
					return err
				}

				err = stage.Perform("Simple stage B", func() error {
					actionsPerformed = append(actionsPerformed, "B")
					fakeTimeService.Increment(time.Minute)
					return nil
				})
				if err != nil {
					return err
				}

				return nil
			})
			Expect(err).ToNot(HaveOccurred())

			expectedOutput := `
Started Complex stage 1
  Simple stage A... Finished (00:01:00)
  Simple stage B... Finished (00:01:00)
Finished Complex stage 1 (00:02:00)
`
			Expect(uiOut.String()).To(Equal(expectedOutput))
			Expect(actionsPerformed).To(Equal([]string{"A", "B"}))
		})

		It("prints a multi-line stage (depth: >1)", func() {
			actionsPerformed := []string{}

			err := stage.PerformComplex("Complex stage 1", func(stage Stage) error {
				err := stage.Perform("Simple stage A", func() error {
					actionsPerformed = append(actionsPerformed, "A")
					fakeTimeService.Increment(time.Minute)
					return nil
				})
				if err != nil {
					return err
				}

				err = stage.PerformComplex("Complex stage B", func(stage Stage) error {
					err := stage.Perform("Simple stage X", func() error {
						actionsPerformed = append(actionsPerformed, "X")
						fakeTimeService.Increment(time.Minute)
						return nil
					})
					if err != nil {
						return err
					}

					err = stage.Perform("Simple stage Y", func() error {
						actionsPerformed = append(actionsPerformed, "Y")
						fakeTimeService.Increment(time.Minute)
						return nil
					})
					if err != nil {
						return err
					}

					return nil
				})
				if err != nil {
					return err
				}

				return nil
			})
			Expect(err).ToNot(HaveOccurred())

			expectedOutput := `
Started Complex stage 1
  Simple stage A... Finished (00:01:00)
  #
  Started Complex stage B
    Simple stage X... Finished (00:01:00)
    Simple stage Y... Finished (00:01:00)
  Finished Complex stage B (00:02:00)
Finished Complex stage 1 (00:03:00)
`
			Expect(uiOut.String()).To(Equal(strings.Replace(expectedOutput, "#", "", -1)))
			Expect(actionsPerformed).To(Equal([]string{"A", "X", "Y"}))
		})

		It("fails on error", func() {
			actionsPerformed := []string{}
			stageError := bosherr.Error("fake-stage-1-error")
			err := stage.PerformComplex("Complex stage 1", func(stage Stage) error {
				actionsPerformed = append(actionsPerformed, "1")
				fakeTimeService.Increment(time.Minute)
				return stageError
			})
			Expect(err).To(Equal(stageError))

			expectedOutput := `
Started Complex stage 1
Failed Complex stage 1 (00:01:00)
`
			Expect(uiOut.String()).To(Equal(expectedOutput))
			Expect(actionsPerformed).To(Equal([]string{"1"}))
		})
	})
})
