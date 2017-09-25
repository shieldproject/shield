package release

import (
	"fmt"
	"github.com/cloudfoundry/bosh-init/installation/tarball"
	"github.com/cloudfoundry/bosh-init/release/manifest"
	"github.com/cloudfoundry/bosh-init/ui"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type Fetcher struct {
	tarballProvider  tarball.Provider
	releaseExtractor Extractor
	releaseManager   Manager
}

func NewFetcher(tarballProvider tarball.Provider, releaseExtractor Extractor, releaseManager Manager) Fetcher {
	return Fetcher{
		tarballProvider:  tarballProvider,
		releaseExtractor: releaseExtractor,
		releaseManager:   releaseManager,
	}
}

func (f Fetcher) DownloadAndExtract(releaseRef manifest.ReleaseRef, stage ui.Stage) error {
	releasePath, err := f.tarballProvider.Get(releaseRef, stage)
	if err != nil {
		return err
	}

	err = stage.Perform(fmt.Sprintf("Validating release '%s'", releaseRef.Name), func() error {
		release, err := f.releaseExtractor.Extract(releasePath)
		if err != nil {
			return bosherr.WrapErrorf(err, "Extracting release '%s'", releasePath)
		}

		if release.Name() != releaseRef.Name {
			return bosherr.Errorf("Release name '%s' does not match the name in release tarball '%s'", releaseRef.Name, release.Name())
		}
		f.releaseManager.Add(release)

		return nil
	})
	return err
}
