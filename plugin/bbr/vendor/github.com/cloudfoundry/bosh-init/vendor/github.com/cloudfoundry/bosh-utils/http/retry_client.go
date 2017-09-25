package http

import (
	"net/http"
	"time"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshretry "github.com/cloudfoundry/bosh-utils/retrystrategy"
)

type retryClient struct {
	delegate    Client
	maxAttempts uint
	retryDelay  time.Duration
	logger      boshlog.Logger
}

func NewRetryClient(
	delegate Client,
	maxAttempts uint,
	retryDelay time.Duration,
	logger boshlog.Logger,
) Client {
	return &retryClient{
		delegate:    delegate,
		maxAttempts: maxAttempts,
		retryDelay:  retryDelay,
		logger:      logger,
	}
}

func (r *retryClient) Do(req *http.Request) (*http.Response, error) {
	requestRetryable := NewRequestRetryable(req, r.delegate, r.logger)
	retryStrategy := boshretry.NewAttemptRetryStrategy(int(r.maxAttempts), r.retryDelay, requestRetryable, r.logger)
	err := retryStrategy.Try()

	return requestRetryable.Response(), err
}
