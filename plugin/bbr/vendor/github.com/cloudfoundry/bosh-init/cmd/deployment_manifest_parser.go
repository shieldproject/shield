package cmd

import (
	bideplmanifest "github.com/cloudfoundry/bosh-init/deployment/manifest"
	birel "github.com/cloudfoundry/bosh-init/release"
	birelsetmanifest "github.com/cloudfoundry/bosh-init/release/set/manifest"
	biui "github.com/cloudfoundry/bosh-init/ui"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type DeploymentManifestParser struct {
	DeploymentParser    bideplmanifest.Parser
	DeploymentValidator bideplmanifest.Validator
	ReleaseManager      birel.Manager
}

func (y DeploymentManifestParser) GetDeploymentManifest(deploymentManifestPath string, releaseSetManifest birelsetmanifest.Manifest, stage biui.Stage) (bideplmanifest.Manifest, error) {
	var deploymentManifest bideplmanifest.Manifest
	err := stage.Perform("Validating deployment manifest", func() error {
		var err error
		deploymentManifest, err = y.DeploymentParser.Parse(deploymentManifestPath)
		if err != nil {
			return bosherr.WrapErrorf(err, "Parsing deployment manifest '%s'", deploymentManifestPath)
		}

		err = y.DeploymentValidator.Validate(deploymentManifest, releaseSetManifest)
		if err != nil {
			return bosherr.WrapError(err, "Validating deployment manifest")
		}

		err = y.DeploymentValidator.ValidateReleaseJobs(deploymentManifest, y.ReleaseManager)
		if err != nil {
			return bosherr.WrapError(err, "Validating deployment jobs refer to jobs in release")
		}

		return nil
	})
	if err != nil {
		return bideplmanifest.Manifest{}, err
	}

	return deploymentManifest, nil
}
