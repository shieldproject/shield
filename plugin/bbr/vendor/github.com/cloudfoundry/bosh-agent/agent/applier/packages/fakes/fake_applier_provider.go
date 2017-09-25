package fakes

import (
	boshbc "github.com/cloudfoundry/bosh-agent/agent/applier/bundlecollection"
	boshpackages "github.com/cloudfoundry/bosh-agent/agent/applier/packages"
)

type FakeApplierProvider struct {
	RootApplier                          *FakeApplier
	JobSpecificAppliers                  map[string]*FakeApplier
	RootBundleCollectionBundleCollection boshbc.BundleCollection
}

func NewFakeApplierProvider() *FakeApplierProvider {
	return &FakeApplierProvider{
		JobSpecificAppliers: map[string]*FakeApplier{},
	}
}

func (p *FakeApplierProvider) Root() boshpackages.Applier {
	if p.RootApplier == nil {
		panic("Root package applier not found")
	}
	return p.RootApplier
}

func (p *FakeApplierProvider) JobSpecific(jobName string) boshpackages.Applier {
	applier := p.JobSpecificAppliers[jobName]
	if applier == nil {
		panic("Job specific package applier not found")
	}
	return applier
}

func (p *FakeApplierProvider) RootBundleCollection() boshbc.BundleCollection {
	return p.RootBundleCollectionBundleCollection
}
