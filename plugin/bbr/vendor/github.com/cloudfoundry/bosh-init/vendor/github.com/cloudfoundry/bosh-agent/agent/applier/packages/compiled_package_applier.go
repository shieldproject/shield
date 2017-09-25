package packages

import (
	bc "github.com/cloudfoundry/bosh-agent/agent/applier/bundlecollection"
	models "github.com/cloudfoundry/bosh-agent/agent/applier/models"
	boshblob "github.com/cloudfoundry/bosh-utils/blobstore"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

const logTag = "compiledPackageApplier"

type compiledPackageApplier struct {
	packagesBc bc.BundleCollection

	// KeepOnly will permanently uninstall packages when operating as owner
	packagesBcOwner bool

	blobstore  boshblob.Blobstore
	compressor boshcmd.Compressor
	fs         boshsys.FileSystem
	logger     boshlog.Logger
}

func NewCompiledPackageApplier(
	packagesBc bc.BundleCollection,
	packagesBcOwner bool,
	blobstore boshblob.Blobstore,
	compressor boshcmd.Compressor,
	fs boshsys.FileSystem,
	logger boshlog.Logger,
) Applier {
	return &compiledPackageApplier{
		packagesBc:      packagesBc,
		packagesBcOwner: packagesBcOwner,
		blobstore:       blobstore,
		compressor:      compressor,
		fs:              fs,
		logger:          logger,
	}
}

func (s compiledPackageApplier) Prepare(pkg models.Package) error {
	s.logger.Debug(logTag, "Preparing package %v", pkg)

	pkgBundle, err := s.packagesBc.Get(pkg)
	if err != nil {
		return bosherr.WrapError(err, "Getting package bundle")
	}

	pkgInstalled, err := pkgBundle.IsInstalled()
	if err != nil {
		return bosherr.WrapError(err, "Checking if package is installed")
	}

	if !pkgInstalled {
		err := s.downloadAndInstall(pkg, pkgBundle)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s compiledPackageApplier) Apply(pkg models.Package) error {
	s.logger.Debug(logTag, "Applying package %v", pkg)

	err := s.Prepare(pkg)
	if err != nil {
		return err
	}

	pkgBundle, err := s.packagesBc.Get(pkg)
	if err != nil {
		return bosherr.WrapError(err, "Getting package bundle")
	}

	_, _, err = pkgBundle.Enable()
	if err != nil {
		return bosherr.WrapError(err, "Enabling package")
	}

	return nil
}

func (s *compiledPackageApplier) downloadAndInstall(pkg models.Package, pkgBundle bc.Bundle) error {
	tmpDir, err := s.fs.TempDir("bosh-agent-applier-packages-CompiledPackageApplier-Apply")
	if err != nil {
		return bosherr.WrapError(err, "Getting temp dir")
	}

	defer func() {
		if err = s.fs.RemoveAll(tmpDir); err != nil {
			s.logger.Warn(logTag, "Failed to clean up tmpDir: %s", err.Error())
		}
	}()

	file, err := s.blobstore.Get(pkg.Source.BlobstoreID, pkg.Source.Sha1)
	if err != nil {
		return bosherr.WrapError(err, "Fetching package blob")
	}

	defer func() {
		if err = s.blobstore.CleanUp(file); err != nil {
			s.logger.Warn(logTag, "Failed to clean up blobstore blog: %s", err.Error())
		}
	}()

	err = s.compressor.DecompressFileToDir(file, tmpDir, boshcmd.CompressorOptions{})
	if err != nil {
		return bosherr.WrapError(err, "Decompressing package files")
	}

	_, _, err = pkgBundle.Install(tmpDir)
	if err != nil {
		return bosherr.WrapError(err, "Installling package directory")
	}

	return nil
}

func (s *compiledPackageApplier) KeepOnly(pkgs []models.Package) error {
	s.logger.Debug(logTag, "Keeping only packages %v", pkgs)

	installedBundles, err := s.packagesBc.List()
	if err != nil {
		return bosherr.WrapError(err, "Retrieving installed bundles")
	}

	for _, installedBundle := range installedBundles {
		var shouldKeep bool

		for _, pkg := range pkgs {
			pkgBundle, err := s.packagesBc.Get(pkg)
			if err != nil {
				return bosherr.WrapError(err, "Getting package bundle")
			}

			if pkgBundle == installedBundle {
				shouldKeep = true
				break
			}
		}

		if !shouldKeep {
			err = installedBundle.Disable()
			if err != nil {
				return bosherr.WrapError(err, "Disabling package bundle")
			}

			if s.packagesBcOwner {
				// If we uninstall the bundle first, and the disable failed (leaving the symlink),
				// then the next time bundle collection will not include bundle in its list
				// which means that symlink will never be deleted.
				err = installedBundle.Uninstall()
				if err != nil {
					return bosherr.WrapError(err, "Uninstalling package bundle")
				}
			}
		}
	}

	return nil
}
