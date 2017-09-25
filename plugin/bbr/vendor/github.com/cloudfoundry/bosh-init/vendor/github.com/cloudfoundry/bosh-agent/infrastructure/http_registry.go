package infrastructure

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	boshplat "github.com/cloudfoundry/bosh-agent/platform"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type httpRegistry struct {
	metadataService   MetadataService
	platform          boshplat.Platform
	useServerNameAsID bool
}

func NewHTTPRegistry(
	metadataService MetadataService,
	platform boshplat.Platform,
	useServerNameAsID bool,
) Registry {
	return httpRegistry{
		metadataService:   metadataService,
		platform:          platform,
		useServerNameAsID: useServerNameAsID,
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
	wrapperResponse, err := http.Get(settingsURL)
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
