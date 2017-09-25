package director

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"gopkg.in/yaml.v2"
)

type RuntimeConfigManifest struct {
	Releases []RuntimeConfigManifestRelease
}

type RuntimeConfigManifestRelease struct {
	Name    string
	Version string

	URL  string
	SHA1 string
}

func NewRuntimeConfigManifestFromBytes(bytes []byte) (RuntimeConfigManifest, error) {
	var rc RuntimeConfigManifest

	err := yaml.Unmarshal(bytes, &rc)
	if err != nil {
		return rc, bosherr.WrapError(err, "Unmarshalling runtime config")
	}

	return rc, nil
}
