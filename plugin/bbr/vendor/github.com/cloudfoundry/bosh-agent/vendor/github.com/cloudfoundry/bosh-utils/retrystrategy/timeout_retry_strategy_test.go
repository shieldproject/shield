package retrystrategy_test

import (
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/pivotal-golang/clock/fakeclock"

	. "github.com/cloudfoundry/bosh-utils/retrystrategy"
)

var _ = Describe("TimeoutRetryStrategy", func() {
	var (
		fakeTimeService *fakeclock.FakeClock
		logger          boshlog.Logger
	)

	BeforeEach(func() {
		fakeTimeService = fakeclock.NewFakeClock(time.Now())
		logger = boshlog.NewLogger(boshlog.LevelNone)
	})

	Describe("Try", func() {
		Context("when there are errors during a try", func() {
			It("retries until the timeout", func() {
				retryable := newSimpleRetryable([]attemptOutput{
					{
						IsRetryable: true,
						AttemptErr:  errors.New("first-error"),
					},
					{
						IsRetryable: true,
						AttemptErr:  errors.New("second-error"),
					},
					{
						IsRetryable: true,
						AttemptErr:  errors.New("third-error"),
					},
					{
						IsRetryable: true,
						AttemptErr:  errors.New("fourth-error"),
					},
				})
				// deadline between 2nd and 3rd attempts
				delay := 10 * time.Second
				timeoutRetryStrategy := NewTimeoutRetryStrategy(25*time.Second, delay, retryable, fakeTimeService, logger)

				doneChan := incrementSleepInBackground(fakeTimeService, delay)
				err := timeoutRetryStrategy.Try()
				close(doneChan)
				Expect(fakeTimeService.WatcherCount()).To(Equal(0))

				Expect(err.Error()).To(ContainSubstring("third-error"))
				Expect(retryable.Attempts).To(Equal(3))
			})

			It("stops without a trailing delay", func() {
				retryable := newSimpleRetryable([]attemptOutput{
					{
						IsRetryable: true,
						AttemptErr:  errors.New("first-error"),
					},
					{
						IsRetryable: true,
						AttemptErr:  errors.New("second-error"),
					},
					{
						IsRetryable: true,
						AttemptErr:  errors.New("third-error"),
					},
				})
				// deadline after 2nd attempt errors, but (deadline - delay) between 2nd and 3rd attempts
				delay := 20 * time.Second
				timeoutRetryStrategy := NewTimeoutRetryStrategy(25*time.Second, delay, retryable, fakeTimeService, logger)

				doneChan := incrementSleepInBackground(fakeTimeService, delay)
				err := timeoutRetryStrategy.Try()
				close(doneChan)
				Expect(fakeTimeService.WatcherCount()).To(Equal(0))

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("second-error"))
				Expect(retryable.Attempts).To(Equal(2))
			})
		})

		Context("when the attempt stops being retryable", func() {
			It("stops trying", func() {
				retryable := newSimpleRetryable([]attemptOutput{
					{
						IsRetryable: true,
						AttemptErr:  errors.New("first-error"),
					},
					{
						IsRetryable: false,
						AttemptErr:  errors.New("second-error"),
					},
				})
				timeoutRetryStrategy := NewTimeoutRetryStrategy(10*time.Second, time.Second, retryable, fakeTimeService, logger)

				doneChan := incrementSleepInBackground(fakeTimeService, time.Second)
				err := timeoutRetryStrategy.Try()
				close(doneChan)
				Expect(fakeTimeService.WatcherCount()).To(Equal(0))

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("second-error"))
				Expect(retryable.Attempts).To(Equal(2))
			})
		})

		Context("when there are no errors", func() {
			It("does not retry", func() {
				retryable := newSimpleRetryable([]attemptOutput{
					{
						IsRetryable: true,
						AttemptErr:  nil,
					},
				})
				timeoutRetryStrategy := NewTimeoutRetryStrategy(5*time.Second, 1*time.Second, retryable, fakeTimeService, logger)
				doneChan := incrementSleepInBackground(fakeTimeService, time.Second)
				err := timeoutRetryStrategy.Try()
				close(doneChan)
				Expect(fakeTimeService.WatcherCount()).To(Equal(0))

				Expect(err).ToNot(HaveOccurred())
				Expect(retryable.Attempts).To(Equal(1))
			})
		})
	})
})

func incrementSleepInBackground(fakeTimeService *fakeclock.FakeClock, delay time.Duration) chan struct{} {
	doneChan := make(chan struct{})
	go func() {
		for {
			select {
			case <-doneChan:
				return
			default:
				if fakeTimeService.WatcherCount() > 0 {
					fakeTimeService.Increment(delay)
				}
			}
		}
	}()
	return doneChan
}
