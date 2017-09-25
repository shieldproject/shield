package infrastructure

import (
	"encoding/json"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
)

type FileSettingsSource struct {
	settingsFilePath string

	fs boshsys.FileSystem

	logger boshlog.Logger
	logTag string
}

func NewFileSettingsSource(
	settingsFilePath string,
	fs boshsys.FileSystem,
	logger boshlog.Logger,
) *FileSettingsSource {
	return &FileSettingsSource{
		settingsFilePath: settingsFilePath,

		fs: fs,

		logTag: "FileSettingsSource",
		logger: logger,
	}
}

func (s *FileSettingsSource) PublicSSHKeyForUsername(string) (string, error) {
	return "", nil
}

func (s *FileSettingsSource) Settings() (boshsettings.Settings, error) {
	var settings boshsettings.Settings

	contents, err := s.fs.ReadFile(s.settingsFilePath)
	if err != nil {
		return settings, bosherr.WrapErrorf(
			err, "Reading from file '%s'", s.settingsFilePath)
	}

	err = json.Unmarshal(contents, &settings)
	if err != nil {
		return settings, bosherr.WrapErrorf(
			err, "Parsing file settings from '%s'", s.settingsFilePath)
	}

	return settings, nil
}
