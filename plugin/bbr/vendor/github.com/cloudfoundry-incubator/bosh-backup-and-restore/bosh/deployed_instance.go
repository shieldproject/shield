package bosh

import (
	"fmt"

	"github.com/cloudfoundry/bosh-cli/director"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"
	"github.com/pkg/errors"
)

type BoshDeployedInstance struct {
	Deployment director.Deployment
	*instance.DeployedInstance
}

func NewBoshDeployedInstance(instanceGroupName,
	instanceIndex,
	instanceID string,
	connection ssh.SSHConnection,
	deployment director.Deployment,
	artifactDirectoryCreated bool,
	logger Logger,
	jobs instance.Jobs,
) orchestrator.Instance {
	return &BoshDeployedInstance{
		Deployment:       deployment,
		DeployedInstance: instance.NewDeployedInstance(instanceIndex, instanceGroupName, instanceID, artifactDirectoryCreated, connection, logger, jobs),
	}
}

func (d *BoshDeployedInstance) Cleanup() error {
	var errs []error

	if d.ArtifactDirCreated() {
		removeArtifactError := d.removeBackupArtifacts()
		if removeArtifactError != nil {
			errs = append(errs, errors.Wrap(removeArtifactError, "failed to remove backup artifact"))
		}
	}

	d.Logger.Debug("bbr", "Cleaning up SSH connection on instance %s %s", d.Name(), d.ID())
	cleanupSSHError := d.Deployment.CleanUpSSH(director.NewAllOrInstanceGroupOrInstanceSlug(d.Name(), d.ID()), director.SSHOpts{Username: d.SSHConnection.Username()})
	if cleanupSSHError != nil {
		errs = append(errs, errors.Wrap(cleanupSSHError, "failed to cleanup ssh"))
	}

	return orchestrator.ConvertErrors(errs)
}

func (d *BoshDeployedInstance) CleanupPrevious() error {
	var errs []error

	removeArtifactError := d.removeBackupArtifacts()
	if removeArtifactError != nil {
		errs = append(errs, errors.Wrap(removeArtifactError, "failed to remove backup artifact"))
	}

	d.Logger.Debug("bbr", "Cleaning up SSH connection on instance %s %s", d.Name(), d.ID())
	cleanupSSHError := d.Deployment.CleanUpSSH(director.NewAllOrInstanceGroupOrInstanceSlug(d.Name(), d.ID()), director.SSHOpts{Username: d.SSHConnection.Username()})
	if cleanupSSHError != nil {
		errs = append(errs, errors.Wrap(cleanupSSHError, "failed to cleanup ssh"))
	}

	return orchestrator.ConvertErrors(errs)
}

func (d *BoshDeployedInstance) removeBackupArtifacts() error {
	_, _, _, err := d.RunOnInstance(fmt.Sprintf("sudo rm -rf %s", orchestrator.ArtifactDirectory), "remove backup artifacts")
	return err
}
