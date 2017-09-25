package packages

import (
	models "github.com/cloudfoundry/bosh-agent/agent/applier/models"
)

type Applier interface {
	Prepare(pkg models.Package) error
	Apply(pkg models.Package) error
	KeepOnly(pkgs []models.Package) error
}
