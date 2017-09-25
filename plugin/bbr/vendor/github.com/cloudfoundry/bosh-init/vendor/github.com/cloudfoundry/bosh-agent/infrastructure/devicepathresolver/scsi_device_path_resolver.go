package devicepathresolver

import (
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type scsiDevicePathResolver struct {
	scsiVolumeIDPathResolver DevicePathResolver
	scsiIDPathResolver       DevicePathResolver
	scsiLunPathResolver      DevicePathResolver
}

func NewScsiDevicePathResolver(
	scsiVolumeIDPathResolver DevicePathResolver,
	scsiIDPathResolver DevicePathResolver,
	scsiLunPathResolver DevicePathResolver,
) DevicePathResolver {
	return scsiDevicePathResolver{
		scsiVolumeIDPathResolver: scsiVolumeIDPathResolver,
		scsiIDPathResolver:       scsiIDPathResolver,
		scsiLunPathResolver:      scsiLunPathResolver,
	}
}

func (sr scsiDevicePathResolver) GetRealDevicePath(diskSettings boshsettings.DiskSettings) (string, bool, error) {
	if len(diskSettings.DeviceID) > 0 {
		return sr.scsiIDPathResolver.GetRealDevicePath(diskSettings)
	}

	if len(diskSettings.VolumeID) > 0 {
		return sr.scsiVolumeIDPathResolver.GetRealDevicePath(diskSettings)
	}

	if len(diskSettings.Lun) > 0 && len(diskSettings.HostDeviceID) > 0 {
		return sr.scsiLunPathResolver.GetRealDevicePath(diskSettings)
	}

	return "", false, bosherr.Error("Neither ID, VolumeID nor (Lun, HostDeviceID) provided in disk settings")
}
