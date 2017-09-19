package retrystrategy_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	. "github.com/cloudfoundry/bosh-utils/retrystrategy"
)

var _ = Describe("UnlimitedRetryStrategy", func() {
	var (
		logger boshlog.Logger
	)

	BeforeEach(func() {
		logger = boshlog.NewLogger(boshlog.LevelNone)
	})

	Describe("Try", func() {
		It("stops retrying when it receives a non-retryable error", func() {
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
				{
					IsRetryable: true,
					AttemptErr:  errors.New("fifth-error"),
				},
				{
					IsRetryable: true,
					AttemptErr:  errors.New("sixth-error"),
				},
				{
					IsRetryable: false,
					AttemptErr:  errors.New("seventh-error"),
				},
			})
			attemptRetryStrategy := NewUnlimitedRetryStrategy(0, retryable, logger)
			err := attemptRetryStrategy.Try()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("seventh-error"))
			Expect(retryable.Attempts).To(Equal(7))
		})

		It("stops retrying when it stops receiving errors", func() {
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
				{
					IsRetryable: true,
					AttemptErr:  errors.New("fifth-error"),
				},
				{
					IsRetryable: true,
					AttemptErr:  errors.New("sixth-error"),
				},
				{
					IsRetryable: true,
					AttemptErr:  nil,
				},
			})
			attemptRetryStrategy := NewUnlimitedRetryStrategy(0, retryable, logger)
			err := attemptRetryStrategy.Try()
			Expect(err).ToNot(HaveOccurred())
			Expect(retryable.Attempts).To(Equal(7))
		})
	})
})
