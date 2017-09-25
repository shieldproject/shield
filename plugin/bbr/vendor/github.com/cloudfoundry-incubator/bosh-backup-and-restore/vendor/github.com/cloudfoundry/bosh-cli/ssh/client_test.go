package ssh_test

import (
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/clock/fakeclock"

	. "github.com/cloudfoundry/bosh-cli/ssh"
)

var _ = Describe("ClientConnectRetryStrategy", func() {
	var (
		connectionRefusedTimeout time.Duration
		authFailureTimeout       time.Duration

		timeService *fakeclock.FakeClock
		strategy    *ClientConnectRetryStrategy
	)

	BeforeEach(func() {
		timeService = fakeclock.NewFakeClock(time.Now())
		connectionRefusedTimeout = 10 * time.Minute
		authFailureTimeout = 5 * time.Minute

		strategy = &ClientConnectRetryStrategy{
			ConnectionRefusedTimeout: connectionRefusedTimeout,
			AuthFailureTimeout:       authFailureTimeout,
			TimeService:              timeService,
		}
	})

	Describe("IsRetryable", func() {
		refusedErr := errors.New("connection refused")
		authErr := errors.New("unable to authenticate")

		Context("when err is connection refused", func() {
			It("retries for connectionRefusedTimeout", func() {
				Expect(strategy.IsRetryable(refusedErr)).To(BeTrue())

				timeService.Increment(connectionRefusedTimeout - time.Second)
				Expect(strategy.IsRetryable(refusedErr)).To(BeTrue())

				timeService.Increment(time.Second)
				Expect(strategy.IsRetryable(refusedErr)).To(BeFalse())
			})
		})

		Context("when err is unable to authenticate", func() {
			It("retries for authFailureTimeout", func() {
				Expect(strategy.IsRetryable(authErr)).To(BeTrue())

				timeService.Increment(authFailureTimeout - time.Second)
				Expect(strategy.IsRetryable(authErr)).To(BeTrue())

				timeService.Increment(time.Second)
				Expect(strategy.IsRetryable(authErr)).To(BeFalse())
			})
		})

		Context("when connection is refused, then err becomes unable to authenticate", func() {
			It("retries for connectionRefusedTimeout", func() {
				Expect(strategy.IsRetryable(refusedErr)).To(BeTrue())

				timeService.Increment(time.Minute)
				Expect(strategy.IsRetryable(refusedErr)).To(BeTrue())

				timeService.Increment(authFailureTimeout - time.Second)
				Expect(strategy.IsRetryable(authErr)).To(BeTrue())

				timeService.Increment(time.Second)
				Expect(strategy.IsRetryable(authErr)).To(BeFalse())
			})
		})

		It("'no common algorithms' error fails immediately", func() {
			Expect(strategy.IsRetryable(errors.New("no common algorithms"))).To(BeFalse())
		})

		It("all other errors fail after the connection refused timeout", func() {
			Expect(strategy.IsRetryable(errors.New("another error"))).To(BeTrue())

			timeService.Increment(connectionRefusedTimeout - time.Second)
			Expect(strategy.IsRetryable(errors.New("another error"))).To(BeTrue())

			timeService.Increment(time.Second)
			Expect(strategy.IsRetryable(errors.New("another error"))).To(BeFalse())
		})
	})
})
