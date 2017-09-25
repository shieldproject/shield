package fakes

import (
	boshinf "github.com/cloudfoundry/bosh-agent/infrastructure"
)

type FakeRegistryProvider struct {
	GetRegistryRegistry boshinf.Registry
	GetRegistryErr      error
}

func (p *FakeRegistryProvider) GetRegistry() (boshinf.Registry, error) {
	return p.GetRegistryRegistry, p.GetRegistryErr
}
