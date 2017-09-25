package monit

import (
	"strings"
	"time"

	boshhttp "github.com/cloudfoundry/bosh-utils/http"
	boshretry "github.com/cloudfoundry/bosh-utils/retrystrategy"
	"github.com/pivotal-golang/clock"
)

type monitRetryStrategy struct {
	retryable boshhttp.RequestRetryable

	maxUnavailableAttempts uint
	maxOtherAttempts       uint

	delay       time.Duration
	timeService clock.Clock

	unavailableAttempts uint
	otherAttempts       uint
}

func NewMonitRetryStrategy(
	retryable boshhttp.RequestRetryable,
	maxUnavailableAttempts uint,
	maxOtherAttempts uint,
	delay time.Duration,
	timeService clock.Clock,
) boshretry.RetryStrategy {
	return &monitRetryStrategy{
		retryable:              retryable,
		maxUnavailableAttempts: maxUnavailableAttempts,
		maxOtherAttempts:       maxOtherAttempts,
		unavailableAttempts:    0,
		otherAttempts:          0,
		delay:                  delay,
		timeService:            timeService,
	}
}

func (m *monitRetryStrategy) Try() error {
	var err error
	var isRetryable bool

	for m.hasMoreAttempts() {
		isRetryable, err = m.retryable.Attempt()
		if !isRetryable {
			break
		}

		is503 := m.retryable.Response() != nil && m.retryable.Response().StatusCode == 503
		isCanceled := err != nil && strings.Contains(err.Error(), "request canceled")

		if (is503 || isCanceled) && m.unavailableAttempts < m.maxUnavailableAttempts {
			m.unavailableAttempts = m.unavailableAttempts + 1
		} else {
			// once a non-503 error is received, all errors count as 'other' errors
			m.unavailableAttempts = m.maxUnavailableAttempts + 1
			m.otherAttempts = m.otherAttempts + 1
		}

		m.timeService.Sleep(m.delay)
	}

	if err != nil && m.retryable.Response() != nil {
		_ = m.retryable.Response().Body.Close()
	}

	return err
}

func (m *monitRetryStrategy) hasMoreAttempts() bool {
	return m.unavailableAttempts < m.maxUnavailableAttempts || m.otherAttempts < m.maxOtherAttempts
}
