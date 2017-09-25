package compiler

import (
	"fmt"
	"os"
	"path"

	boshbc "github.com/cloudfoundry/bosh-agent/agent/applier/bundlecollection"
	boshmodels "github.com/cloudfoundry/bosh-agent/agent/applier/models"
	"github.com/cloudfoundry/bosh-agent/agent/applier/packages"
	boshcmdrunner "github.com/cloudfoundry/bosh-agent/agent/cmdrunner"
	boshblob "github.com/cloudfoundry/bosh-utils/blobstore"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

const PackagingScriptName = "packaging"

type CompileDirProvider interface {
	CompileDir() string
}

type concreteCompiler struct {
	compressor         boshcmd.Compressor
	blobstore          boshblob.Blobstore
	fs                 boshsys.FileSystem
	runner             boshcmdrunner.CmdRunner
	compileDirProvider CompileDirProvider
	packageApplier     packages.Applier
	packagesBc         boshbc.BundleCollection
}

func NewConcreteCompiler(
	compressor boshcmd.Compressor,
	blobstore boshblob.Blobstore,
	fs boshsys.FileSystem,
	runner boshcmdrunner.CmdRunner,
	compileDirProvider CompileDirProvider,
	packageApplier packages.Applier,
	packagesBc boshbc.BundleCollection,
) Compiler {
	return concreteCompiler{
		compressor:         compressor,
		blobstore:          blobstore,
		fs:                 fs,
		runner:             runner,
		compileDirProvider: compileDirProvider,
		packageApplier:     packageApplier,
		packagesBc:         packagesBc,
	}
}

func (c concreteCompiler) Compile(pkg Package, deps []boshmodels.Package) (string, string, error) {
	err := c.packageApplier.KeepOnly([]boshmodels.Package{})
	if err != nil {
		return "", "", bosherr.WrapError(err, "Removing packages")
	}

	for _, dep := range deps {
		err := c.packageApplier.Apply(dep)
		if err != nil {
			return "", "", bosherr.WrapErrorf(err, "Installing dependent package: '%s'", dep.Name)
		}
	}

	compilePath := path.Join(c.compileDirProvider.CompileDir(), pkg.Name)
	err = c.fetchAndUncompress(pkg, compilePath)
	if err != nil {
		return "", "", bosherr.WrapErrorf(err, "Fetching package %s", pkg.Name)
	}

	defer func() {
		_ = c.fs.RemoveAll(compilePath)
	}()

	compiledPkg := boshmodels.Package{
		Name:    pkg.Name,
		Version: pkg.Version,
	}

	compiledPkgBundle, err := c.packagesBc.Get(compiledPkg)
	if err != nil {
		return "", "", bosherr.WrapError(err, "Getting bundle for new package")
	}

	_, installPath, err := compiledPkgBundle.InstallWithoutContents()
	if err != nil {
		return "", "", bosherr.WrapError(err, "Setting up new package bundle")
	}

	_, enablePath, err := compiledPkgBundle.Enable()
	if err != nil {
		return "", "", bosherr.WrapError(err, "Enabling new package bundle")
	}

	scriptPath := path.Join(compilePath, PackagingScriptName)

	if c.fs.FileExists(scriptPath) {
		if err := c.runPackagingCommand(compilePath, enablePath, pkg); err != nil {
			return "", "", bosherr.WrapError(err, "Running packaging script")
		}
	}

	tmpPackageTar, err := c.compressor.CompressFilesInDir(installPath)
	if err != nil {
		return "", "", bosherr.WrapError(err, "Compressing compiled package")
	}

	defer func() {
		_ = c.compressor.CleanUp(tmpPackageTar)
	}()

	uploadedBlobID, sha1, err := c.blobstore.Create(tmpPackageTar)
	if err != nil {
		return "", "", bosherr.WrapError(err, "Uploading compiled package")
	}

	err = compiledPkgBundle.Disable()
	if err != nil {
		return "", "", bosherr.WrapError(err, "Disabling compiled package")
	}

	err = compiledPkgBundle.Uninstall()
	if err != nil {
		return "", "", bosherr.WrapError(err, "Uninstalling compiled package")
	}

	err = c.packageApplier.KeepOnly([]boshmodels.Package{})
	if err != nil {
		return "", "", bosherr.WrapError(err, "Removing packages")
	}

	return uploadedBlobID, sha1, nil
}

func (c concreteCompiler) fetchAndUncompress(pkg Package, targetDir string) error {
	if pkg.BlobstoreID == "" {
		return bosherr.Error(fmt.Sprintf("Blobstore ID for package '%s' is empty", pkg.Name))
	}

	// Do not verify integrity of the download via SHA1
	// because Director might have stored non-matching SHA1.
	// This will be fixed in future by explicitly asking to verify SHA1
	// instead of doing that by default like all other downloads.
	// (Ruby agent mistakenly never checked SHA1.)
	depFilePath, err := c.blobstore.Get(pkg.BlobstoreID, "")
	if err != nil {
		return bosherr.WrapErrorf(err, "Fetching package blob %s", pkg.BlobstoreID)
	}

	err = c.atomicDecompress(depFilePath, targetDir)
	if err != nil {
		return bosherr.WrapErrorf(err, "Uncompressing package %s", pkg.Name)
	}

	return nil
}

func (c concreteCompiler) atomicDecompress(archivePath string, finalDir string) error {
	tmpInstallPath := finalDir + "-bosh-agent-unpack"

	{
		err := c.fs.RemoveAll(finalDir)
		if err != nil {
			return bosherr.WrapErrorf(err, "Removing install path %s", finalDir)
		}

		err = c.fs.MkdirAll(finalDir, os.FileMode(0755))
		if err != nil {
			return bosherr.WrapErrorf(err, "Creating install path %s", finalDir)
		}
	}

	{
		err := c.fs.RemoveAll(tmpInstallPath)
		if err != nil {
			return bosherr.WrapErrorf(err, "Removing temporary compile directory %s", tmpInstallPath)
		}

		err = c.fs.MkdirAll(tmpInstallPath, os.FileMode(0755))
		if err != nil {
			return bosherr.WrapErrorf(err, "Creating temporary compile directory %s", tmpInstallPath)
		}
	}

	err := c.compressor.DecompressFileToDir(archivePath, tmpInstallPath, boshcmd.CompressorOptions{})
	if err != nil {
		return bosherr.WrapErrorf(err, "Decompressing files from %s to %s", archivePath, tmpInstallPath)
	}

	err = c.fs.Rename(tmpInstallPath, finalDir)
	if err != nil {
		return bosherr.WrapErrorf(err, "Moving temporary directory %s to final destination %s", tmpInstallPath, finalDir)
	}

	return nil
}
