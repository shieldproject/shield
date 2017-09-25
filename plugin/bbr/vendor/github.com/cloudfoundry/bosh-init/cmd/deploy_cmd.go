package cmd

import (
	"errors"
	"path/filepath"
	"strings"

	biui "github.com/cloudfoundry/bosh-init/ui"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type deployCmd struct {
	deploymentPreparerProvider func(deploymentManifestPath string) (DeploymentPreparer, error)
	ui                         biui.UI
	fs                         boshsys.FileSystem
	eventLogger                biui.Stage
	logger                     boshlog.Logger
	logTag                     string
}

func NewDeployCmd(
	ui biui.UI,
	fs boshsys.FileSystem,
	logger boshlog.Logger,
	deploymentPreparerProvider func(deploymentManifestPath string) (DeploymentPreparer, error),
) Cmd {
	return &deployCmd{
		ui: ui,
		fs: fs,
		deploymentPreparerProvider: deploymentPreparerProvider,
		logger: logger,
		logTag: "deployCmd",
	}
}

func (c *deployCmd) Name() string {
	return "deploy"
}

func (c *deployCmd) Meta() Meta {
	return Meta{
		Synopsis: "Create or update a deployment",
		Usage:    "<deployment_manifest_path>",
		Env:      genericEnv,
	}
}

func (c *deployCmd) Run(stage biui.Stage, args []string) error {
	deploymentManifestPath, err := c.parseCmdInputs(args)
	if err != nil {
		return err
	}

	manifestAbsFilePath, err := filepath.Abs(deploymentManifestPath)
	if err != nil {
		c.ui.ErrorLinef("Failed getting absolute path to deployment file '%s'", deploymentManifestPath)
		return bosherr.WrapErrorf(err, "Getting absolute path to deployment file '%s'", deploymentManifestPath)
	}

	if !c.fs.FileExists(manifestAbsFilePath) {
		c.ui.ErrorLinef("Deployment '%s' does not exist", manifestAbsFilePath)
		return bosherr.Errorf("Deployment manifest does not exist at '%s'", manifestAbsFilePath)
	}

	c.ui.PrintLinef("Deployment manifest: '%s'", manifestAbsFilePath)

	deploymentPreparer, err := c.deploymentPreparerProvider(manifestAbsFilePath)
	if err != nil {
		return err
	}

	return deploymentPreparer.PrepareDeployment(stage)
}

func (c *deployCmd) parseCmdInputs(args []string) (string, error) {
	if len(args) != 1 {
		c.logger.Error(c.logTag, "Invalid arguments: %#v", args)
		return "", errors.New("Invalid usage - deploy command requires exactly 1 argument")
	}
	return args[0], nil
}

func (c *deployCmd) isBlank(str string) bool {
	return str == "" || strings.TrimSpace(str) == ""
}
