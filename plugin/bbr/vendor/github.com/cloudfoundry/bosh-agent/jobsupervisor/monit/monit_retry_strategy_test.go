package monit_test

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	boshretry "github.com/cloudfoundry/bosh-utils/retrystrategy"

	fakeboshaction "github.com/cloudfoundry/bosh-agent/agent/action/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/jobsupervisor/monit"
	"github.com/cloudfoundry/bosh-agent/jobsupervisor/monit/monitfakes"
)

var _ = Describe("MonitRetryStrategy", func() {
	var (
		retryable              *monitfakes.FakeRequestRetryable
		monitRetryStrategy     boshretry.RetryStrategy
		maxUnavailableAttempts int
		maxOtherAttempts       int
		timeService            *fakeboshaction.FakeClock
		delay                  time.Duration
	)

	BeforeEach(func() {
		maxUnavailableAttempts = 6
		maxOtherAttempts = 7
		retryable = &monitfakes.FakeRequestRetryable{}
		timeService = &fakeboshaction.FakeClock{}
		delay = 10 * time.Millisecond
		monitRetryStrategy = NewMonitRetryStrategy(
			retryable,
			uint(maxUnavailableAttempts),
			uint(maxOtherAttempts),
			delay,
			timeService,
		)
	})

	Describe("Try", func() {
		var (
			lastError   error
			unavailable *http.Response
			notFound    *http.Response
		)

		BeforeEach(func() {
			lastError = errors.New("last-error")
			unavailable = &http.Response{StatusCode: 503, Body: ioutil.NopCloser(bytes.NewBufferString(""))}
			notFound = &http.Response{StatusCode: 404, Body: ioutil.NopCloser(bytes.NewBufferString(""))}
		})

		Context("when all responses are only 503s", func() {
			It("retries until maxUnavailableAttempts + maxOtherAttempts are exhausted", func() {
				retryable.ResponseReturns(unavailable)

				retryable.AttemptStub = func() (bool, error) {
					if retryable.AttemptCallCount() < maxUnavailableAttempts+maxOtherAttempts-1 {
						return true, errors.New("fake-error")
					}

					return true, lastError
				}

				errChan := tryInBackground(monitRetryStrategy)

				Eventually(errChan).Should(Receive(Equal(lastError)))
				Expect(retryable.AttemptCallCount()).To(Equal(maxUnavailableAttempts + maxOtherAttempts))
			})
		})

		Context("when all requests cancelled", func() {
			It("retries until maxUnavailableAttempts + maxOtherAttempts are exhausted", func() {
				retryable.AttemptStub = func() (bool, error) {
					if retryable.AttemptCallCount() < maxUnavailableAttempts+maxOtherAttempts-1 {
						retryable.ResponseReturns(nil)
						return true, errors.New("net/http: request canceled")
					}

					retryable.ResponseReturns(unavailable)
					return true, lastError
				}

				errChan := tryInBackground(monitRetryStrategy)

				Eventually(errChan).Should(Receive(Equal(lastError)))
				Expect(retryable.AttemptCallCount()).To(Equal(maxUnavailableAttempts + maxOtherAttempts))
			})
		})

		Context("when there are both 503 and canceled responses", func() {
			It("retries until maxUnavailableAttempts + maxOtherAttempts are exhausted", func() {
				retryable.AttemptStub = func() (bool, error) {
					if retryable.AttemptCallCount() < maxUnavailableAttempts+maxOtherAttempts-1 {
						if retryable.AttemptCallCount()%2 == 0 {
							retryable.ResponseReturns(nil)
							return true, errors.New("net/http: request canceled")
						}

						retryable.ResponseReturns(unavailable)
						return true, errors.New("fake-error")
					}

					retryable.ResponseReturns(unavailable)
					return true, lastError
				}

				errChan := tryInBackground(monitRetryStrategy)

				Eventually(errChan).Should(Receive(Equal(lastError)))
				Expect(retryable.AttemptCallCount()).To(Equal(maxUnavailableAttempts + maxOtherAttempts))
			})
		})

		Context("when there are < maxUnavailableAttempts initial 503s", func() {
			var expectedAttempts int

			BeforeEach(func() {
				expectedAttempts = maxUnavailableAttempts + maxOtherAttempts - 1
			})

			Context("when maxOtherAttempts non-503 errors", func() {
				It("retries the unavailable then until maxOtherAttempts are exhausted", func() {
					retryable.AttemptStub = func() (bool, error) {
						if retryable.AttemptCallCount() < maxUnavailableAttempts {
							retryable.ResponseReturns(unavailable)
							return true, errors.New("unavailable-error")
						} else if retryable.AttemptCallCount() < maxUnavailableAttempts+maxOtherAttempts-1 {
							retryable.ResponseReturns(notFound)
							return true, errors.New("not-found-error")
						}

						return true, lastError
					}

					errChan := tryInBackground(monitRetryStrategy)

					Eventually(errChan).Should(Receive(Equal(lastError)))
					Expect(retryable.AttemptCallCount()).To(Equal(expectedAttempts))
				})
			})

			Context("when maxOtherAttempts include 503s after non-503", func() {
				It("retries the unavailable then until maxOtherAttempts are exhausted", func() {
					retryable.AttemptStub = func() (bool, error) {
						if retryable.AttemptCallCount() < maxUnavailableAttempts {
							retryable.ResponseReturns(unavailable)
							return true, errors.New("unavailable-error")
						} else if retryable.AttemptCallCount() == maxUnavailableAttempts {
							retryable.ResponseReturns(notFound)
							return true, errors.New("not-found-error")
						} else if retryable.AttemptCallCount() < maxUnavailableAttempts-maxOtherAttempts-1 {
							retryable.ResponseReturns(unavailable)
							return true, errors.New("unavailable-error")
						}

						retryable.ResponseReturns(unavailable)
						return true, lastError
					}

					errChan := tryInBackground(monitRetryStrategy)

					Eventually(errChan).Should(Receive(Equal(lastError)))
					Expect(retryable.AttemptCallCount()).To(Equal(expectedAttempts))
				})
			})
		})

		Context("when the initial attempt is a non-503 error", func() {
			It("retries for maxOtherAttempts", func() {
				retryable.AttemptStub = func() (bool, error) {
					retryable.ResponseReturns(notFound)
					if retryable.AttemptCallCount() < maxOtherAttempts-2 {
						return true, errors.New("not-found-error")
					}

					return true, lastError
				}

				errChan := tryInBackground(monitRetryStrategy)

				Eventually(errChan).Should(Receive(Equal(lastError)))
				Expect(retryable.AttemptCallCount()).To(Equal(maxOtherAttempts))
			})

			Context("when other attempts are all unavailble", func() {
				It("retries for maxOtherAttempts", func() {
					retryable.AttemptStub = func() (bool, error) {
						if retryable.AttemptCallCount() == 1 {
							retryable.ResponseReturns(notFound)
							return true, errors.New("not-found-error")
						} else if retryable.AttemptCallCount() < maxOtherAttempts-1 {
							retryable.ResponseReturns(unavailable)
							return true, errors.New("unavailable-error")
						}

						retryable.ResponseReturns(unavailable)
						return true, lastError
					}

					errChan := tryInBackground(monitRetryStrategy)

					Eventually(errChan).Should(Receive(Equal(lastError)))
					Expect(retryable.AttemptCallCount()).To(Equal(maxOtherAttempts))
				})
			})
		})

		It("waits for retry delay between retries", func() {
			retryable.AttemptStub = func() (bool, error) {
				retryable.ResponseReturns(unavailable)
				return true, lastError
			}

			errChan := tryInBackground(monitRetryStrategy)

			Eventually(errChan).Should(Receive(Equal(lastError)))
			Expect(timeService.SleepCallCount()).To(Equal(maxUnavailableAttempts + maxOtherAttempts))
		})

		Context("when error is not due to failed response", func() {
			It("retries until maxOtherAttempts are exhausted", func() {
				retryable.AttemptStub = func() (bool, error) {
					retryable.ResponseReturns(nil)

					if retryable.AttemptCallCount() < maxOtherAttempts-1 {
						return true, errors.New("request error")
					}

					return true, lastError
				}

				errChan := tryInBackground(monitRetryStrategy)

				Eventually(errChan).Should(Receive(Equal(lastError)))
				Expect(retryable.AttemptCallCount()).To(Equal(maxOtherAttempts))
			})
		})

		Context("when attempt is not retryable", func() {
			It("does not retry", func() {
				retryable.AttemptReturns(false, lastError)

				err := monitRetryStrategy.Try()
				Expect(err).To(Equal(lastError))

				Expect(retryable.AttemptCallCount()).To(Equal(1))
			})
		})
	})
})

func tryInBackground(monitRetryStrategy boshretry.RetryStrategy) chan error {
	errChan := make(chan error)
	go func() {
		errChan <- monitRetryStrategy.Try()
	}()
	return errChan
}
