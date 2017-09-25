package devicepathresolver

import (
	"strings"
	"time"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type mappedDevicePathResolver struct {
	diskWaitTimeout time.Duration
	fs              boshsys.FileSystem
}

func NewMappedDevicePathResolver(
	diskWaitTimeout time.Duration,
	fs boshsys.FileSystem,
) DevicePathResolver {
	return mappedDevicePathResolver{fs: fs, diskWaitTimeout: diskWaitTimeout}
}

func (dpr mappedDevicePathResolver) GetRealDevicePath(diskSettings boshsettings.DiskSettings) (string, bool, error) {
	stopAfter := time.Now().Add(dpr.diskWaitTimeout)

	devicePath := diskSettings.Path
	if len(devicePath) == 0 {
		return "", false, bosherr.Error("Getting real device path: path is missing")
	}

	realPath, found := dpr.findPossibleDevice(devicePath)

	for !found {
		if time.Now().After(stopAfter) {
			return "", true, bosherr.Errorf("Timed out getting real device path for %s", devicePath)
		}

		time.Sleep(100 * time.Millisecond)

		realPath, found = dpr.findPossibleDevice(devicePath)
	}

	return realPath, false, nil
}

func (dpr mappedDevicePathResolver) findPossibleDevice(devicePath string) (string, bool) {
	needsMapping := strings.HasPrefix(devicePath, "/dev/sd")

	if needsMapping {
		pathSuffix := strings.Split(devicePath, "/dev/sd")[1]

		possiblePrefixes := []string{
			"/dev/xvd", // Xen
			"/dev/vd",  // KVM
			"/dev/sd",
		}

		for _, prefix := range possiblePrefixes {
			path := prefix + pathSuffix
			if dpr.fs.FileExists(path) {
				return path, true
			}
		}
	} else {
		if dpr.fs.FileExists(devicePath) {
			return devicePath, true
		}
	}

	return "", false
}
