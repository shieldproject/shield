package devicepathresolver

import (
	"strings"
	"time"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

// SCSIIDDevicePathResolver resolves device path by performing a
// SCSI rescan then looking under "/dev/disk/by-id/*uuid"
// where "uuid" is the cloud ID of the disk
type SCSIIDDevicePathResolver struct {
	diskWaitTimeout time.Duration
	fs              boshsys.FileSystem

	logTag string
	logger boshlog.Logger
}

func NewSCSIIDDevicePathResolver(
	diskWaitTimeout time.Duration,
	fs boshsys.FileSystem,
	logger boshlog.Logger,
) SCSIIDDevicePathResolver {
	return SCSIIDDevicePathResolver{
		diskWaitTimeout: diskWaitTimeout,
		fs:              fs,

		logTag: "scsiIDresolver",
		logger: logger,
	}
}

func (idpr SCSIIDDevicePathResolver) GetRealDevicePath(diskSettings boshsettings.DiskSettings) (string, bool, error) {
	if diskSettings.DeviceID == "" {
		return "", false, bosherr.Errorf("Disk device ID is not set")
	}

	hostPaths, err := idpr.fs.Glob("/sys/class/scsi_host/host*/scan")
	if err != nil {
		return "", false, bosherr.WrapError(err, "Could not list SCSI hosts")
	}

	for _, hostPath := range hostPaths {
		idpr.logger.Info(idpr.logTag, "Performing SCSI rescan of "+hostPath)
		err = idpr.fs.WriteFileString(hostPath, "- - -")
		if err != nil {
			return "", false, bosherr.WrapError(err, "Starting SCSI rescan")
		}
	}

	stopAfter := time.Now().Add(idpr.diskWaitTimeout)
	found := false

	var realPath string

	for !found {
		idpr.logger.Debug(idpr.logTag, "Waiting for device to appear")

		if time.Now().After(stopAfter) {
			return "", true, bosherr.Errorf("Timed out getting real device path for '%s'", diskSettings.DeviceID)
		}

		time.Sleep(100 * time.Millisecond)

		uuid := strings.Replace(diskSettings.DeviceID, "-", "", -1)
		disks, err := idpr.fs.Glob("/dev/disk/by-id/*" + uuid)
		if err != nil {
			return "", false, bosherr.WrapError(err, "Could not list disks by id")
		}
		for _, path := range disks {
			idpr.logger.Debug(idpr.logTag, "Reading link "+path)
			realPath, err = idpr.fs.ReadLink(path)
			if err != nil {
				continue
			}

			if idpr.fs.FileExists(realPath) {
				idpr.logger.Debug(idpr.logTag, "Found real path "+realPath)
				found = true
				break
			}
		}
	}

	return realPath, false, nil
}
