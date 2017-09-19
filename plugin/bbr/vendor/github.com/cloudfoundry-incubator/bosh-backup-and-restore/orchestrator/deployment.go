package orchestrator

import (
	"fmt"

	"strings"

	"github.com/pkg/errors"
)

const ArtifactDirectory = "/var/vcap/store/bbr-backup"

//go:generate counterfeiter -o fakes/fake_deployment.go . Deployment
type Deployment interface {
	IsBackupable() bool
	HasUniqueCustomArtifactNames() bool
	CheckArtifactDir() error
	IsRestorable() bool
	PreBackupLock() error
	Backup() error
	PostBackupUnlock() error
	Restore() error
	CopyRemoteBackupToLocal(Backup) error
	CopyLocalBackupToRemote(Backup) error
	Cleanup() error
	CleanupPrevious() error
	Instances() []Instance
	CustomArtifactNamesMatch() error
	PostRestoreUnlock() error
}

type deployment struct {
	Logger

	instances instances
}

func NewDeployment(logger Logger, instancesArray []Instance) Deployment {
	return &deployment{Logger: logger, instances: instances(instancesArray)}
}

func (bd *deployment) IsBackupable() bool {
	backupableInstances := bd.instances.AllBackupable()
	return !backupableInstances.IsEmpty()
}

func (bd *deployment) HasUniqueCustomArtifactNames() bool {
	names := bd.instances.CustomArtifactNames()

	uniqueNames := map[string]bool{}
	for _, name := range names {
		if _, found := uniqueNames[name]; found {
			return false
		}
		uniqueNames[name] = true
	}
	return true
}

func (bd *deployment) CheckArtifactDir() error {
	errs := []string{}

	for _, inst := range bd.instances {
		exists, err := inst.ArtifactDirExists()
		if err != nil {
			errs = append(errs, fmt.Sprintf("Error checking %s on instance %s/%s", ArtifactDirectory, inst.Name(), inst.ID()))
		} else if exists {
			errs = append(errs, fmt.Sprintf("Directory %s already exists on instance %s/%s", ArtifactDirectory, inst.Name(), inst.ID()))
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}

func (bd *deployment) PreBackupLock() error {
	bd.Logger.Info("bbr", "Running pre-backup scripts...")
	err := bd.instances.PreBackupLock()
	bd.Logger.Info("bbr", "Done.")
	return err
}

func (bd *deployment) Backup() error {
	bd.Logger.Info("bbr", "Running backup scripts...")
	return bd.instances.AllBackupable().Backup()
}

func (bd *deployment) PostBackupUnlock() error {
	bd.Logger.Info("bbr", "Running post-backup scripts...")
	err := bd.instances.PostBackupUnlock()
	bd.Logger.Info("bbr", "Done.")
	return err
}

func (bd *deployment) Restore() error {
	bd.Logger.Info("bbr", "Running restore scripts...")
	return bd.instances.AllRestoreable().Restore()
}

func (bd *deployment) PostRestoreUnlock() error {
	return bd.instances.PostRestoreUnlock()
}

func (bd *deployment) Cleanup() error {
	return bd.instances.Cleanup()
}

func (bd *deployment) CleanupPrevious() error {
	return bd.instances.AllBackupableOrRestorable().CleanupPrevious()
}

func (bd *deployment) IsRestorable() bool {
	restoreableInstances := bd.instances.AllRestoreable()
	return !restoreableInstances.IsEmpty()
}

func (bd *deployment) CustomArtifactNamesMatch() error {
	for _, instance := range bd.Instances() {
		jobName := instance.Name()
		for _, restoreName := range instance.CustomRestoreArtifactNames() {
			var found bool
			for _, backupName := range bd.instances.CustomArtifactNames() {
				if restoreName == backupName {
					found = true
				}
			}
			if !found {
				return errors.New(
					fmt.Sprintf(
						"The %s restore script expects a backup script which produces %s artifact which is not present in the deployment.",
						jobName,
						restoreName,
					),
				)
			}
		}
	}
	return nil
}

func (bd *deployment) CopyRemoteBackupToLocal(backup Backup) error {
	instances := bd.instances.AllBackupable()
	for _, instance := range instances {
		for _, backupArtifact := range instance.ArtifactsToBackup() {
			writer, err := backup.CreateArtifact(backupArtifact)

			if err != nil {
				return err
			}

			size, err := backupArtifact.Size()
			if err != nil {
				return err
			}

			bd.Logger.Info("bbr", "Copying backup -- %s uncompressed -- from %s/%s...", size, instance.Name(), instance.ID())
			if err := backupArtifact.StreamFromRemote(writer); err != nil {
				return err
			}

			if err := writer.Close(); err != nil {
				return err
			}
			bd.Logger.Info("bbr", "Finished copying backup -- from %s/%s...", instance.Name(), instance.ID())

			bd.Logger.Info("bbr", "Starting validity checks")
			localChecksum, err := backup.CalculateChecksum(backupArtifact)
			if err != nil {
				return err
			}

			remoteChecksum, err := backupArtifact.Checksum()
			if err != nil {
				return err
			}
			bd.Logger.Debug("bbr", "Comparing shasums")
			if !localChecksum.Match(remoteChecksum) {
				return errors.Errorf("Backup is corrupted, checksum failed for %s/%s %s,  remote file: %s, local file: %s", instance.Name(), instance.ID(), backupArtifact.Name(), remoteChecksum, localChecksum)
			}

			backup.AddChecksum(backupArtifact, localChecksum)

			err = backupArtifact.Delete()
			if err != nil {
				return err
			}
			bd.Logger.Info("bbr", "Finished validity checks")
		}
	}
	return nil
}

func (bd *deployment) CopyLocalBackupToRemote(backup Backup) error {
	instances := bd.instances.AllRestoreable()

	for _, instance := range instances {
		for _, artifact := range instance.ArtifactsToRestore() {
			reader, err := backup.ReadArtifact(artifact)

			if err != nil {
				return err
			}

			bd.Logger.Info("bbr", "Copying backup to %s/%s...", instance.Name(), instance.Index())
			if err := artifact.StreamToRemote(reader); err != nil {
				return err
			} else {
				instance.MarkArtifactDirCreated()
			}

			localChecksum, err := backup.FetchChecksum(artifact)
			if err != nil {
				return err
			}

			remoteChecksum, err := artifact.Checksum()
			if err != nil {
				return err
			}
			if !localChecksum.Match(remoteChecksum) {
				return errors.Errorf("Backup couldn't be transfered, checksum failed for %s/%s %s,  remote file: %s, local file: %s", instance.Name(), instance.ID(), artifact.Name(), remoteChecksum, localChecksum)
			}
			bd.Logger.Info("bbr", "Done.")
		}
	}
	return nil
}

func (bd *deployment) Instances() []Instance {
	return bd.instances
}
