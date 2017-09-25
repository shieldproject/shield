package infrastructure

import (
	"encoding/json"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type PublicKeyContent struct {
	PublicKey string `json:"public_key"`
}

type fileMetadataService struct {
	metaDataFilePath string
	userDataFilePath string
	settingsFilePath string
	fs               boshsys.FileSystem

	logger boshlog.Logger
	logTag string
}

func NewFileMetadataService(
	metaDataFilePath string,
	userDataFilePath string,
	settingsFilePath string,
	fs boshsys.FileSystem,
	logger boshlog.Logger,
) MetadataService {
	return fileMetadataService{
		metaDataFilePath: metaDataFilePath,
		userDataFilePath: userDataFilePath,
		settingsFilePath: settingsFilePath,
		fs:               fs,
		logTag:           "fileMetadataService",
		logger:           logger,
	}
}

func (ms fileMetadataService) Load() error {
	return nil
}

func (ms fileMetadataService) GetPublicKey() (string, error) {
	var p PublicKeyContent

	contents, err := ms.fs.ReadFile(ms.settingsFilePath)
	if err != nil {
		return "", bosherr.WrapError(err, "Reading metadata file")
	}

	err = json.Unmarshal([]byte(contents), &p)
	if err != nil {
		return "", bosherr.WrapError(err, "Unmarshalling metadata")
	}

	return p.PublicKey, nil
}

func (ms fileMetadataService) GetInstanceID() (string, error) {
	var metadata MetadataContentsType

	contents, err := ms.fs.ReadFile(ms.metaDataFilePath)
	if err != nil {
		return "", bosherr.WrapError(err, "Reading metadata file")
	}

	err = json.Unmarshal([]byte(contents), &metadata)
	if err != nil {
		return "", bosherr.WrapError(err, "Unmarshalling metadata")
	}

	ms.logger.Debug(ms.logTag, "Read metadata '%#v'", metadata)

	return metadata.InstanceID, nil
}

func (ms fileMetadataService) GetServerName() (string, error) {
	var userData UserDataContentsType

	contents, err := ms.fs.ReadFile(ms.userDataFilePath)
	if err != nil {
		return "", bosherr.WrapError(err, "Reading user data")
	}

	err = json.Unmarshal([]byte(contents), &userData)
	if err != nil {
		return "", bosherr.WrapError(err, "Unmarshalling user data")
	}

	ms.logger.Debug(ms.logTag, "Read user data '%#v'", userData)

	return userData.Server.Name, nil
}

func (ms fileMetadataService) GetRegistryEndpoint() (string, error) {
	var userData UserDataContentsType

	contents, err := ms.fs.ReadFile(ms.userDataFilePath)
	if err != nil {
		// Older versions of bosh-warden-cpi placed
		// full settings file at a specific location.
		return ms.settingsFilePath, nil
	}

	err = json.Unmarshal([]byte(contents), &userData)
	if err != nil {
		return "", bosherr.WrapError(err, "Unmarshalling user data")
	}

	ms.logger.Debug(ms.logTag, "Read user data '%#v'", userData)

	return userData.Registry.Endpoint, nil
}

func (ms fileMetadataService) GetNetworks() (boshsettings.Networks, error) {
	var userData UserDataContentsType

	contents, err := ms.fs.ReadFile(ms.userDataFilePath)
	if err != nil {
		return nil, bosherr.WrapError(err, "Reading user data")
	}

	err = json.Unmarshal([]byte(contents), &userData)
	if err != nil {
		return nil, bosherr.WrapError(err, "Unmarshalling user data")
	}

	ms.logger.Debug(ms.logTag, "Read user data '%#v'", userData)

	return userData.Networks, nil
}

func (ms fileMetadataService) IsAvailable() bool {
	return ms.fs.FileExists(ms.settingsFilePath)
}
