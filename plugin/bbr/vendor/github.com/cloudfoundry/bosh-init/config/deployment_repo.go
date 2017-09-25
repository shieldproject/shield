package config

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type DeploymentRepo interface {
	UpdateCurrent(manifestSHA1 string) error
	FindCurrent() (manifestSHA1 string, found bool, err error)
}

type deploymentRepo struct {
	deploymentStateService DeploymentStateService
}

func NewDeploymentRepo(deploymentStateService DeploymentStateService) DeploymentRepo {
	return deploymentRepo{
		deploymentStateService: deploymentStateService,
	}
}

func (r deploymentRepo) FindCurrent() (string, bool, error) {
	deploymentState, err := r.deploymentStateService.Load()
	if err != nil {
		return "", false, bosherr.WrapError(err, "Loading existing config")
	}

	currentManifestSHA1 := deploymentState.CurrentManifestSHA1
	if currentManifestSHA1 != "" {
		return currentManifestSHA1, true, nil
	}

	return "", false, nil
}

func (r deploymentRepo) UpdateCurrent(manifestSHA1 string) error {
	deploymentState, err := r.deploymentStateService.Load()
	if err != nil {
		return bosherr.WrapError(err, "Loading existing config")
	}

	deploymentState.CurrentManifestSHA1 = manifestSHA1

	err = r.deploymentStateService.Save(deploymentState)
	if err != nil {
		return bosherr.WrapError(err, "Saving new config")
	}
	return nil
}
