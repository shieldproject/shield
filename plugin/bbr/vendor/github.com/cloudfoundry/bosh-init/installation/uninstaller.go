package installation

import (
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type Uninstaller interface {
	Uninstall(Target) error
}

type uninstaller struct {
	fs     boshsys.FileSystem
	logger boshlog.Logger
	logTag string
}

func NewUninstaller(fs boshsys.FileSystem, logger boshlog.Logger) Uninstaller {
	return &uninstaller{
		fs:     fs,
		logger: logger,
		logTag: "uninstaller",
	}
}

func (u *uninstaller) Uninstall(installationTarget Target) error {
	err := u.fs.RemoveAll(installationTarget.Path())
	if err != nil {
		u.logger.Error(u.logTag, "Failed to uninstall CPI from '%s': %s", installationTarget.Path(), err.Error())
		return err
	}

	u.logger.Info(u.logTag, "Successfully uninstalled CPI from '%s'", installationTarget.Path())
	return nil
}
