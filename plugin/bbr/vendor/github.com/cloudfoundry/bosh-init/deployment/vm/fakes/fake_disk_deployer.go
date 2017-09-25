package fakes

import (
	bicloud "github.com/cloudfoundry/bosh-init/cloud"
	bidisk "github.com/cloudfoundry/bosh-init/deployment/disk"
	bideplmanifest "github.com/cloudfoundry/bosh-init/deployment/manifest"
	bivm "github.com/cloudfoundry/bosh-init/deployment/vm"
	biui "github.com/cloudfoundry/bosh-init/ui"
)

type FakeDiskDeployer struct {
	DeployInputs  []DeployInput
	deployOutputs deployOutput
}

type DeployInput struct {
	DiskPool         bideplmanifest.DiskPool
	Cloud            bicloud.Cloud
	VM               bivm.VM
	EventLoggerStage biui.Stage
}

type deployOutput struct {
	disks []bidisk.Disk
	err   error
}

func NewFakeDiskDeployer() *FakeDiskDeployer {
	return &FakeDiskDeployer{
		DeployInputs: []DeployInput{},
	}
}

func (d *FakeDiskDeployer) Deploy(
	diskPool bideplmanifest.DiskPool,
	cloud bicloud.Cloud,
	vm bivm.VM,
	eventLoggerStage biui.Stage,
) ([]bidisk.Disk, error) {
	d.DeployInputs = append(d.DeployInputs, DeployInput{
		DiskPool:         diskPool,
		Cloud:            cloud,
		VM:               vm,
		EventLoggerStage: eventLoggerStage,
	})

	return d.deployOutputs.disks, d.deployOutputs.err
}

func (d *FakeDiskDeployer) SetDeployBehavior(disks []bidisk.Disk, err error) {
	d.deployOutputs = deployOutput{
		disks: disks,
		err:   err,
	}
}
