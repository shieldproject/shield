package udevdevice

import (
	"os"
	"time"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type ConcreteUdevDevice struct {
	runner boshsys.CmdRunner
	logger boshlog.Logger
	logtag string
}

func NewConcreteUdevDevice(runner boshsys.CmdRunner, logger boshlog.Logger) ConcreteUdevDevice {
	return ConcreteUdevDevice{
		runner: runner,
		logger: logger,
		logtag: "ConcreteUdevDevice",
	}
}

func (udev ConcreteUdevDevice) KickDevice(filePath string) {
	maxTries := 5
	for i := 0; i < maxTries; i++ {
		udev.logger.Debug(udev.logtag, "Kicking device, attempt %d of %d", i, maxTries)
		err := udev.readByte(filePath)
		if err == nil {
			break
		}
		time.Sleep(time.Second / 2)
	}

	if err := udev.readByte(filePath); err != nil {
		udev.logger.Error(udev.logtag, "Failed to red byte from device: %s", err.Error())
	}

	return
}

func (udev ConcreteUdevDevice) Settle() (err error) {
	udev.logger.Debug(udev.logtag, "Settling UdevDevice")
	switch {
	case udev.runner.CommandExists("udevadm"):
		_, _, _, err = udev.runner.RunCommand("udevadm", "settle")
	case udev.runner.CommandExists("udevsettle"):
		_, _, _, err = udev.runner.RunCommand("udevsettle")
	default:
		err = bosherr.Error("can not find udevadm or udevsettle commands")
	}
	return
}

func (udev ConcreteUdevDevice) Trigger() (err error) {
	udev.logger.Debug(udev.logtag, "Triggering UdevDevice")
	switch {
	case udev.runner.CommandExists("udevadm"):
		_, _, _, err = udev.runner.RunCommand("udevadm", "trigger")
	case udev.runner.CommandExists("udevtrigger"):
		_, _, _, err = udev.runner.RunCommand("udevtrigger")
	default:
		err = bosherr.Error("can not find udevadm or udevtrigger commands")
	}
	return
}

func (udev ConcreteUdevDevice) EnsureDeviceReadable(filePath string) error {
	maxTries := 5
	for i := 0; i < maxTries; i++ {
		udev.logger.Debug(udev.logtag, "Ensuring Device Readable, Attempt %d out of %d", i, maxTries)
		err := udev.readByte(filePath)
		if err != nil {
			udev.logger.Debug(udev.logtag, "Ignorable error from readByte: %s", err.Error())
		}

		time.Sleep(time.Second / 2)
	}

	err := udev.readByte(filePath)
	if err != nil {
		return bosherr.WrapError(err, "Reading udev device")
	}

	return nil
}

func (udev ConcreteUdevDevice) readByte(filePath string) error {
	udev.logger.Debug(udev.logtag, "readBytes from file: %s", filePath)
	device, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() {
		if err = device.Close(); err != nil {
			udev.logger.Warn(udev.logtag, "Failed to close device: %s", err.Error())
		}
	}()
	udev.logger.Debug(udev.logtag, "Successfully open file: %s", filePath)

	bytes := make([]byte, 1, 1)
	read, err := device.Read(bytes)
	if err != nil {
		return err
	}
	udev.logger.Debug(udev.logtag, "Successfully read %d bytes from file: %s", read, filePath)

	if read != 1 {
		return bosherr.Error("Device readable but zero length")
	}

	return nil
}
