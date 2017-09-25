package infrastructure

import (
	"encoding/json"

	boshplatform "github.com/cloudfoundry/bosh-agent/platform"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type MetadataContentsType struct {
	PublicKeys map[string]PublicKeyType `json:"public-keys"`
	InstanceID string                   `json:"instance-id"` // todo remove
}

type PublicKeyType map[string]string

type ConfigDriveSettingsSource struct {
	diskPaths    []string
	metadataPath string
	settingsPath string

	platform boshplatform.Platform

	logTag string
	logger boshlog.Logger
}

func NewConfigDriveSettingsSource(
	diskPaths []string,
	metadataPath string,
	settingsPath string,
	platform boshplatform.Platform,
	logger boshlog.Logger,
) *ConfigDriveSettingsSource {
	return &ConfigDriveSettingsSource{
		diskPaths:    diskPaths,
		metadataPath: metadataPath,
		settingsPath: settingsPath,

		platform: platform,

		logTag: "ConfigDriveSettingsSource",
		logger: logger,
	}
}

func (s *ConfigDriveSettingsSource) PublicSSHKeyForUsername(string) (string, error) {
	metadataContent, err := s.loadFileFromConfigDrive(s.metadataPath)
	if err != nil {
		return "", err
	}

	var metadata MetadataContentsType
	err = json.Unmarshal(metadataContent, &metadata)
	if err != nil {
		return "", bosherr.WrapErrorf(err, "Parsing config drive metadata from '%s'", s.metadataPath)
	}

	if firstPublicKey, ok := metadata.PublicKeys["0"]; ok {
		if openSSHKey, ok := firstPublicKey["openssh-key"]; ok {
			return openSSHKey, nil
		}
	}

	return "", nil
}

func (s *ConfigDriveSettingsSource) Settings() (boshsettings.Settings, error) {
	settingsContent, err := s.loadFileFromConfigDrive(s.settingsPath)
	if err != nil {
		return boshsettings.Settings{}, err
	}

	var settings boshsettings.Settings
	err = json.Unmarshal(settingsContent, &settings)
	if err != nil {
		return boshsettings.Settings{}, bosherr.WrapErrorf(
			err, "Parsing config drive settings from '%s'", s.settingsPath)
	}

	return settings, err
}

func (s *ConfigDriveSettingsSource) loadFileFromConfigDrive(contentPath string) ([]byte, error) {
	var err error
	var contents [][]byte

	for _, diskPath := range s.diskPaths {
		contents, err = s.platform.GetFilesContentsFromDisk(diskPath, []string{contentPath})

		if err == nil {
			s.logger.Debug(s.logTag, "Successfully loaded file '%s' from config drive: '%s'", contentPath, diskPath)
			return contents[0], nil
		}
		s.logger.Warn(s.logTag, "Failed to load config from %s - %s", diskPath, err.Error())
	}

	return []byte{}, bosherr.WrapErrorf(err, "Loading file '%s' from config drive", contentPath)
}
