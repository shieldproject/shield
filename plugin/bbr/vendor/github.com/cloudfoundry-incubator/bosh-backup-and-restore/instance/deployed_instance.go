package instance

import (
	"fmt"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	"github.com/pkg/errors"
)

type DeployedInstance struct {
	backupAndRestoreInstanceIndex string
	instanceID                    string
	instanceGroupName             string
	artifactDirCreated            bool
	ssh.SSHConnection
	Logger
	Jobs
}

func NewDeployedInstance(instanceIndex string, instanceGroupName string, instanceID string, artifactDirCreated bool, connection ssh.SSHConnection, logger Logger, jobs Jobs) *DeployedInstance {
	deployedInstance := &DeployedInstance{
		backupAndRestoreInstanceIndex: instanceIndex,
		instanceGroupName:             instanceGroupName,
		instanceID:                    instanceID,
		artifactDirCreated:            artifactDirCreated,
		SSHConnection:                 connection,
		Logger:                        logger,
		Jobs:                          jobs,
	}
	return deployedInstance
}

func (d *DeployedInstance) ArtifactDirExists() (bool, error) {
	_, _, exitCode, err := d.RunOnInstance(
		fmt.Sprintf(
			"stat %s",
			orchestrator.ArtifactDirectory,
		),
		"artifact directory check",
	)

	return exitCode == 0, err
}

func (d *DeployedInstance) IsBackupable() bool {
	return d.Jobs.AnyAreBackupable()
}

func (d *DeployedInstance) ArtifactDirCreated() bool {
	return d.artifactDirCreated
}

func (d *DeployedInstance) MarkArtifactDirCreated() {
	d.artifactDirCreated = true
}

func (d *DeployedInstance) CustomBackupArtifactNames() []string {
	return d.Jobs.CustomBackupArtifactNames()
}

func (d *DeployedInstance) CustomRestoreArtifactNames() []string {
	return d.Jobs.CustomRestoreArtifactNames()
}

func (d *DeployedInstance) PreBackupLock() error {
	var preBackupLockErrors []error
	for _, job := range d.Jobs {
		if err := job.PreBackupLock(); err != nil {
			preBackupLockErrors = append(preBackupLockErrors, err)
		}
	}

	return orchestrator.ConvertErrors(preBackupLockErrors)
}

func (d *DeployedInstance) Backup() error {
	var backupErrors []error
	for _, job := range d.Jobs {
		if err := job.Backup(); err != nil {
			backupErrors = append(backupErrors, err)
		}
	}

	if d.IsBackupable() {
		d.artifactDirCreated = true
	}

	return orchestrator.ConvertErrors(backupErrors)
}

func artifactDirectoryVariables(artifactDirectory string) string {
	return fmt.Sprintf("BBR_ARTIFACT_DIRECTORY=%s/ ARTIFACT_DIRECTORY=%[1]s/", artifactDirectory)
}

func (d *DeployedInstance) PostBackupUnlock() error {
	var unlockErrors []error
	for _, job := range d.Jobs {
		if err := job.PostBackupUnlock(); err != nil {
			unlockErrors = append(unlockErrors, err)
		}
	}

	return orchestrator.ConvertErrors(unlockErrors)
}

func (d *DeployedInstance) Restore() error {
	var restoreErrors []error
	for _, job := range d.Jobs {
		if err := job.Restore(); err != nil {
			restoreErrors = append(restoreErrors, err)
		}
	}

	return orchestrator.ConvertErrors(restoreErrors)
}

func (d *DeployedInstance) PostRestoreUnlock() error {
	var unlockErrors []error
	for _, job := range d.Jobs {
		if err := job.PostRestoreUnlock(); err != nil {
			unlockErrors = append(unlockErrors, err)
		}
	}

	return orchestrator.ConvertErrors(unlockErrors)
}

func (d *DeployedInstance) IsRestorable() bool {
	return d.Jobs.AnyAreRestorable()
}

func (d *DeployedInstance) ArtifactsToBackup() []orchestrator.BackupArtifact {
	artifacts := []orchestrator.BackupArtifact{}

	for _, job := range d.Jobs {
		artifacts = append(artifacts, NewBackupArtifact(job, d, d.SSHConnection, d.Logger))
	}

	return artifacts
}

func (d *DeployedInstance) ArtifactsToRestore() []orchestrator.BackupArtifact {
	artifacts := []orchestrator.BackupArtifact{}

	for _, job := range d.Jobs {
		artifacts = append(artifacts, NewRestoreArtifact(job, d, d.SSHConnection, d.Logger))
	}

	return artifacts
}

func (d *DeployedInstance) RunOnInstance(cmd, label string) ([]byte, []byte, int, error) {
	d.Logger.Debug("bbr", "Running %s on %s/%s", label, d.instanceGroupName, d.instanceID)

	stdout, stderr, exitCode, err := d.Run(cmd)
	d.Logger.Debug("bbr", "Stdout: %s", string(stdout))
	d.Logger.Debug("bbr", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Debug("bbr", "Error running %s on instance %s/%s. Exit code %d, error: %s", label, d.instanceGroupName, d.instanceID, exitCode, err.Error())
	}

	return stdout, stderr, exitCode, err
}

func (d *DeployedInstance) Name() string {
	return d.instanceGroupName
}

func (d *DeployedInstance) Index() string {
	return d.backupAndRestoreInstanceIndex
}

func (d *DeployedInstance) ID() string {
	return d.instanceID
}

func (d *DeployedInstance) handleErrs(jobName, label string, err error, exitCode int, stdout, stderr []byte) error {
	var foundErrors []error

	if err != nil {
		d.Logger.Error("bbr", fmt.Sprintf(
			"Error attempting to run %s script for job %s on %s/%s. Error: %s",
			label,
			jobName,
			d.instanceGroupName,
			d.instanceID,
			err.Error(),
		))
		foundErrors = append(foundErrors, err)
	}

	if exitCode != 0 {
		errorString := fmt.Sprintf(
			"%s script for job %s failed on %s/%s.\nStdout: %s\nStderr: %s",
			label,
			jobName,
			d.instanceGroupName,
			d.instanceID,
			stdout,
			stderr,
		)

		foundErrors = append(foundErrors, errors.New(errorString))

		d.Logger.Error("bbr", errorString)
	}

	return orchestrator.ConvertErrors(foundErrors)
}
