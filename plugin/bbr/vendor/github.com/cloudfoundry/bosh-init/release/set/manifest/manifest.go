package manifest

import (
	birelmanifest "github.com/cloudfoundry/bosh-init/release/manifest"
)

type Manifest struct {
	Releases []birelmanifest.ReleaseRef
}

func (d Manifest) ReleasesByName() map[string]birelmanifest.ReleaseRef {
	releasesByName := map[string]birelmanifest.ReleaseRef{}
	for _, release := range d.Releases {
		releasesByName[release.Name] = release
	}
	return releasesByName
}

func (d Manifest) FindByName(name string) (birelmanifest.ReleaseRef, bool) {
	for _, release := range d.Releases {
		if release.Name == name {
			return release, true
		}
	}
	return birelmanifest.ReleaseRef{}, false
}
