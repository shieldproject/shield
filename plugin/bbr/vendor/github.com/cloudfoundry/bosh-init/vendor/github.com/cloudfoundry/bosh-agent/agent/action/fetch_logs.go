package action

import (
	"errors"

	boshdirs "github.com/cloudfoundry/bosh-agent/settings/directories"
	boshblob "github.com/cloudfoundry/bosh-utils/blobstore"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
)

type FetchLogsAction struct {
	compressor  boshcmd.Compressor
	copier      boshcmd.Copier
	blobstore   boshblob.Blobstore
	settingsDir boshdirs.Provider
}

func NewFetchLogs(
	compressor boshcmd.Compressor,
	copier boshcmd.Copier,
	blobstore boshblob.Blobstore,
	settingsDir boshdirs.Provider,
) (action FetchLogsAction) {
	action.compressor = compressor
	action.copier = copier
	action.blobstore = blobstore
	action.settingsDir = settingsDir
	return
}

func (a FetchLogsAction) IsAsynchronous() bool {
	return true
}

func (a FetchLogsAction) IsPersistent() bool {
	return false
}

func (a FetchLogsAction) Run(logType string, filters []string) (value map[string]string, err error) {
	var logsDir string

	switch logType {
	case "job":
		if len(filters) == 0 {
			filters = []string{"**/*"}
		}
		logsDir = a.settingsDir.LogsDir()
	case "agent":
		if len(filters) == 0 {
			filters = []string{"**/*"}
		}
		logsDir = a.settingsDir.AgentLogsDir()
	default:
		err = bosherr.Error("Invalid log type")
		return
	}

	tmpDir, err := a.copier.FilteredCopyToTemp(logsDir, filters)
	if err != nil {
		err = bosherr.WrapError(err, "Copying filtered files to temp directory")
		return
	}

	defer a.copier.CleanUp(tmpDir)

	tarball, err := a.compressor.CompressFilesInDir(tmpDir)
	if err != nil {
		err = bosherr.WrapError(err, "Making logs tarball")
		return
	}

	defer func() {
		_ = a.compressor.CleanUp(tarball)
	}()

	blobID, _, err := a.blobstore.Create(tarball)
	if err != nil {
		err = bosherr.WrapError(err, "Create file on blobstore")
		return
	}

	value = map[string]string{"blobstore_id": blobID}
	return
}

func (a FetchLogsAction) Resume() (interface{}, error) {
	return nil, errors.New("not supported")
}

func (a FetchLogsAction) Cancel() error {
	return errors.New("not supported")
}
