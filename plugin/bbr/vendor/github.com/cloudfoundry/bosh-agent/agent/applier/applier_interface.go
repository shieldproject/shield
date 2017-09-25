package applier

import (
	boshas "github.com/cloudfoundry/bosh-agent/agent/applier/applyspec"
)

type Applier interface {
	Prepare(desiredApplySpec boshas.ApplySpec) error
	ConfigureJobs(desiredApplySpec boshas.ApplySpec) error
	Apply(currentApplySpec, desiredApplySpec boshas.ApplySpec) error
}
