package release

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type Extractor interface {
	Extract(releaseTarballPath string) (Release, error)
}

type extractor struct {
	fs         boshsys.FileSystem
	compressor boshcmd.Compressor
	validator  Validator
	logger     boshlog.Logger
	logTag     string
}

func NewExtractor(
	fs boshsys.FileSystem,
	compressor boshcmd.Compressor,
	validator Validator,
	logger boshlog.Logger,
) Extractor {
	return &extractor{
		fs:         fs,
		compressor: compressor,
		validator:  validator,
		logger:     logger,
		logTag:     "releaseExtractor",
	}
}

// Extract decompresses a release tarball into a temp directory (release.extractedPath),
// parses the release manifest, decompresses the packages and jobs, and validates the release.
// Use release.Delete() to clean up the temp directory.
func (e *extractor) Extract(releaseTarballPath string) (Release, error) {
	extractedReleasePath, err := e.fs.TempDir("bosh-init-release")
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Creating temp directory to extract release '%s'", releaseTarballPath)
	}

	e.logger.Info(e.logTag, "Extracting release tarball '%s' to '%s'", releaseTarballPath, extractedReleasePath)

	releaseReader := NewReader(releaseTarballPath, extractedReleasePath, e.fs, e.compressor)
	release, err := releaseReader.Read()
	if err != nil {
		if removeErr := e.fs.RemoveAll(extractedReleasePath); removeErr != nil {
			e.logger.Warn(e.logTag, "Failed to remove extracted release: %s", removeErr.Error())
		}
		return nil, bosherr.WrapErrorf(err, "Reading release from '%s'", releaseTarballPath)
	}

	err = e.validator.Validate(release)
	if err != nil {
		if removeErr := e.fs.RemoveAll(extractedReleasePath); removeErr != nil {
			e.logger.Warn(e.logTag, "Failed to remove extracted release: %s", removeErr.Error())
		}
		return nil, bosherr.WrapErrorf(err, "Validating release '%s-%s'", release.Name(), release.Version())
	}

	e.logger.Info(e.logTag, "Extracted release %s version %s", release.Name(), release.Version())

	return release, nil
}
