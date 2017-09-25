package state

import (
	"encoding/json"

	boshplatform "github.com/cloudfoundry/bosh-agent/platform"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
)

type SyncDNSState struct {
	platform      boshplatform.Platform
	fs            boshsys.FileSystem
	path          string
	uuidGenerator boshuuid.Generator
}

func NewSyncDNSState(platform boshplatform.Platform, path string, generator boshuuid.Generator) SyncDNSState {
	return SyncDNSState{
		platform:      platform,
		fs:            platform.GetFs(),
		path:          path,
		uuidGenerator: generator,
	}
}

func (s SyncDNSState) SaveState(localDNSState []byte) error {
	uuid, err := s.uuidGenerator.Generate()
	if err != nil {
		return bosherr.WrapError(err, "generating uuid for temp file")
	}

	tmpFilePath := s.path + uuid

	err = s.fs.WriteFileQuietly(tmpFilePath, localDNSState)
	if err != nil {
		return bosherr.WrapError(err, "writing the blobstore DNS state")
	}

	err = s.platform.SetupRecordsJSONPermission(tmpFilePath)
	if err != nil {
		return bosherr.WrapError(err, "setting permissions of blobstore DNS state")
	}

	err = s.fs.Rename(tmpFilePath, s.path)
	if err != nil {
		return bosherr.WrapError(err, "renaming")
	}

	return nil
}

func (s SyncDNSState) NeedsUpdate(newVersion uint64) bool {
	if !s.fs.FileExists(s.path) {
		return true
	}

	version, err := s.loadVersion()
	if err != nil {
		return true
	}

	return version < newVersion
}

func (s SyncDNSState) loadVersion() (uint64, error) {
	contents, err := s.fs.ReadFile(s.path)
	if err != nil {
		return 0, bosherr.WrapError(err, "reading state file")
	}

	var localVersion struct {
		Version uint64 `json:"version"`
	}

	err = json.Unmarshal(contents, &localVersion)
	if err != nil {
		return 0, bosherr.WrapError(err, "unmarshalling state file")
	}

	return localVersion.Version, nil
}
