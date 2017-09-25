package devicepathresolver

import (
	"fmt"
	"path"
	"time"

	boshudev "github.com/cloudfoundry/bosh-agent/platform/udevdevice"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

const defaultDevicePrefix = "virtio"

type idDevicePathResolver struct {
	diskWaitTimeout time.Duration
	devicePrefix    string
	udev            boshudev.UdevDevice
	fs              boshsys.FileSystem
}

func NewIDDevicePathResolver(
	diskWaitTimeout time.Duration,
	devicePrefix string,
	udev boshudev.UdevDevice,
	fs boshsys.FileSystem,
) DevicePathResolver {
	return idDevicePathResolver{
		diskWaitTimeout: diskWaitTimeout,
		devicePrefix:    devicePrefix,
		udev:            udev,
		fs:              fs,
	}
}

func (idpr idDevicePathResolver) GetRealDevicePath(diskSettings boshsettings.DiskSettings) (string, bool, error) {
	if diskSettings.ID == "" {
		return "", false, bosherr.Errorf("Disk ID is not set")
	}

	if len(diskSettings.ID) < 20 {
		return "", false, bosherr.Errorf("Disk ID is not the correct format")
	}

	err := idpr.udev.Trigger()
	if err != nil {
		return "", false, bosherr.WrapError(err, "Running udevadm trigger")
	}

	err = idpr.udev.Settle()
	if err != nil {
		return "", false, bosherr.WrapError(err, "Running udevadm settle")
	}

	stopAfter := time.Now().Add(idpr.diskWaitTimeout)
	found := false

	var realPath string

	diskID := diskSettings.ID[0:20]
	deviceID := fmt.Sprintf("%s-%s", defaultDevicePrefix, diskID)
	if idpr.devicePrefix != "" {
		deviceID = fmt.Sprintf("%s-%s", idpr.devicePrefix, diskID)
	}

	for !found {
		if time.Now().After(stopAfter) {
			return "", true, bosherr.Errorf("Timed out getting real device path for '%s'", diskID)
		}

		time.Sleep(100 * time.Millisecond)

		deviceIDPath := path.Join("/", "dev", "disk", "by-id", deviceID)
		realPath, err = idpr.fs.ReadLink(deviceIDPath)
		if err != nil {
			continue
		}

		if idpr.fs.FileExists(realPath) {
			found = true
		}
	}

	return realPath, false, nil
}
