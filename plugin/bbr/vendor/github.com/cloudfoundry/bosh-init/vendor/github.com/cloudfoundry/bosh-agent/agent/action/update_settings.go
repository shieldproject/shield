package action

import (
	"errors"

	"github.com/cloudfoundry/bosh-agent/platform/cert"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	"github.com/cloudfoundry/bosh-utils/logger"
)

type UpdateSettingsAction struct {
	trustedCertManager cert.Manager
	logger             logger.Logger
}

func NewUpdateSettings(trustedCertManager cert.Manager, logger logger.Logger) UpdateSettingsAction {
	return UpdateSettingsAction{
		trustedCertManager: trustedCertManager,
		logger:             logger,
	}
}

func (a UpdateSettingsAction) IsAsynchronous() bool {
	return true
}

func (a UpdateSettingsAction) IsPersistent() bool {
	return false
}

func (a UpdateSettingsAction) Run(newSettings boshsettings.Settings) (string, error) {
	a.logger.Info("update-settings-action", "Running Update Settings command")

	err := a.trustedCertManager.UpdateCertificates(newSettings.TrustedCerts)
	if err != nil {
		return "", err
	}

	return "updated", nil
}

func (a UpdateSettingsAction) Resume() (interface{}, error) {
	return nil, errors.New("not supported")
}

func (a UpdateSettingsAction) Cancel() error {
	return errors.New("not supported")
}
