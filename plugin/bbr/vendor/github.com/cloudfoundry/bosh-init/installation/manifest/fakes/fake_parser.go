package fakes

import (
	biinstallmanifest "github.com/cloudfoundry/bosh-init/installation/manifest"
	birelsetmanifest "github.com/cloudfoundry/bosh-init/release/set/manifest"
)

type FakeParser struct {
	ParsePath          string
	ReleaseSetManifest birelsetmanifest.Manifest
	ParseManifest      biinstallmanifest.Manifest
	ParseErr           error
}

func NewFakeParser() *FakeParser {
	return &FakeParser{}
}

func (p *FakeParser) Parse(path string, releaseSetManifest birelsetmanifest.Manifest) (biinstallmanifest.Manifest, error) {
	p.ParsePath = path
	p.ReleaseSetManifest = releaseSetManifest
	return p.ParseManifest, p.ParseErr
}
