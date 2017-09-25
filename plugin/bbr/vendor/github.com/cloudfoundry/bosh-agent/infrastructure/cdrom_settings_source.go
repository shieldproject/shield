package infrastructure

import (
	"encoding/json"

	boshplatform "github.com/cloudfoundry/bosh-agent/platform"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type CDROMSettingsSource struct {
	settingsFileName string

	platform boshplatform.Platform

	logTag string
	logger boshlog.Logger
}

func NewCDROMSettingsSource(
	settingsFileName string,
	platform boshplatform.Platform,
	logger boshlog.Logger,
) *CDROMSettingsSource {
	return &CDROMSettingsSource{
		settingsFileName: settingsFileName,

		platform: platform,

		logTag: "CDROMSettingsSource",
		logger: logger,
	}
}

func (s CDROMSettingsSource) PublicSSHKeyForUsername(string) (string, error) {
	return "", nil
}

func (s *CDROMSettingsSource) Settings() (boshsettings.Settings, error) {
	var settings boshsettings.Settings

	contents, err := s.platform.GetFileContentsFromCDROM(s.settingsFileName)
	if err != nil {
		return settings, bosherr.WrapError(err, "Reading files from CDROM")
	}

	err = json.Unmarshal(contents, &settings)
	if err != nil {
		return settings, bosherr.WrapErrorf(
			err, "Parsing CDROM settings from '%s'", s.settingsFileName)
	}

	return settings, nil
}
