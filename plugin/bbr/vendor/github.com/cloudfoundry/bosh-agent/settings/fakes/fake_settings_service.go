package fakes

import (
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
)

type FakeSettingsService struct {
	PublicKey    string
	PublicKeyErr error

	LoadSettingsError  error
	SettingsWereLoaded bool

	InvalidateSettingsError error
	SettingsWereInvalidated bool

	Settings boshsettings.Settings
}

func (service *FakeSettingsService) InvalidateSettings() error {
	service.SettingsWereInvalidated = true
	return service.InvalidateSettingsError
}

func (service *FakeSettingsService) PublicSSHKeyForUsername(_ string) (string, error) {
	return service.PublicKey, service.PublicKeyErr
}

func (service *FakeSettingsService) LoadSettings() error {
	service.SettingsWereLoaded = true
	return service.LoadSettingsError
}

func (service FakeSettingsService) GetSettings() boshsettings.Settings {
	return service.Settings
}
