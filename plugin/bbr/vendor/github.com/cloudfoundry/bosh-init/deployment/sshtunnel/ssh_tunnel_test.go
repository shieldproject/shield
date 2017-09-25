package sshtunnel

import (
	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"

	"github.com/pivotal-golang/clock/fakeclock"
)

var _ = Describe("SSH", func() {
	Describe("SSHRetryStrategy", func() {
		var (
			sshRetryStrategy         *SSHRetryStrategy
			fakeTimeService          *fakeclock.FakeClock
			connectionRefusedTimeout time.Duration
			authFailureTimeout       time.Duration
			startTime                time.Time
		)

		BeforeEach(func() {
			startTime = time.Now()
			fakeTimeService = fakeclock.NewFakeClock(startTime)
			connectionRefusedTimeout = 10 * time.Minute
			authFailureTimeout = 5 * time.Minute

			sshRetryStrategy = &SSHRetryStrategy{
				ConnectionRefusedTimeout: connectionRefusedTimeout,
				AuthFailureTimeout:       authFailureTimeout,
				TimeService:              fakeTimeService,
			}
		})

		Describe("IsRetryable", func() {
			refusedErr := errors.New("connection refused")
			authErr := errors.New("unable to authenticate")

			Context("when err is connection refused", func() {
				It("retries for connectionRefusedTimeout", func() {
					Expect(sshRetryStrategy.IsRetryable(refusedErr)).To(BeTrue())

					fakeTimeService.Increment(connectionRefusedTimeout - time.Second)
					Expect(sshRetryStrategy.IsRetryable(refusedErr)).To(BeTrue())

					fakeTimeService.Increment(time.Second)
					Expect(sshRetryStrategy.IsRetryable(refusedErr)).To(BeFalse())
				})
			})

			Context("when err is unable to authenticate", func() {
				It("retries for authFailureTimeout", func() {
					Expect(sshRetryStrategy.IsRetryable(authErr)).To(BeTrue())

					fakeTimeService.Increment(authFailureTimeout - time.Second)
					Expect(sshRetryStrategy.IsRetryable(authErr)).To(BeTrue())

					fakeTimeService.Increment(time.Second)
					Expect(sshRetryStrategy.IsRetryable(authErr)).To(BeFalse())
				})
			})

			Context("when connection is refused, then err becomes unable to authenticate", func() {
				It("retries for connectionRefusedTimeout", func() {
					Expect(sshRetryStrategy.IsRetryable(refusedErr)).To(BeTrue())

					fakeTimeService.Increment(time.Minute)
					Expect(sshRetryStrategy.IsRetryable(refusedErr)).To(BeTrue())

					fakeTimeService.Increment(authFailureTimeout - time.Second)
					Expect(sshRetryStrategy.IsRetryable(authErr)).To(BeTrue())

					fakeTimeService.Increment(time.Second)
					Expect(sshRetryStrategy.IsRetryable(authErr)).To(BeFalse())
				})
			})

			It("'no common algorithms' error fails immediately", func() {
				Expect(sshRetryStrategy.IsRetryable(errors.New("no common algorithms"))).To(BeFalse())
			})

			It("all other errors fail after the connection refused timeout", func() {
				Expect(sshRetryStrategy.IsRetryable(errors.New("another error"))).To(BeTrue())

				fakeTimeService.Increment(connectionRefusedTimeout - time.Second)
				Expect(sshRetryStrategy.IsRetryable(errors.New("another error"))).To(BeTrue())

				fakeTimeService.Increment(time.Second)
				Expect(sshRetryStrategy.IsRetryable(errors.New("another error"))).To(BeFalse())
			})
		})
	})
})
