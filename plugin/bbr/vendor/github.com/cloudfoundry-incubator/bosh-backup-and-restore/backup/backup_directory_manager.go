package backup

import (
	"os"

	"time"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/pkg/errors"
)

type BackupDirectoryManager struct{}

func (BackupDirectoryManager) Create(name string, logger orchestrator.Logger, nowFunc func() time.Time) (orchestrator.Backup, error) {
	directoryName := name + "_" + nowFunc().UTC().Format("20060102T150405Z")
	err := os.Mkdir(directoryName, 0700)
	return &BackupDirectory{baseDirName: directoryName, Logger: logger}, errors.Wrap(err, "failed creating directory")
}

func (BackupDirectoryManager) Open(name string, logger orchestrator.Logger) (orchestrator.Backup, error) {
	_, err := os.Stat(name)
	return &BackupDirectory{baseDirName: name, Logger: logger}, errors.Wrap(err, "failed opening the directory")
}
