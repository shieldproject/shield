package fakes

import (
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
)

type FakeDefaultNetworkResolver struct {
	GetDefaultNetworkNetwork boshsettings.Network
	GetDefaultNetworkErr     error
	GetDefaultNetworkCalled  bool
}

func (r *FakeDefaultNetworkResolver) GetDefaultNetwork() (boshsettings.Network, error) {
	r.GetDefaultNetworkCalled = true
	return r.GetDefaultNetworkNetwork, r.GetDefaultNetworkErr
}
