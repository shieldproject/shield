package fakes

import (
	boshas "github.com/cloudfoundry/bosh-agent/agent/applier/applyspec"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
)

type FakeV1Service struct {
	ActionsCalled []string

	Spec   boshas.V1ApplySpec
	GetErr error
	SetErr error

	PopulateDHCPNetworksSpec       boshas.V1ApplySpec
	PopulateDHCPNetworksSettings   boshsettings.Settings
	PopulateDHCPNetworksResultSpec boshas.V1ApplySpec
	PopulateDHCPNetworksErr        error
}

func NewFakeV1Service() *FakeV1Service {
	return &FakeV1Service{}
}

func (s *FakeV1Service) Get() (boshas.V1ApplySpec, error) {
	s.ActionsCalled = append(s.ActionsCalled, "Get")
	return s.Spec, s.GetErr
}

func (s *FakeV1Service) Set(spec boshas.V1ApplySpec) error {
	s.ActionsCalled = append(s.ActionsCalled, "Set")
	s.Spec = spec
	return s.SetErr
}

func (s *FakeV1Service) PopulateDHCPNetworks(spec boshas.V1ApplySpec, settings boshsettings.Settings) (boshas.V1ApplySpec, error) {
	s.ActionsCalled = append(s.ActionsCalled, "PopulateDHCPNetworks")
	s.PopulateDHCPNetworksSpec = spec
	s.PopulateDHCPNetworksSettings = settings
	return s.PopulateDHCPNetworksResultSpec, s.PopulateDHCPNetworksErr
}
