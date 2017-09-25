package cdrom

import (
	"os"
	"path/filepath"

	"errors"
	boshdevutil "github.com/cloudfoundry/bosh-agent/platform/deviceutil"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type cdUtil struct {
	settingsMountPath string
	fs                boshsys.FileSystem
	cdrom             Cdrom
	logger            boshlog.Logger
	logTag            string
}

func NewCdUtil(settingsMountPath string, fs boshsys.FileSystem, cdrom Cdrom, logger boshlog.Logger) boshdevutil.DeviceUtil {
	return cdUtil{
		settingsMountPath: settingsMountPath,
		fs:                fs,
		cdrom:             cdrom,
		logger:            logger,
		logTag:            "cdUtil",
	}
}

func (util cdUtil) GetFilesContents(fileNames []string) ([][]byte, error) {
	err := util.cdrom.WaitForMedia()
	if err != nil {
		return [][]byte{}, bosherr.WrapError(err, "Waiting for CDROM to be ready")
	}

	util.logger.Debug(util.logTag, "Mkdiring %s", util.settingsMountPath)
	err = util.fs.MkdirAll(util.settingsMountPath, os.FileMode(0700))
	if err != nil {
		return [][]byte{}, bosherr.WrapError(err, "Creating CDROM mount point")
	}

	util.logger.Debug(util.logTag, "Mounting %s", util.settingsMountPath)
	err = util.cdrom.Mount(util.settingsMountPath)
	if err != nil {
		return [][]byte{}, bosherr.WrapError(err, "Mounting CDROM")
	}

	contents := [][]byte{}
	for _, fileName := range fileNames {
		settingsPath := filepath.Join(util.settingsMountPath, fileName)
		util.logger.Debug(util.logTag, "Reading %s", settingsPath)
		stringContents, err := util.fs.ReadFile(settingsPath)
		if err != nil {
			return [][]byte{}, bosherr.WrapError(err, "Reading from CDROM")
		}

		contents = append(contents, []byte(stringContents))
	}

	util.logger.Debug(util.logTag, "Umounting CDROM")
	err = util.cdrom.Unmount()
	if err != nil {
		return [][]byte{}, bosherr.WrapError(err, "Unmounting CDROM")
	}

	util.logger.Debug(util.logTag, "Ejecting CDROM")
	err = util.cdrom.Eject()
	if err != nil {
		return [][]byte{}, bosherr.WrapError(err, "Ejecting CDROM")
	}

	return contents, nil
}

func (util cdUtil) GetBlockDeviceSize() (size uint64, err error) {
	return 0, errors.New("not supported")
}
