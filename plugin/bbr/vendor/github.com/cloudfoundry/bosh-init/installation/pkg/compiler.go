package pkg

import (
	"os"
	"path"
	"path/filepath"

	"github.com/cloudfoundry/bosh-init/installation/blobextract"
	birelpkg "github.com/cloudfoundry/bosh-init/release/pkg"
	bistatepkg "github.com/cloudfoundry/bosh-init/state/pkg"
	boshblob "github.com/cloudfoundry/bosh-utils/blobstore"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type compiler struct {
	runner              boshsys.CmdRunner
	packagesDir         string
	fileSystem          boshsys.FileSystem
	compressor          boshcmd.Compressor
	blobstore           boshblob.Blobstore
	compiledPackageRepo bistatepkg.CompiledPackageRepo
	blobExtractor       blobextract.Extractor
	logger              boshlog.Logger
	logTag              string
}

func NewPackageCompiler(
	runner boshsys.CmdRunner,
	packagesDir string,
	fileSystem boshsys.FileSystem,
	compressor boshcmd.Compressor,
	blobstore boshblob.Blobstore,
	compiledPackageRepo bistatepkg.CompiledPackageRepo,
	blobExtractor blobextract.Extractor,
	logger boshlog.Logger,
) bistatepkg.Compiler {
	return &compiler{
		runner:              runner,
		packagesDir:         packagesDir,
		fileSystem:          fileSystem,
		compressor:          compressor,
		blobstore:           blobstore,
		compiledPackageRepo: compiledPackageRepo,
		blobExtractor:       blobExtractor,
		logger:              logger,
		logTag:              "packageCompiler",
	}
}

func (c *compiler) Compile(pkg *birelpkg.Package) (bistatepkg.CompiledPackageRecord, bool, error) {

	// This is a variable being used now to fulfill the requirement of the compiler_interface compile method
	// to indicate whether the package is already compiled. Compiled CPI releases are not currently allowed.
	// No other packages, but CPI ones, are currently being compiled locally.
	isCompiledPackage := false

	c.logger.Debug(c.logTag, "Checking for compiled package '%s/%s'", pkg.Name, pkg.Fingerprint)
	record, found, err := c.compiledPackageRepo.Find(*pkg)
	if err != nil {
		return record, isCompiledPackage, bosherr.WrapErrorf(err, "Attempting to find compiled package '%s'", pkg.Name)
	}
	if found {
		return record, isCompiledPackage, nil
	}

	c.logger.Debug(c.logTag, "Installing dependencies of package '%s/%s'", pkg.Name, pkg.Fingerprint)
	err = c.installPackages(pkg.Dependencies)
	if err != nil {
		return record, isCompiledPackage, bosherr.WrapErrorf(err, "Installing dependencies of package '%s'", pkg.Name)
	}
	defer func() {
		if err = c.fileSystem.RemoveAll(c.packagesDir); err != nil {
			c.logger.Warn(c.logTag, "Failed to remove packages dir: %s", err.Error())
		}
	}()

	c.logger.Debug(c.logTag, "Compiling package '%s/%s'", pkg.Name, pkg.Fingerprint)
	installDir := path.Join(c.packagesDir, pkg.Name)
	err = c.fileSystem.MkdirAll(installDir, os.ModePerm)
	if err != nil {
		return record, isCompiledPackage, bosherr.WrapError(err, "Creating package install dir")
	}

	packageSrcDir := pkg.ExtractedPath
	if !c.fileSystem.FileExists(path.Join(packageSrcDir, "packaging")) {
		return record, isCompiledPackage, bosherr.Errorf("Packaging script for package '%s' not found", pkg.Name)
	}

	cmd := boshsys.Command{
		Name: "bash",
		Args: []string{"-x", "packaging"},
		Env: map[string]string{
			"BOSH_COMPILE_TARGET": packageSrcDir,
			"BOSH_INSTALL_TARGET": installDir,
			"BOSH_PACKAGE_NAME":   pkg.Name,
			"BOSH_PACKAGES_DIR":   c.packagesDir,
			"PATH":                "/usr/local/bin:/usr/bin:/bin",
		},
		UseIsolatedEnv: true,
		WorkingDir:     packageSrcDir,
	}

	_, _, _, err = c.runner.RunComplexCommand(cmd)
	if err != nil {
		return record, isCompiledPackage, bosherr.WrapError(err, "Compiling package")
	}

	tarball, err := c.compressor.CompressFilesInDir(installDir)
	if err != nil {
		return record, isCompiledPackage, bosherr.WrapError(err, "Compressing compiled package")
	}
	defer func() {
		if err = c.compressor.CleanUp(tarball); err != nil {
			c.logger.Warn(c.logTag, "Failed to clean up tarball: %s", err.Error())
		}
	}()

	blobID, blobSHA1, err := c.blobstore.Create(tarball)
	if err != nil {
		return record, isCompiledPackage, bosherr.WrapError(err, "Creating blob")
	}

	record = bistatepkg.CompiledPackageRecord{
		BlobID:   blobID,
		BlobSHA1: blobSHA1,
	}
	err = c.compiledPackageRepo.Save(*pkg, record)
	if err != nil {
		return record, isCompiledPackage, bosherr.WrapError(err, "Saving compiled package")
	}

	return record, isCompiledPackage, nil
}

func (c *compiler) installPackages(packages []*birelpkg.Package) error {
	for _, pkg := range packages {
		c.logger.Debug(c.logTag, "Checking for compiled package '%s/%s'", pkg.Name, pkg.Fingerprint)
		record, found, err := c.compiledPackageRepo.Find(*pkg)
		if err != nil {
			return bosherr.WrapErrorf(err, "Attempting to find compiled package '%s'", pkg.Name)
		}
		if !found {
			return bosherr.Errorf("Finding compiled package '%s'", pkg.Name)
		}

		c.logger.Debug(c.logTag, "Installing package '%s/%s'", pkg.Name, pkg.Fingerprint)

		err = c.blobExtractor.Extract(record.BlobID, record.BlobSHA1, filepath.Join(c.packagesDir, pkg.Name))
		if err != nil {
			return bosherr.WrapErrorf(err, "Installing package '%s' into '%s'", pkg.Name, c.packagesDir)
		}
	}

	return nil
}
