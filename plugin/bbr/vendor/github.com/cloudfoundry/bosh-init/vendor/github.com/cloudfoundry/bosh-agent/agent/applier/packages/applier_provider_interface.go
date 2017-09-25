package packages

import (
	boshbc "github.com/cloudfoundry/bosh-agent/agent/applier/bundlecollection"
)

type ApplierProvider interface {
	Root() Applier
	JobSpecific(jobName string) Applier
	RootBundleCollection() boshbc.BundleCollection
}
