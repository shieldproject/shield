package cmd

import (
	"strings"

	bihttpagent "github.com/cloudfoundry/bosh-agent/agentclient/http"
	biblobstore "github.com/cloudfoundry/bosh-init/blobstore"
	bicloud "github.com/cloudfoundry/bosh-init/cloud"
	biconfig "github.com/cloudfoundry/bosh-init/config"
	bicpirel "github.com/cloudfoundry/bosh-init/cpi/release"
	bidepl "github.com/cloudfoundry/bosh-init/deployment"
	bideplmanifest "github.com/cloudfoundry/bosh-init/deployment/manifest"
	bivm "github.com/cloudfoundry/bosh-init/deployment/vm"
	biinstall "github.com/cloudfoundry/bosh-init/installation"
	biinstallmanifest "github.com/cloudfoundry/bosh-init/installation/manifest"
	birel "github.com/cloudfoundry/bosh-init/release"
	birelsetmanifest "github.com/cloudfoundry/bosh-init/release/set/manifest"
	bistemcell "github.com/cloudfoundry/bosh-init/stemcell"
	biui "github.com/cloudfoundry/bosh-init/ui"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	bihttpclient "github.com/cloudfoundry/bosh-utils/httpclient"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

func NewDeploymentPreparer(
	ui biui.UI,
	logger boshlog.Logger,
	logTag string,
	deploymentStateService biconfig.DeploymentStateService,
	legacyDeploymentStateMigrator biconfig.LegacyDeploymentStateMigrator,
	releaseManager birel.Manager,
	deploymentRecord bidepl.Record,
	cloudFactory bicloud.Factory,
	stemcellManagerFactory bistemcell.ManagerFactory,
	agentClientFactory bihttpagent.AgentClientFactory,
	vmManagerFactory bivm.ManagerFactory,
	blobstoreFactory biblobstore.Factory,
	deployer bidepl.Deployer,
	deploymentManifestPath string,
	cpiInstaller bicpirel.CpiInstaller,
	releaseFetcher birel.Fetcher,
	stemcellFetcher bistemcell.Fetcher,
	releaseSetAndInstallationManifestParser ReleaseSetAndInstallationManifestParser,
	deploymentManifestParser DeploymentManifestParser,
	tempRootConfigurator TempRootConfigurator,
	targetProvider biinstall.TargetProvider,
) DeploymentPreparer {
	return DeploymentPreparer{
		ui:                                      ui,
		logger:                                  logger,
		logTag:                                  logTag,
		deploymentStateService:                  deploymentStateService,
		legacyDeploymentStateMigrator:           legacyDeploymentStateMigrator,
		releaseManager:                          releaseManager,
		deploymentRecord:                        deploymentRecord,
		cloudFactory:                            cloudFactory,
		stemcellManagerFactory:                  stemcellManagerFactory,
		agentClientFactory:                      agentClientFactory,
		vmManagerFactory:                        vmManagerFactory,
		blobstoreFactory:                        blobstoreFactory,
		deployer:                                deployer,
		deploymentManifestPath:                  deploymentManifestPath,
		cpiInstaller:                            cpiInstaller,
		releaseFetcher:                          releaseFetcher,
		stemcellFetcher:                         stemcellFetcher,
		releaseSetAndInstallationManifestParser: releaseSetAndInstallationManifestParser,
		deploymentManifestParser:                deploymentManifestParser,
		tempRootConfigurator:                    tempRootConfigurator,
		targetProvider:                          targetProvider,
	}
}

type DeploymentPreparer struct {
	ui                                      biui.UI
	logger                                  boshlog.Logger
	logTag                                  string
	deploymentStateService                  biconfig.DeploymentStateService
	legacyDeploymentStateMigrator           biconfig.LegacyDeploymentStateMigrator
	releaseManager                          birel.Manager
	deploymentRecord                        bidepl.Record
	cloudFactory                            bicloud.Factory
	stemcellManagerFactory                  bistemcell.ManagerFactory
	agentClientFactory                      bihttpagent.AgentClientFactory
	vmManagerFactory                        bivm.ManagerFactory
	blobstoreFactory                        biblobstore.Factory
	deployer                                bidepl.Deployer
	deploymentManifestPath                  string
	cpiInstaller                            bicpirel.CpiInstaller
	releaseFetcher                          birel.Fetcher
	stemcellFetcher                         bistemcell.Fetcher
	releaseSetAndInstallationManifestParser ReleaseSetAndInstallationManifestParser
	deploymentManifestParser                DeploymentManifestParser
	tempRootConfigurator                    TempRootConfigurator
	targetProvider                          biinstall.TargetProvider
}

func (c *DeploymentPreparer) PrepareDeployment(stage biui.Stage) (err error) {
	c.ui.PrintLinef("Deployment state: '%s'", c.deploymentStateService.Path())

	if !c.deploymentStateService.Exists() {
		migrated, err := c.legacyDeploymentStateMigrator.MigrateIfExists(biconfig.LegacyDeploymentStatePath(c.deploymentManifestPath))
		if err != nil {
			return bosherr.WrapError(err, "Migrating legacy deployment state file")
		}
		if migrated {
			c.ui.PrintLinef("Migrated legacy deployments file: '%s'", biconfig.LegacyDeploymentStatePath(c.deploymentManifestPath))
		}
	}

	deploymentState, err := c.deploymentStateService.Load()
	if err != nil {
		return bosherr.WrapError(err, "Loading deployment state")
	}

	target, err := c.targetProvider.NewTarget()
	if err != nil {
		return bosherr.WrapError(err, "Determining installation target")
	}

	err = c.tempRootConfigurator.PrepareAndSetTempRoot(target.TmpPath(), c.logger)
	if err != nil {
		return bosherr.WrapError(err, "Setting temp root")
	}

	defer func() {
		err := c.releaseManager.DeleteAll()
		if err != nil {
			c.logger.Warn(c.logTag, "Deleting all extracted releases: %s", err.Error())
		}
	}()

	var (
		extractedStemcell    bistemcell.ExtractedStemcell
		deploymentManifest   bideplmanifest.Manifest
		installationManifest biinstallmanifest.Manifest
	)
	err = stage.PerformComplex("validating", func(stage biui.Stage) error {
		var releaseSetManifest birelsetmanifest.Manifest
		releaseSetManifest, installationManifest, err = c.releaseSetAndInstallationManifestParser.ReleaseSetAndInstallationManifest(c.deploymentManifestPath)
		if err != nil {
			return err
		}

		for _, releaseRef := range releaseSetManifest.Releases {
			err = c.releaseFetcher.DownloadAndExtract(releaseRef, stage)
			if err != nil {
				return err
			}
		}

		err := c.cpiInstaller.ValidateCpiRelease(installationManifest, stage)
		if err != nil {
			return err
		}

		deploymentManifest, err = c.deploymentManifestParser.GetDeploymentManifest(c.deploymentManifestPath, releaseSetManifest, stage)
		if err != nil {
			return err
		}

		extractedStemcell, err = c.stemcellFetcher.GetStemcell(deploymentManifest, stage)

		nonCpiReleasesMap, _ := deploymentManifest.GetListOfTemplateReleases()
		delete(nonCpiReleasesMap, installationManifest.Template.Release) // remove CPI release from nonCpiReleasesMap

		for _, release := range c.releaseManager.List() {
			if _, ok := nonCpiReleasesMap[release.Name()]; ok {
				if release.IsCompiled() {
					compilationOsAndVersion := release.Packages()[0].Stemcell
					if strings.ToLower(compilationOsAndVersion) != strings.ToLower(extractedStemcell.OsAndVersion()) {
						return bosherr.Errorf("OS/Version mismatch between deployment stemcell and compiled package stemcell for release '%s'", release.Name())
					}
				}
			} else {
				// It is a CPI release, check if it is compiled
				if release.IsCompiled() {
					return bosherr.Errorf("CPI is not allowed to be a compiled release. The provided CPI release '%s' is compiled", release.Name())
				}
			}
		}

		return err
	})
	if err != nil {
		return err
	}
	defer func() {
		deleteErr := extractedStemcell.Delete()
		if deleteErr != nil {
			c.logger.Warn(c.logTag, "Failed to delete extracted stemcell: %s", deleteErr.Error())
		}
	}()

	isDeployed, err := c.deploymentRecord.IsDeployed(c.deploymentManifestPath, c.releaseManager.List(), extractedStemcell)
	if err != nil {
		return bosherr.WrapError(err, "Checking if deployment has changed")
	}

	if isDeployed {
		c.ui.PrintLinef("No deployment, stemcell or release changes. Skipping deploy.")
		return nil
	}

	err = c.cpiInstaller.WithInstalledCpiRelease(installationManifest, target, stage, func(installation biinstall.Installation) error {
		return installation.WithRunningRegistry(c.logger, stage, func() error {
			return c.deploy(
				installation,
				deploymentState,
				extractedStemcell,
				installationManifest,
				deploymentManifest,
				stage)
		})
	})

	return err

}

func (c *DeploymentPreparer) deploy(
	installation biinstall.Installation,
	deploymentState biconfig.DeploymentState,
	extractedStemcell bistemcell.ExtractedStemcell,
	installationManifest biinstallmanifest.Manifest,
	deploymentManifest bideplmanifest.Manifest,
	stage biui.Stage,
) (err error) {
	cloud, err := c.cloudFactory.NewCloud(installation, deploymentState.DirectorID)
	if err != nil {
		return bosherr.WrapError(err, "Creating CPI client from CPI installation")
	}

	stemcellManager := c.stemcellManagerFactory.NewManager(cloud)

	cloudStemcell, err := stemcellManager.Upload(extractedStemcell, stage)
	if err != nil {
		return err
	}

	agentClient := c.agentClientFactory.NewAgentClient(deploymentState.DirectorID, installationManifest.Mbus)
	vmManager := c.vmManagerFactory.NewManager(cloud, agentClient)

	blobstore, err := c.blobstoreFactory.Create(installationManifest.Mbus, bihttpclient.CreateDefaultClientInsecureSkipVerify())
	if err != nil {
		return bosherr.WrapError(err, "Creating blobstore client")
	}

	err = stage.PerformComplex("deploying", func(deployStage biui.Stage) error {
		err = c.deploymentRecord.Clear()
		if err != nil {
			return bosherr.WrapError(err, "Clearing deployment record")
		}

		_, err = c.deployer.Deploy(
			cloud,
			deploymentManifest,
			cloudStemcell,
			installationManifest.Registry,
			vmManager,
			blobstore,
			deployStage,
		)
		if err != nil {
			return bosherr.WrapError(err, "Deploying")
		}

		err = c.deploymentRecord.Update(c.deploymentManifestPath, c.releaseManager.List())
		if err != nil {
			return bosherr.WrapError(err, "Updating deployment record")
		}

		return nil
	})
	if err != nil {
		return err
	}

	// TODO: cleanup unused disks here?

	err = stemcellManager.DeleteUnused(stage)
	if err != nil {
		return err
	}

	return nil
}
