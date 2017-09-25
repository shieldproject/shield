package fakes

import (
	birelsetmanifest "github.com/cloudfoundry/bosh-init/release/set/manifest"
)

type FakeParser struct {
	ParsePath     string
	ParseManifest birelsetmanifest.Manifest
	ParseErr      error
}

func NewFakeParser() *FakeParser {
	return &FakeParser{}
}

func (p *FakeParser) Parse(path string) (birelsetmanifest.Manifest, error) {
	p.ParsePath = path
	return p.ParseManifest, p.ParseErr
}
