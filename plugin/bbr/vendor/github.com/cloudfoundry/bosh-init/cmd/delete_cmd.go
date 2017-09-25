package cmd

import (
	"errors"
	"path/filepath"

	biui "github.com/cloudfoundry/bosh-init/ui"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type deleteCmd struct {
	deploymentDeleterProvider func(deploymentManifestString string) (DeploymentDeleter, error)
	ui                        biui.UI
	fs                        boshsys.FileSystem
	logger                    boshlog.Logger
	logTag                    string
}

func NewDeleteCmd(
	ui biui.UI,
	fs boshsys.FileSystem,
	logger boshlog.Logger,
	deploymentDeleterProvider func(deploymentManifestString string) (DeploymentDeleter, error),
) Cmd {
	return &deleteCmd{
		ui: ui,
		fs: fs,
		deploymentDeleterProvider: deploymentDeleterProvider,
		logger: logger,
		logTag: "deleteCmd",
	}
}

func (c *deleteCmd) Name() string {
	return "delete"
}

func (c *deleteCmd) Meta() Meta {
	return Meta{
		Synopsis: "Delete existing deployment",
		Usage:    "<deployment_manifest_path>",
		Env:      genericEnv,
	}
}

func (c *deleteCmd) Run(stage biui.Stage, args []string) error {
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

	deploymentDeleter, err := c.deploymentDeleterProvider(manifestAbsFilePath)
	if err != nil {
		return err
	}

	return deploymentDeleter.DeleteDeployment(stage)
}

func (c *deleteCmd) parseCmdInputs(args []string) (string, error) {
	if len(args) != 1 {
		c.logger.Error(c.logTag, "Invalid arguments: %#v", args)
		return "", errors.New("Invalid usage - delete command requires exactly 1 argument")
	}
	return args[0], nil
}
