package orchestrator

import (
	"io"
	"time"
)

//go:generate counterfeiter -o fakes/fake_backup_manager.go . BackupManager
type BackupManager interface {
	Create(string, Logger, func() time.Time) (Backup, error)
	Open(string, Logger) (Backup, error)
}

//go:generate counterfeiter -o fakes/fake_backup.go . Backup
type Backup interface {
	CreateArtifact(ArtifactIdentifier) (io.WriteCloser, error)
	ReadArtifact(ArtifactIdentifier) (io.ReadCloser, error)
	AddChecksum(ArtifactIdentifier, BackupChecksum) error
	CreateMetadataFileWithStartTime(time.Time) error
	AddFinishTime(time.Time) error
	FetchChecksum(ArtifactIdentifier) (BackupChecksum, error)
	CalculateChecksum(ArtifactIdentifier) (BackupChecksum, error)
	DeploymentMatches(string, []Instance) (bool, error)
	SaveManifest(manifest string) error
	Valid() (bool, error)
}
