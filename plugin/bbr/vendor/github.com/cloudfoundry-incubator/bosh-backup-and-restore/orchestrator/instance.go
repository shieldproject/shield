package orchestrator

import (
	"io"
)

type InstanceIdentifer interface {
	Name() string
	Index() string
	ID() string
}

//go:generate counterfeiter -o fakes/fake_instance.go . Instance
type Instance interface {
	InstanceIdentifer
	IsBackupable() bool
	ArtifactDirExists() (bool, error)
	ArtifactDirCreated() bool
	MarkArtifactDirCreated()
	IsRestorable() bool
	PreBackupLock() error
	Backup() error
	PostBackupUnlock() error
	Restore() error
	Cleanup() error
	CleanupPrevious() error
	ArtifactsToBackup() []BackupArtifact
	ArtifactsToRestore() []BackupArtifact
	CustomBackupArtifactNames() []string
	CustomRestoreArtifactNames() []string
	PostRestoreUnlock() error
}

type ArtifactIdentifier interface {
	InstanceName() string
	InstanceIndex() string
	Name() string
	HasCustomName() bool
}

//go:generate counterfeiter -o fakes/fake_backup_artifact.go . BackupArtifact
type BackupArtifact interface {
	ArtifactIdentifier
	Size() (string, error)
	Checksum() (BackupChecksum, error)
	StreamFromRemote(io.Writer) error
	Delete() error
	StreamToRemote(io.Reader) error
}

type instances []Instance

func (is instances) IsEmpty() bool {
	return len(is) == 0
}

func (is instances) AllBackupable() instances {
	var backupableInstances []Instance

	for _, instance := range is {
		if instance.IsBackupable() {
			backupableInstances = append(backupableInstances, instance)
		}
	}
	return backupableInstances
}

func (is instances) CustomArtifactNames() []string {
	var artifactNames []string

	for _, instance := range is {
		artifactNames = append(artifactNames, instance.CustomBackupArtifactNames()...)
	}

	return artifactNames
}

func (is instances) RestoreArtifactNames() []string {
	var artifactNames []string

	for _, instance := range is {
		artifactNames = append(artifactNames, instance.CustomRestoreArtifactNames()...)
	}

	return artifactNames
}

func (is instances) AllRestoreable() instances {
	var instances []Instance

	for _, instance := range is {
		if instance.IsRestorable() {
			instances = append(instances, instance)
		}
	}
	return instances
}

func (is instances) AllBackupableOrRestorable() instances {
	var instances []Instance

	for _, instance := range is {
		if instance.IsBackupable() || instance.IsRestorable() {
			instances = append(instances, instance)
		}
	}
	return instances
}

func (is instances) Cleanup() error {
	var cleanupErrors []error
	for _, instance := range is {
		if err := instance.Cleanup(); err != nil {
			cleanupErrors = append(cleanupErrors, err)
		}
	}
	return ConvertErrors(cleanupErrors)
}

func (is instances) CleanupPrevious() error {
	var cleanupPreviousErrors []error
	for _, instance := range is {
		if err := instance.CleanupPrevious(); err != nil {
			cleanupPreviousErrors = append(cleanupPreviousErrors, err)
		}
	}
	return ConvertErrors(cleanupPreviousErrors)
}

func (is instances) PreBackupLock() error {
	var lockErrors []error
	for _, instance := range is {
		if err := instance.PreBackupLock(); err != nil {
			lockErrors = append(lockErrors, err)
		}
	}

	return ConvertErrors(lockErrors)
}

func (is instances) Backup() error {
	for _, instance := range is {
		err := instance.Backup()
		if err != nil {
			return err
		}
	}
	return nil
}

func (is instances) PostBackupUnlock() error {
	var unlockErrors []error
	for _, instance := range is {
		if err := instance.PostBackupUnlock(); err != nil {
			unlockErrors = append(unlockErrors, err)
		}
	}
	return ConvertErrors(unlockErrors)
}

func (is instances) Restore() error {
	for _, instance := range is {
		err := instance.Restore()
		if err != nil {
			return err
		}
	}
	return nil
}

func (is instances) PostRestoreUnlock() error {
	var unlockErrors []error
	for _, instance := range is {
		if err := instance.PostRestoreUnlock(); err != nil {
			unlockErrors = append(unlockErrors, err)
		}
	}
	return ConvertErrors(unlockErrors)
}
