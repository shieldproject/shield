package monit

import (
	"net/http"
	"time"

	"code.cloudfoundry.org/clock"
	"github.com/cloudfoundry/bosh-utils/httpclient"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type monitRetryClient struct {
	delegate               HTTPClient
	maxUnavailableAttempts uint
	maxOtherAttempts       uint
	retryDelay             time.Duration
	logger                 boshlog.Logger
}

func NewMonitRetryClient(
	delegate HTTPClient,
	maxUnavailableAttempts uint,
	maxOtherAttempts uint,
	retryDelay time.Duration,
	logger boshlog.Logger,
) httpclient.Client {
	return &monitRetryClient{
		delegate:               delegate,
		maxUnavailableAttempts: maxUnavailableAttempts,
		maxOtherAttempts:       maxOtherAttempts,
		retryDelay:             retryDelay,
		logger:                 logger,
	}
}

func (r *monitRetryClient) Do(req *http.Request) (*http.Response, error) {
	requestRetryable := httpclient.NewRequestRetryable(req, r.delegate, r.logger, nil)
	timeService := clock.NewClock()
	retryStrategy := NewMonitRetryStrategy(
		requestRetryable,
		r.maxUnavailableAttempts,
		r.maxOtherAttempts,
		r.retryDelay,
		timeService,
	)

	err := retryStrategy.Try()

	return requestRetryable.Response(), err
}
