package devicepathresolver

import (
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type identityDevicePathResolver struct{}

func NewIdentityDevicePathResolver() DevicePathResolver {
	return identityDevicePathResolver{}
}

func (r identityDevicePathResolver) GetRealDevicePath(diskSettings boshsettings.DiskSettings) (string, bool, error) {
	if len(diskSettings.Path) == 0 {
		return "", false, bosherr.Error("Getting real device path: path is missing")
	}

	return diskSettings.Path, false, nil
}
