package instance

import (
	biblobstore "github.com/cloudfoundry/bosh-init/blobstore"
	bicloud "github.com/cloudfoundry/bosh-init/cloud"
	bisshtunnel "github.com/cloudfoundry/bosh-init/deployment/sshtunnel"
	bivm "github.com/cloudfoundry/bosh-init/deployment/vm"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type ManagerFactory interface {
	NewManager(bicloud.Cloud, bivm.Manager, biblobstore.Blobstore) Manager
}

type managerFactory struct {
	sshTunnelFactory bisshtunnel.Factory
	instanceFactory  Factory
	logger           boshlog.Logger
}

func NewManagerFactory(
	sshTunnelFactory bisshtunnel.Factory,
	instanceFactory Factory,
	logger boshlog.Logger,
) ManagerFactory {
	return &managerFactory{
		sshTunnelFactory: sshTunnelFactory,
		instanceFactory:  instanceFactory,
		logger:           logger,
	}
}

func (f *managerFactory) NewManager(cloud bicloud.Cloud, vmManager bivm.Manager, blobstore biblobstore.Blobstore) Manager {
	return NewManager(
		cloud,
		vmManager,
		blobstore,
		f.sshTunnelFactory,
		f.instanceFactory,
		f.logger,
	)
}
