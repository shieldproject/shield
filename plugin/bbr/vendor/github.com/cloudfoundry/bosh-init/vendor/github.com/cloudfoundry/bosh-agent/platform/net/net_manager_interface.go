package net

import (
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
)

type Manager interface {
	// SetupNetworking configures network interfaces with either a static ip or dhcp.
	// If errCh is provided, nil or an error will be sent
	// upon completion of background network reconfiguration (e.g. arping).
	SetupNetworking(networks boshsettings.Networks, errCh chan error) error

	// Returns the list of interfaces that have configurations for them present
	GetConfiguredNetworkInterfaces() ([]string, error)
}
