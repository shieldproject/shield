package fakes

import (
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
)

type FakeManager struct {
	FakeDefaultNetworkResolver

	SetupNetworkingNetworks boshsettings.Networks
	SetupNetworkingErr      error

	SetupIPv6Config boshsettings.IPv6
	SetupIPv6StopCh <-chan struct{}
	SetupIPv6Err    error

	GetConfiguredNetworkInterfacesInterfaces []string
	GetConfiguredNetworkInterfacesErr        error

	SetupDhcpNetworks boshsettings.Networks
	SetupDhcpErr      error
}

func (net *FakeManager) SetupIPv6(config boshsettings.IPv6, stopCh <-chan struct{}) error {
	net.SetupIPv6Config = config
	net.SetupIPv6StopCh = stopCh
	return net.SetupIPv6Err
}

func (net *FakeManager) SetupNetworking(networks boshsettings.Networks, errCh chan error) error {
	net.SetupNetworkingNetworks = networks
	return net.SetupNetworkingErr
}

func (net *FakeManager) GetConfiguredNetworkInterfaces() ([]string, error) {
	return net.GetConfiguredNetworkInterfacesInterfaces, net.GetConfiguredNetworkInterfacesErr
}

func (net *FakeManager) SetupDhcp(networks boshsettings.Networks, errCh chan error) error {
	net.SetupDhcpNetworks = networks
	return net.SetupDhcpErr
}
