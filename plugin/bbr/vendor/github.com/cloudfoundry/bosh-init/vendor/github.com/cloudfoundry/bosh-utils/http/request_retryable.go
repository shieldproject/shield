package http

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
)

type RequestRetryable interface {
	Attempt() (bool, error)
	Response() *http.Response
}

type requestRetryable struct {
	request   *http.Request
	requestID string
	delegate  Client
	attempt   int

	bodyBytes []byte // buffer request body to memory for retries
	response  *http.Response

	uuidGenerator boshuuid.Generator

	logger boshlog.Logger
	logTag string
}

func NewRequestRetryable(
	request *http.Request,
	delegate Client,
	logger boshlog.Logger,
) RequestRetryable {
	return &requestRetryable{
		request:       request,
		delegate:      delegate,
		attempt:       0,
		uuidGenerator: boshuuid.NewGenerator(),
		logger:        logger,
		logTag:        "clientRetryable",
	}
}

func (r *requestRetryable) Attempt() (bool, error) {
	var err error

	if r.requestID == "" {
		r.requestID, err = r.uuidGenerator.Generate()
		if err != nil {
			return false, bosherr.WrapError(err, "Generating request uuid")
		}
	}

	if r.request.Body != nil && r.bodyBytes == nil {
		r.bodyBytes, err = ReadAndClose(r.request.Body)
		if err != nil {
			return false, bosherr.WrapError(err, "Buffering request body")
		}
	}

	// reset request body, because readers cannot be re-read
	if r.bodyBytes != nil {
		r.request.Body = ioutil.NopCloser(bytes.NewReader(r.bodyBytes))
	}

	// close previous attempt's response body to prevent HTTP client resource leaks
	if r.response != nil {
		ioutil.ReadAll(r.response.Body)
		r.response.Body.Close()
	}

	r.attempt++

	r.logger.Debug(r.logTag, "[requestID=%s] Requesting (attempt=%d): %s", r.requestID, r.attempt, r.formatRequest(r.request))
	r.response, err = r.delegate.Do(r.request)
	if err != nil {
		r.logger.Debug(r.logTag, "[requestID=%s] Request attempt failed (attempts=%d), error: %s", r.requestID, r.attempt, err)
		return true, err
	}

	if r.wasSuccessful(r.response) {
		r.logger.Debug(r.logTag, "[requestID=%s] Request succeeded (attempts=%d), response: %s", r.requestID, r.attempt, r.formatResponse(r.response))
		return false, nil
	}

	r.logger.Debug(r.logTag, "[requestID=%s] Request attempt failed (attempts=%d), response: %s", r.requestID, r.attempt, r.formatResponse(r.response))
	return true, bosherr.Errorf("Request failed, response: %s", r.formatResponse(r.response))
}

func (r *requestRetryable) Response() *http.Response {
	return r.response
}

func (r *requestRetryable) wasSuccessful(resp *http.Response) bool {
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func (r *requestRetryable) formatRequest(req *http.Request) string {
	if req == nil {
		return "Request(nil)"
	}

	return fmt.Sprintf("Request{ Method: '%s', URL: '%s' }", req.Method, req.URL)
}

func (r *requestRetryable) formatResponse(resp *http.Response) string {
	if resp == nil {
		return "Response(nil)"
	}

	return fmt.Sprintf("Response{ StatusCode: %d, Status: '%s' }", resp.StatusCode, resp.Status)
}
