package fakes

import boshvitals "github.com/cloudfoundry/bosh-agent/platform/vitals"

type FakeService struct {
	GetVitals boshvitals.Vitals
	GetErr    error
}

func NewFakeService() (fakeService *FakeService) {
	fakeService = new(FakeService)
	return
}

func (s *FakeService) Get() (vitals boshvitals.Vitals, err error) {
	vitals = s.GetVitals
	err = s.GetErr
	return
}
