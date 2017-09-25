package fakes

import (
	bideplmanifest "github.com/cloudfoundry/bosh-init/deployment/manifest"
)

type FakeParser struct {
	ParsePath     string
	ParseManifest bideplmanifest.Manifest
	ParseErr      error
}

func NewFakeParser() *FakeParser {
	return &FakeParser{}
}

func (p *FakeParser) Parse(path string) (bideplmanifest.Manifest, error) {
	p.ParsePath = path
	return p.ParseManifest, p.ParseErr
}
