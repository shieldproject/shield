package standalone

import (
	"fmt"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	"github.com/pkg/errors"

	"io/ioutil"

	gossh "golang.org/x/crypto/ssh"
)

type DeploymentManager struct {
	orchestrator.Logger
	hostName          string
	username          string
	privateKeyFile    string
	jobFinder         instance.JobFinder
	connectionFactory ssh.SSHConnectionFactory
}

func NewDeploymentManager(
	logger orchestrator.Logger,
	hostName, username, privateKey string,
	jobFinder instance.JobFinder,
	connectionFactory ssh.SSHConnectionFactory,
) DeploymentManager {
	return DeploymentManager{
		Logger:            logger,
		hostName:          hostName,
		username:          username,
		privateKeyFile:    privateKey,
		jobFinder:         jobFinder,
		connectionFactory: connectionFactory,
	}

}

func (dm DeploymentManager) Find(deploymentName string) (orchestrator.Deployment, error) {
	keyContents, err := ioutil.ReadFile(dm.privateKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed reading private key")
	}

	connection, err := dm.connectionFactory(dm.hostName, dm.username, string(keyContents), gossh.InsecureIgnoreHostKey(), nil, dm.Logger)
	if err != nil {
		return nil, err
	}

	//TODO: change hostIdentifier, its not always bosh
	jobs, err := dm.jobFinder.FindJobs("bosh/0", connection)
	if err != nil {
		return nil, err
	}

	return orchestrator.NewDeployment(dm.Logger, []orchestrator.Instance{
		NewDeployedInstance("bosh", connection, dm.Logger, jobs, false),
	}), nil
}

func (DeploymentManager) SaveManifest(deploymentName string, artifact orchestrator.Backup) error {
	return nil
}

type DeployedInstance struct {
	*instance.DeployedInstance
}

func (d DeployedInstance) Cleanup() error {
	d.Logger.Info("", "Cleaning up...")
	if !d.ArtifactDirCreated() {
		d.Logger.Debug("", "Backup directory was never created - skipping cleanup")
		return nil
	}

	stdout, stderr, exitCode, err := d.SSHConnection.Run(fmt.Sprintf("sudo rm -rf %s", orchestrator.ArtifactDirectory))
	d.Logger.Debug("", "Stdout: %s", string(stdout))
	d.Logger.Debug("", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Error("", "Backup artifact clean up failed")
		return errors.Wrap(err, "standalone.DeployedInstance.Cleanup failed")
	}

	if exitCode != 0 {
		return errors.New("Unable to clean up backup artifact")
	}

	return nil
}

func (d DeployedInstance) CleanupPrevious() error {
	d.Logger.Info("", "Cleaning up...")

	stdout, stderr, exitCode, err := d.SSHConnection.Run(fmt.Sprintf("sudo rm -rf %s", orchestrator.ArtifactDirectory))
	d.Logger.Debug("", "Stdout: %s", string(stdout))
	d.Logger.Debug("", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Error("", "Backup artifact clean up failed")
		return errors.Wrap(err, "standalone.DeployedInstance.Cleanup failed")
	}

	if exitCode != 0 {
		return errors.New("Unable to clean up backup artifact")
	}

	return nil
}

func NewDeployedInstance(instanceGroupName string, connection ssh.SSHConnection, logger instance.Logger, jobs instance.Jobs, artifactDirCreated bool) DeployedInstance {
	return DeployedInstance{
		DeployedInstance: instance.NewDeployedInstance("0", instanceGroupName, "0", artifactDirCreated, connection, logger, jobs),
	}
}
