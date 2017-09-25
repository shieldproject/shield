package fakes

import (
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
)

type FakeSettingsSource struct {
	PublicKey    string
	PublicKeyErr error

	SettingsValue boshsettings.Settings
	SettingsErr   error
}

func (s FakeSettingsSource) PublicSSHKeyForUsername(string) (string, error) {
	return s.PublicKey, s.PublicKeyErr
}

func (s FakeSettingsSource) Settings() (boshsettings.Settings, error) {
	return s.SettingsValue, s.SettingsErr
}
