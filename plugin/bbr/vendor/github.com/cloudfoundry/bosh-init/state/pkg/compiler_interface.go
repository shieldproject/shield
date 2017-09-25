package pkg

import (
	birelpkg "github.com/cloudfoundry/bosh-init/release/pkg"
)

type Compiler interface {
	Compile(*birelpkg.Package) (CompiledPackageRecord, bool, error)
}
