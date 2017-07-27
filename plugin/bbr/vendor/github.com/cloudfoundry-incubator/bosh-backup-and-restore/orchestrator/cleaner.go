package orchestrator

func NewCleaner(logger Logger, deploymentManager DeploymentManager) *Cleaner {
	return &Cleaner{
		Logger:            logger,
		DeploymentManager: deploymentManager,
	}
}

type Cleaner struct {
	Logger
	DeploymentManager
}

func (c Cleaner) Cleanup(deploymentName string) Error {
	deployment, err := c.DeploymentManager.Find(deploymentName)
	if err != nil {
		return Error{err}
	}

	var currentError = Error{}

	err = deployment.PostBackupUnlock()
	if err != nil {
		currentError = append(currentError, err)
	}

	err = deployment.CleanupPrevious()
	if err != nil {
		currentError = append(currentError, err)
	}

	if len(currentError) == 0 {
		c.Logger.Info("bbr", "'%s' cleaned up\n", deploymentName)
	}
	return currentError
}
