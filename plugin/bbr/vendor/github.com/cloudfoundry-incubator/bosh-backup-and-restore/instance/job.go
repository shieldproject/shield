package instance

import (
	"fmt"

	"errors"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
)

func NewJob(sshConnection SSHConnection, instanceIdentifier string, logger Logger, jobScripts BackupAndRestoreScripts, metadata Metadata) Job {
	jobName := jobScripts[0].JobName()
	return Job{
		Logger:                       logger,
		sshConnection:                sshConnection,
		instanceIdentifierForLogging: instanceIdentifier,
		name:              jobName,
		metadata:          metadata,
		backupScript:      jobScripts.BackupOnly().firstOrBlank(),
		restoreScript:     jobScripts.RestoreOnly().firstOrBlank(),
		preBackupScript:   jobScripts.PreBackupLockOnly().firstOrBlank(),
		postBackupScript:  jobScripts.PostBackupUnlockOnly().firstOrBlank(),
		postRestoreScript: jobScripts.SinglePostRestoreUnlockScript(),
	}
}

type Job struct {
	Logger                       Logger
	name                         string
	metadata                     Metadata
	backupScript                 Script
	preBackupScript              Script
	postBackupScript             Script
	restoreScript                Script
	postRestoreScript            Script
	sshConnection                SSHConnection
	instanceIdentifierForLogging string
}

func (j Job) Name() string {
	return j.name
}

func (j Job) BackupArtifactName() string {
	return j.metadata.BackupName
}

func (j Job) RestoreArtifactName() string {
	return j.metadata.RestoreName
}

func (j Job) BackupArtifactDirectory() string {
	return fmt.Sprintf("%s/%s", orchestrator.ArtifactDirectory, j.backupArtifactOrJobName())
}

func (j Job) RestoreArtifactDirectory() string {
	return fmt.Sprintf("%s/%s", orchestrator.ArtifactDirectory, j.restoreArtifactOrJobName())
}

func (j Job) RestoreScript() Script {
	return j.restoreScript
}

func (j Job) HasBackup() bool {
	return j.backupScript != ""
}

func (j Job) HasRestore() bool {
	return j.RestoreScript() != ""
}

func (j Job) HasNamedBackupArtifact() bool {
	return j.metadata.BackupName != ""
}

func (j Job) HasNamedRestoreArtifact() bool {
	return j.metadata.RestoreName != ""
}

func (j Job) Backup() error {
	if j.backupScript != "" {
		j.Logger.Debug("bbr", "> %s", j.backupScript)
		j.Logger.Info("bbr", "Backing up %s on %s...", j.name, j.instanceIdentifierForLogging)

		stdout, stderr, exitCode, err := j.runOnInstance(
			fmt.Sprintf(
				"sudo mkdir -p %s && sudo %s %s",
				j.BackupArtifactDirectory(),
				artifactDirectoryVariables(j.BackupArtifactDirectory()),
				j.backupScript,
			),
			"backup",
		)

		j.Logger.Info("bbr", "Done.")
		return j.handleErrs(j.name, "backup", err, exitCode, stdout, stderr)
	}

	return nil
}

func (j Job) PreBackupLock() error {
	if j.preBackupScript != "" {
		j.Logger.Debug("bbr", "> %s", j.preBackupScript)
		j.Logger.Info("bbr", "Locking %s on %s for backup...", j.name, j.instanceIdentifierForLogging)

		stdout, stderr, exitCode, err := j.runOnInstance(fmt.Sprintf("sudo %s", j.preBackupScript), "pre backup lock")

		j.Logger.Info("bbr", "Done.")
		return j.handleErrs(j.name, "pre backup lock", err, exitCode, stdout, stderr)
	}

	return nil
}

func (j Job) PostBackupUnlock() error {
	if j.postBackupScript != "" {
		j.Logger.Debug("bbr", "> %s", j.postBackupScript)
		j.Logger.Info("bbr", "Unlocking %s on %s...", j.name, j.instanceIdentifierForLogging)

		stdout, stderr, exitCode, err := j.runOnInstance(fmt.Sprintf("sudo %s", j.postBackupScript), "unlock")

		j.Logger.Info("bbr", "Done.")
		return j.handleErrs(j.name, "unlock", err, exitCode, stdout, stderr)
	}

	return nil
}

func (j Job) Restore() error {
	if j.restoreScript != "" {
		j.Logger.Debug("bbr", "> %s", j.restoreScript)
		j.Logger.Info("bbr", "Restoring %s on %s...", j.name, j.instanceIdentifierForLogging)

		stdout, stderr, exitCode, err := j.runOnInstance(
			fmt.Sprintf(
				"sudo %s %s",
				artifactDirectoryVariables(j.RestoreArtifactDirectory()),
				j.restoreScript,
			),
			"restore")

		j.Logger.Info("bbr", "Done.")
		return j.handleErrs(j.name, "restore", err, exitCode, stdout, stderr)
	}

	return nil
}

func (j Job) PostRestoreUnlock() error {
	if j.postRestoreScript != "" {
		j.Logger.Debug("bbr", "> %s", j.postRestoreScript)
		j.Logger.Info("bbr", "Unlocking %s on %s...", j.name, j.instanceIdentifierForLogging)

		stdout, stderr, exitCode, err := j.runOnInstance(fmt.Sprintf("sudo %s", j.postRestoreScript), "post restore unlock")

		j.Logger.Info("bbr", "Done.")
		return j.handleErrs(j.name, "post-restore-unlock", err, exitCode, stdout, stderr)
	}

	return nil
}

func (j Job) backupArtifactOrJobName() string {
	if j.HasNamedBackupArtifact() {
		return j.BackupArtifactName()
	}

	return j.name
}

func (j Job) restoreArtifactOrJobName() string {
	if j.HasNamedRestoreArtifact() {
		return j.RestoreArtifactName()
	}

	return j.name
}

func (j Job) runOnInstance(cmd, label string) ([]byte, []byte, int, error) {
	j.Logger.Debug("bbr", "Running %s on %s", label, j.instanceIdentifierForLogging)

	stdout, stderr, exitCode, err := j.sshConnection.Run(cmd)
	j.Logger.Debug("bbr", "Stdout: %s", string(stdout))
	j.Logger.Debug("bbr", "Stderr: %s", string(stderr))

	if err != nil {
		j.Logger.Debug("bbr", "Error running %s on instance %s. Exit code %j, error: %s", label, j.instanceIdentifierForLogging, exitCode, err.Error())
	}

	return stdout, stderr, exitCode, err
}

func (j Job) handleErrs(jobName, label string, err error, exitCode int, stdout, stderr []byte) error {
	var foundErrors []error

	if err != nil {
		j.Logger.Error("bbr", fmt.Sprintf(
			"Error attempting to run %s script for job %s on %s. Error: %s",
			label,
			jobName,
			j.instanceIdentifierForLogging,
			err.Error(),
		))
		foundErrors = append(foundErrors, err)
	}

	if exitCode != 0 {
		errorString := fmt.Sprintf(
			"%s script for job %s failed on %s.\nStdout: %s\nStderr: %s",
			label,
			jobName,
			j.instanceIdentifierForLogging,
			stdout,
			stderr,
		)

		foundErrors = append(foundErrors, errors.New(errorString))

		j.Logger.Error("bbr", errorString)
	}

	return orchestrator.ConvertErrors(foundErrors)
}
