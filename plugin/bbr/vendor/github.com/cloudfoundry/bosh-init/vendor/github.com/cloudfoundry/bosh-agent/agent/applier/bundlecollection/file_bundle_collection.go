package bundlecollection

import (
	"path"
	"path/filepath"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

const fileBundleCollectionLogTag = "FileBundleCollection"

type fileBundleDefinition struct {
	name    string
	version string
}

func newFileBundleDefinition(installPath string) fileBundleDefinition {
	cleanInstallPath := cleanPath(installPath) // no trailing slash

	// If the path is empty, Base returns ".".
	// If the path consists entirely of separators, Base returns a single separator.

	name := path.Base(path.Dir(cleanInstallPath))
	if name == "." || name == string("/") {
		name = ""
	}

	version := path.Base(cleanInstallPath)
	if version == "." || version == string("/") {
		version = ""
	}

	return fileBundleDefinition{name: name, version: version}
}

func (bd fileBundleDefinition) BundleName() string    { return bd.name }
func (bd fileBundleDefinition) BundleVersion() string { return bd.version }

type FileBundleCollection struct {
	name        string
	installPath string
	enablePath  string
	fs          boshsys.FileSystem
	logger      boshlog.Logger
}

func NewFileBundleCollection(
	installPath, enablePath, name string,
	fs boshsys.FileSystem,
	logger boshlog.Logger,
) FileBundleCollection {
	return FileBundleCollection{
		name:        cleanPath(name),
		installPath: cleanPath(installPath),
		enablePath:  cleanPath(enablePath),
		fs:          fs,
		logger:      logger,
	}
}

func (bc FileBundleCollection) Get(definition BundleDefinition) (Bundle, error) {
	if len(definition.BundleName()) == 0 {
		return nil, bosherr.Error("Missing bundle name")
	}

	if len(definition.BundleVersion()) == 0 {
		return nil, bosherr.Error("Missing bundle version")
	}

	installPath := path.Join(bc.installPath, bc.name, definition.BundleName(), definition.BundleVersion())
	enablePath := path.Join(bc.enablePath, bc.name, definition.BundleName())
	return NewFileBundle(installPath, enablePath, bc.fs, bc.logger), nil
}

func (bc FileBundleCollection) List() ([]Bundle, error) {
	var bundles []Bundle

	bundleInstallPaths, err := bc.fs.Glob(path.Join(bc.installPath, bc.name, "*", "*"))
	if err != nil {
		return bundles, bosherr.WrapError(err, "Globbing bundles")
	}

	for _, path := range bundleInstallPaths {
		bundle, err := bc.Get(newFileBundleDefinition(path))
		if err != nil {
			return bundles, bosherr.WrapError(err, "Getting bundle")
		}

		bundles = append(bundles, bundle)
	}

	bc.logger.Debug(fileBundleCollectionLogTag, "Collection contains bundles %v", bundles)

	return bundles, nil
}

func cleanPath(name string) string {
	return path.Clean(filepath.ToSlash(name))
}
