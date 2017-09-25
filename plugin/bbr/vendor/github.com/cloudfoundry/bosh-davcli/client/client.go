package client

import (
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	davconf "github.com/cloudfoundry/bosh-davcli/config"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"github.com/cloudfoundry/bosh-utils/httpclient"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type Client interface {
	Get(path string) (content io.ReadCloser, err error)
	Put(path string, content io.ReadCloser, contentLength int64) (err error)
	Exists(path string) (err error)
	Delete(path string) (err error)
}

func NewClient(config davconf.Config, httpClient httpclient.Client, logger boshlog.Logger) (c Client) {
	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}

	// @todo should a logger now be passed in to this client?
	duration := time.Duration(0)
	retryClient := httpclient.NewRetryClient(
		httpClient,
		config.RetryAttempts,
		duration,
		logger,
	)

	return client{
		config:     config,
		httpClient: retryClient,
	}
}

type client struct {
	config     davconf.Config
	httpClient httpclient.Client
}

func (c client) Get(path string) (content io.ReadCloser, err error) {
	req, err := c.createReq("GET", path, nil)
	if err != nil {
		return
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		err = bosherr.WrapErrorf(err, "Getting dav blob %s", path)
		return
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("Getting dav blob %s: Wrong response code: %d; body: %s", path, resp.StatusCode, c.readAndTruncateBody(resp))
		return
	}

	content = resp.Body
	return
}

func (c client) Put(path string, content io.ReadCloser, contentLength int64) (err error) {
	req, err := c.createReq("PUT", path, content)
	if err != nil {
		return
	}
	defer content.Close()
	req.ContentLength = contentLength

	resp, err := c.httpClient.Do(req)
	if err != nil {
		err = bosherr.WrapErrorf(err, "Putting dav blob %s", path)
		return
	}

	if resp.StatusCode != 201 && resp.StatusCode != 204 {
		err = fmt.Errorf("Putting dav blob %s: Wrong response code: %d; body: %s", path, resp.StatusCode, c.readAndTruncateBody(resp))
		return
	}

	return
}

func (c client) Exists(path string) (err error) {
	req, err := c.createReq("HEAD", path, nil)
	if err != nil {
		return
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			err = fmt.Errorf("%s not found", path)
		}
		err = bosherr.WrapErrorf(err, "Checking if dav blob %s exists", path)
		return
	}

	return
}

func (c client) Delete(path string) (err error) {
	req, err := c.createReq("DELETE", path, nil)
	if err != nil {
		err = bosherr.WrapErrorf(err, "Creating delete request for blob '%s'", path)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			err = nil
		} else {
			err = bosherr.WrapErrorf(err, "Deleting blob '%s'", path)
		}
	}

	return
}

func (c client) createReq(method, blobID string, body io.Reader) (req *http.Request, err error) {
	blobURL, err := url.Parse(c.config.Endpoint)
	if err != nil {
		return
	}

	digester := sha1.New()
	digester.Write([]byte(blobID))
	blobPrefix := fmt.Sprintf("%02x", digester.Sum(nil)[0])

	newPath := path.Join(blobURL.Path, blobPrefix, blobID)
	if !strings.HasPrefix(newPath, "/") {
		newPath = "/" + newPath
	}

	blobURL.Path = newPath

	req, err = http.NewRequest(method, blobURL.String(), body)
	if err != nil {
		return
	}

	req.SetBasicAuth(c.config.User, c.config.Password)
	return
}

func (c client) readAndTruncateBody(resp *http.Response) string {
	body := ""
	if resp.Body != nil {
		buf := make([]byte, 1024)
		n, err := resp.Body.Read(buf)
		if err == io.EOF || err == nil {
			body = string(buf[0:n])
		}
	}
	return body
}
