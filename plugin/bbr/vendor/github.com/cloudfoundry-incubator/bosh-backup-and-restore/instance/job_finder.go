package instance

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

//go:generate counterfeiter -o fakes/fake_job_finder.go . JobFinder
type JobFinder interface {
	FindJobs(hostIdentifier string, connection SSHConnection) (Jobs, error)
}

type JobFinderFromScripts struct {
	Logger Logger
}

func NewJobFinder(logger Logger) *JobFinderFromScripts {
	return &JobFinderFromScripts{
		Logger: logger,
	}
}

func (j *JobFinderFromScripts) FindJobs(hostIdentifierForLogging string, connection SSHConnection) (Jobs, error) {
	findOutput, err := j.findScripts(hostIdentifierForLogging, connection)
	if err != nil {
		return nil, err
	}
	metadata := map[string]Metadata{}
	scripts := NewBackupAndRestoreScripts(findOutput)
	for _, script := range scripts {
		j.Logger.Info("bbr", "%s/%s/%s", hostIdentifierForLogging, script.JobName(), script.Name())
	}
	for _, script := range scripts.MetadataOnly() {
		jobMetadata, err := j.findMetadata(hostIdentifierForLogging, script, connection)

		if err != nil {
			return nil, err
		}

		jobName := script.JobName()
		metadata[jobName] = *jobMetadata
	}

	return NewJobs(connection, hostIdentifierForLogging, j.Logger, scripts, metadata), nil
}

func (j *JobFinderFromScripts) findMetadata(hostIdentifier string, script Script, connection SSHConnection) (*Metadata, error) {
	metadataContent, _, _, err := connection.Run(string(script))

	if err != nil {
		errorString := fmt.Sprintf(
			"An error occurred while running job metadata scripts on %s: %s",
			hostIdentifier,
			err,
		)
		j.Logger.Error("bbr", errorString)
		return nil, errors.New(errorString)
	}

	jobMetadata, err := NewJobMetadata(metadataContent)

	if err != nil {
		errorString := fmt.Sprintf(
			"Reading job metadata for %s failed: %s",
			hostIdentifier,
			err.Error(),
		)
		j.Logger.Error("bbr", errorString)
		return nil, errors.New(errorString)
	}

	return jobMetadata, nil
}

func (j *JobFinderFromScripts) findScripts(hostIdentifierForLogging string, sshConnection SSHConnection) ([]string, error) {
	j.Logger.Debug("bbr", "Attempting to find scripts on %s", hostIdentifierForLogging)

	stdout, stderr, exitCode, err := sshConnection.Run("find /var/vcap/jobs/*/bin/bbr/* -type f")
	if err != nil {
		j.Logger.Error(
			"",
			"Failed to run find on %s. Error: %s\nStdout: %s\nStderr%s",
			hostIdentifierForLogging,
			err,
			stdout,
			stderr,
		)
		return nil, err
	}

	if exitCode != 0 {
		if strings.Contains(string(stderr), "No such file or directory") {
			j.Logger.Debug(
				"",
				"Running find failed on %s.\nStdout: %s\nStderr: %s",
				hostIdentifierForLogging,
				stdout,
				stderr,
			)
		} else {
			j.Logger.Error(
				"",
				"Running find failed on %s.\nStdout: %s\nStderr: %s",
				hostIdentifierForLogging,
				stdout,
				stderr,
			)
			return nil, errors.Errorf(
				"Running find failed on %s.\nStdout: %s\nStderr: %s",
				hostIdentifierForLogging,
				stdout,
				stderr,
			)
		}
	}
	return strings.Split(string(stdout), "\n"), nil
}
