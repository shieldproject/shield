package tarball

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	bicrypto "github.com/cloudfoundry/bosh-init/crypto"
	biui "github.com/cloudfoundry/bosh-init/ui"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	bihttpclient "github.com/cloudfoundry/bosh-utils/httpclient"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshretry "github.com/cloudfoundry/bosh-utils/retrystrategy"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type Source interface {
	GetURL() string
	GetSHA1() string
	Description() string
}

type Provider interface {
	Get(Source, biui.Stage) (path string, err error)
}

var HTTPClient = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 0 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
	},
}

type provider struct {
	cache            Cache
	fs               boshsys.FileSystem
	httpClient       bihttpclient.HTTPClient
	sha1Calculator   bicrypto.SHA1Calculator
	downloadAttempts int
	delayTimeout     time.Duration
	logger           boshlog.Logger
	logTag           string
}

func NewProvider(
	cache Cache,
	fs boshsys.FileSystem,
	httpClient bihttpclient.HTTPClient,
	sha1Calculator bicrypto.SHA1Calculator,
	downloadAttempts int,
	delayTimeout time.Duration,
	logger boshlog.Logger,
) Provider {
	return &provider{
		cache:            cache,
		fs:               fs,
		httpClient:       httpClient,
		sha1Calculator:   sha1Calculator,
		downloadAttempts: downloadAttempts,
		delayTimeout:     delayTimeout,
		logger:           logger,
		logTag:           "tarballProvider",
	}
}

func (p *provider) Get(source Source, stage biui.Stage) (string, error) {
	if strings.HasPrefix(source.GetURL(), "file://") {
		filePath := strings.TrimPrefix(source.GetURL(), "file://")

		expandedPath, err := p.fs.ExpandPath(filePath)
		if err != nil {
			p.logger.Warn(p.logTag, "Failed to expand file path %s, using original URL", filePath)
			return filePath, nil
		}

		p.logger.Debug(p.logTag, "Using the tarball from file source: '%s'", filePath)
		return expandedPath, nil
	}

	if !strings.HasPrefix(source.GetURL(), "http") {
		return "", bosherr.Errorf("Invalid source URL: '%s', must be either file:// or http(s)://", source.GetURL())
	}

	var cachedPath string
	err := stage.Perform(fmt.Sprintf("Downloading %s", source.Description()), func() error {
		var found bool
		cachedPath, found = p.cache.Get(source)
		if found {
			p.logger.Debug(p.logTag, "Using the tarball from cache: '%s'", cachedPath)
			return biui.NewSkipStageError(bosherr.Error("Already downloaded"), "Found in local cache")
		}

		retryStrategy := boshretry.NewAttemptRetryStrategy(p.downloadAttempts, p.delayTimeout, p.downloadRetryable(source), p.logger)
		err := retryStrategy.Try()
		if err != nil {
			return bosherr.WrapErrorf(err, "Failed to download from '%s'", source.GetURL())
		}

		p.logger.Debug(p.logTag, "Using the downloaded tarball: '%s'", cachedPath)
		return nil
	})

	if err != nil {
		return "", err
	}

	return p.cache.Path(source), nil
}

func (p *provider) downloadRetryable(source Source) boshretry.Retryable {
	return boshretry.NewRetryable(func() (bool, error) {
		downloadedFile, err := p.fs.TempFile("tarballProvider")
		if err != nil {
			return true, bosherr.WrapError(err, "Unable to create temporary file")
		}
		defer func() {
			if err = p.fs.RemoveAll(downloadedFile.Name()); err != nil {
				p.logger.Warn(p.logTag, "Failed to remove downloaded file: %s", err.Error())
			}
		}()

		response, err := p.httpClient.Get(source.GetURL())
		if err != nil {
			return true, bosherr.WrapError(err, "Unable to download")
		}
		defer func() {
			if err = response.Body.Close(); err != nil {
				p.logger.Warn(p.logTag, "Failed to close download response body: %s", err.Error())
			}
		}()

		_, err = io.Copy(downloadedFile, response.Body)
		if err != nil {
			return true, bosherr.WrapError(err, "Saving downloaded bits to temporary file")
		}

		downloadedSha1, err := p.sha1Calculator.Calculate(downloadedFile.Name())
		if err != nil {
			return true, bosherr.WrapError(err, "Calculating sha1 for downloaded file")
		}

		if downloadedSha1 != source.GetSHA1() {
			return true, bosherr.Errorf("SHA1 of downloaded file '%s' does not match expected SHA1 '%s'", downloadedSha1, source.GetSHA1())
		}

		err = p.cache.Save(downloadedFile.Name(), source)
		if err != nil {
			return true, bosherr.WrapError(err, "Saving downloaded file in cache")
		}

		return false, nil
	})
}
