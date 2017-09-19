package orchestrator

import "time"

func NewBackuper(backupManager BackupManager, logger Logger, deploymentManager DeploymentManager, nowFunc func() time.Time) *Backuper {
	return &Backuper{
		BackupManager:     backupManager,
		Logger:            logger,
		DeploymentManager: deploymentManager,
		NowFunc:           nowFunc,
	}
}

//go:generate counterfeiter -o fakes/fake_logger.go . Logger
type Logger interface {
	Debug(tag, msg string, args ...interface{})
	Info(tag, msg string, args ...interface{})
	Warn(tag, msg string, args ...interface{})
	Error(tag, msg string, args ...interface{})
}

//go:generate counterfeiter -o fakes/fake_deployment_manager.go . DeploymentManager
type DeploymentManager interface {
	Find(deploymentName string) (Deployment, error)
	SaveManifest(deploymentName string, artifact Backup) error
}

type Backuper struct {
	BackupManager
	Logger

	DeploymentManager
	NowFunc func() time.Time
}

type AuthInfo struct {
	Type   string
	UaaUrl string
}

//Backup checks if a deployment has backupable instances and backs them up.
func (b Backuper) Backup(deploymentName string) Error {
	bw := newBackupWorkflow(b, deploymentName)

	return bw.Run()
}

func (b Backuper) CanBeBackedUp(deploymentName string) (bool, Error) {
	bw := newBackupCheckWorkflow(b, deploymentName)

	err := bw.Run()
	return err == nil, err
}
