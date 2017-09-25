package stemcell

import (
	bicloud "github.com/cloudfoundry/bosh-init/cloud"
	biconfig "github.com/cloudfoundry/bosh-init/config"
)

type ManagerFactory interface {
	NewManager(bicloud.Cloud) Manager
}

type managerFactory struct {
	repo biconfig.StemcellRepo
}

func NewManagerFactory(repo biconfig.StemcellRepo) ManagerFactory {
	return &managerFactory{
		repo: repo,
	}
}

func (f *managerFactory) NewManager(cloud bicloud.Cloud) Manager {
	return NewManager(f.repo, cloud)
}
