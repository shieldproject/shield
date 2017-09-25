package packages

import (
	"path"

	boshbc "github.com/cloudfoundry/bosh-agent/agent/applier/bundlecollection"
	boshblob "github.com/cloudfoundry/bosh-utils/blobstore"
	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type compiledPackageApplierProvider struct {
	installPath           string
	rootEnablePath        string
	jobSpecificEnablePath string
	name                  string

	blobstore  boshblob.Blobstore
	compressor boshcmd.Compressor
	fs         boshsys.FileSystem
	logger     boshlog.Logger
}

func NewCompiledPackageApplierProvider(
	installPath, rootEnablePath, jobSpecificEnablePath, name string,
	blobstore boshblob.Blobstore,
	compressor boshcmd.Compressor,
	fs boshsys.FileSystem,
	logger boshlog.Logger,
) ApplierProvider {
	return compiledPackageApplierProvider{
		installPath:           installPath,
		rootEnablePath:        rootEnablePath,
		jobSpecificEnablePath: jobSpecificEnablePath,
		name:       name,
		blobstore:  blobstore,
		compressor: compressor,
		fs:         fs,
		logger:     logger,
	}
}

// Root provides package applier that operates on system-wide packages.
// (e.g manages /var/vcap/packages/pkg-a -> /var/vcap/data/packages/pkg-a)
func (p compiledPackageApplierProvider) Root() Applier {
	return NewCompiledPackageApplier(p.RootBundleCollection(), true, p.blobstore, p.compressor, p.fs, p.logger)
}

// JobSpecific provides package applier that operates on job-specific packages.
// (e.g manages /var/vcap/jobs/job-name/packages/pkg-a -> /var/vcap/data/packages/pkg-a)
func (p compiledPackageApplierProvider) JobSpecific(jobName string) Applier {
	enablePath := path.Join(p.jobSpecificEnablePath, jobName)
	packagesBc := boshbc.NewFileBundleCollection(p.installPath, enablePath, p.name, p.fs, p.logger)
	return NewCompiledPackageApplier(packagesBc, false, p.blobstore, p.compressor, p.fs, p.logger)
}

func (p compiledPackageApplierProvider) RootBundleCollection() boshbc.BundleCollection {
	return boshbc.NewFileBundleCollection(p.installPath, p.rootEnablePath, p.name, p.fs, p.logger)
}
