package action

import (
	"errors"

	"encoding/json"
	"github.com/cloudfoundry/bosh-agent/platform"
	"github.com/cloudfoundry/bosh-agent/platform/cert"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"github.com/cloudfoundry/bosh-utils/logger"
	"path/filepath"
)

type UpdateSettingsAction struct {
	trustedCertManager cert.Manager
	logger             logger.Logger
	settingsService    boshsettings.Service
	platform           platform.Platform
}

func NewUpdateSettings(service boshsettings.Service, platform platform.Platform, trustedCertManager cert.Manager, logger logger.Logger) UpdateSettingsAction {
	return UpdateSettingsAction{
		trustedCertManager: trustedCertManager,
		logger:             logger,
		settingsService:    service,
		platform:           platform,
	}
}

func (a UpdateSettingsAction) IsAsynchronous(_ ProtocolVersion) bool {
	return true
}

func (a UpdateSettingsAction) IsPersistent() bool {
	return false
}

func (a UpdateSettingsAction) IsLoggable() bool {
	return true
}

func (a UpdateSettingsAction) Run(newUpdateSettings boshsettings.UpdateSettings) (string, error) {
	err := a.settingsService.LoadSettings()
	if err != nil {
		return "", err
	}

	currentSettings := a.settingsService.GetSettings()

	for _, diskAssociation := range newUpdateSettings.DiskAssociations {
		diskSettings, found := currentSettings.PersistentDiskSettings(diskAssociation.DiskCID)
		if !found {
			return "", bosherr.Errorf("Persistent disk settings contains no disk with CID: %s", diskAssociation.DiskCID)
		}

		err := a.platform.AssociateDisk(diskAssociation.Name, diskSettings)
		if err != nil {
			return "", err
		}
	}

	err = a.trustedCertManager.UpdateCertificates(newUpdateSettings.TrustedCerts)
	if err != nil {
		return "", err
	}

	updateSettingsJSON, err := json.Marshal(newUpdateSettings)
	if err != nil {
		return "", bosherr.WrapError(err, "Marshalling updateSettings json")
	}

	updateSettingsPath := filepath.Join(a.platform.GetDirProvider().BoshDir(), "update_settings.json")
	err = a.platform.GetFs().WriteFile(updateSettingsPath, updateSettingsJSON)
	if err != nil {
		return "", bosherr.WrapError(err, "writing update settings json")
	}

	return "updated", nil
}

func (a UpdateSettingsAction) Resume() (interface{}, error) {
	return nil, errors.New("not supported")
}

func (a UpdateSettingsAction) Cancel() error {
	return errors.New("not supported")
}
