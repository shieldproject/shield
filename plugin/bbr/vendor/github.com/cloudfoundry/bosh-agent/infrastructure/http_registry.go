package infrastructure

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	boshplat "github.com/cloudfoundry/bosh-agent/platform"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"github.com/cloudfoundry/bosh-utils/httpclient"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type httpRegistry struct {
	metadataService   MetadataService
	platform          boshplat.Platform
	useServerNameAsID bool
	logger            boshlog.Logger
	retryDelay        time.Duration
}

func NewHTTPRegistryWithCustomDelay(
	metadataService MetadataService,
	platform boshplat.Platform,
	useServerNameAsID bool,
	logger boshlog.Logger,
	retryDelay time.Duration,
) Registry {
	return httpRegistry{
		metadataService:   metadataService,
		platform:          platform,
		useServerNameAsID: useServerNameAsID,
		logger:            logger,
		retryDelay:        retryDelay,
	}
}

func NewHTTPRegistry(
	metadataService MetadataService,
	platform boshplat.Platform,
	useServerNameAsID bool,
	logger boshlog.Logger,
) Registry {
	return httpRegistry{
		metadataService:   metadataService,
		platform:          platform,
		useServerNameAsID: useServerNameAsID,
		logger:            logger,
		retryDelay:        1 * time.Second,
	}
}

type settingsWrapperType struct {
	Settings string
}

func (r httpRegistry) GetSettings() (boshsettings.Settings, error) {
	var settings boshsettings.Settings

	var identifier string
	var err error

	if r.useServerNameAsID {
		identifier, err = r.metadataService.GetServerName()
		if err != nil {
			return settings, bosherr.WrapError(err, "Getting server name")
		}
	} else {
		identifier, err = r.metadataService.GetInstanceID()
		if err != nil {
			return settings, bosherr.WrapError(err, "Getting instance id")
		}
	}

	registryEndpoint, err := r.metadataService.GetRegistryEndpoint()
	if err != nil {
		return settings, bosherr.WrapError(err, "Getting registry endpoint")
	}

	networks, err := r.metadataService.GetNetworks()
	if err != nil {
		return settings, bosherr.WrapError(err, "Getting networks")
	}

	if len(networks) > 0 {
		err = r.platform.SetupNetworking(networks)
		if err != nil {
			return settings, bosherr.WrapError(err, "Setting up networks")
		}
	}

	settingsURL := fmt.Sprintf("%s/instances/%s/settings", registryEndpoint, identifier)
	client := httpclient.NewRetryClient(httpclient.CreateDefaultClient(nil), 10, r.retryDelay, r.logger)
	wrapperResponse, err := httpclient.NewHTTPClient(client, r.logger).Get(settingsURL)
	if err != nil {
		return settings, bosherr.WrapError(err, "Getting settings from url")
	}

	defer func() {
		_ = wrapperResponse.Body.Close()
	}()

	wrapperBytes, err := ioutil.ReadAll(wrapperResponse.Body)
	if err != nil {
		return settings, bosherr.WrapError(err, "Reading settings response body")
	}

	var wrapper settingsWrapperType

	err = json.Unmarshal(wrapperBytes, &wrapper)
	if err != nil {
		return settings, bosherr.WrapError(err, "Unmarshalling settings wrapper")
	}

	err = json.Unmarshal([]byte(wrapper.Settings), &settings)
	if err != nil {
		return settings, bosherr.WrapError(err, "Unmarshalling wrapped settings")
	}

	return settings, nil
}
