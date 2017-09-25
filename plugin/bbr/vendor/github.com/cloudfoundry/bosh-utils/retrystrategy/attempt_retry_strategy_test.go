package retrystrategy_test

import (
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	fakelogger "github.com/cloudfoundry/bosh-utils/logger/fakes"
	. "github.com/cloudfoundry/bosh-utils/retrystrategy"
)

var _ = Describe("AttemptRetryStrategy", func() {
	var (
		logger *fakelogger.FakeLogger
	)

	BeforeEach(func() {
		logger = &fakelogger.FakeLogger{}
	})

	Describe("Try", func() {
		It("includes type of retryable in log message", func() {
			retryable := newSimpleRetryable([]attemptOutput{
				{
					IsRetryable: true,
					AttemptErr:  nil,
				},
			})

			attemptRetryStrategy := NewAttemptRetryStrategy(3, 0, retryable, logger)
			err := attemptRetryStrategy.Try()
			Expect(err).ToNot(HaveOccurred())
			Expect(logger.DebugCallCount()).To(Equal(1))

			_, message, args := logger.DebugArgsForCall(0)
			Expect(fmt.Sprintf(message, args...)).To(Equal("Making attempt #0 for *retrystrategy_test.simpleRetryable"))
		})

		Context("when there are errors during a try", func() {
			It("retries until the max attempts are used up", func() {
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
				attemptRetryStrategy := NewAttemptRetryStrategy(3, 0, retryable, logger)
				err := attemptRetryStrategy.Try()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("third-error"))
				Expect(retryable.Attempts).To(Equal(3))
			})
		})

		Context("when the attempt is not retryable", func() {
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
				attemptRetryStrategy := NewAttemptRetryStrategy(10, 0, retryable, logger)
				err := attemptRetryStrategy.Try()
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
				attemptRetryStrategy := NewAttemptRetryStrategy(3, 0, retryable, logger)
				err := attemptRetryStrategy.Try()
				Expect(err).ToNot(HaveOccurred())
				Expect(retryable.Attempts).To(Equal(1))
			})
		})
	})
})
