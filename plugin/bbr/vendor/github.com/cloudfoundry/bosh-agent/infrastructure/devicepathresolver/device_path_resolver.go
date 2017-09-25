package devicepathresolver

import (
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
)

type DevicePathResolver interface {
	GetRealDevicePath(diskSettings boshsettings.DiskSettings) (realPath string, timedOut bool, err error)
}
