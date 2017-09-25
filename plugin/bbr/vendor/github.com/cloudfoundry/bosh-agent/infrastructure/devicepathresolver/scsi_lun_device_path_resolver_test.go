package devicepathresolver_test

import (
	"os"
	"time"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/infrastructure/devicepathresolver"
)

var _ = Describe("SCSILunDevicePathResolver", func() {
	var (
		fs           *fakesys.FakeFileSystem
		diskSettings boshsettings.DiskSettings
		pathResolver DevicePathResolver
		hosts        []string
	)

	BeforeEach(func() {
		lun := "0"
		fs = fakesys.NewFakeFileSystem()
		pathResolver = NewSCSILunDevicePathResolver(500*time.Millisecond, fs, boshlog.NewLogger(boshlog.LevelNone))
		diskSettings = boshsettings.DiskSettings{
			Lun:          lun,
			HostDeviceID: "fake-host-device-id",
		}

		hosts = []string{
			"/sys/class/scsi_host/host0/scan",
			"/sys/class/scsi_host/host1/scan",
			"/sys/class/scsi_host/host2/scan",
			"/sys/class/scsi_host/host3/scan",
			"/sys/class/scsi_host/host4/scan",
			"/sys/class/scsi_host/host5/scan",
		}
		fs.SetGlob("/sys/class/scsi_host/host*/scan", hosts)
		fs.SetGlob("/sys/bus/scsi/devices/*:*:*:"+lun+"/block/*", []string{
			"/sys/bus/scsi/devices/2:0:0:0/block/sda",
			"/sys/bus/scsi/devices/3:0:1:0/block/sdb",
			"/sys/bus/scsi/devices/5:0:0:" + lun + "/block/sdc",
		})
		fs.SetGlob("/sys/bus/vmbus/devices/*/device_id", []string{
			"/sys/bus/vmbus/devices/fake-vmbus-device/device_id",
			"/sys/bus/vmbus/devices/vmbus_0_12/device_id",
			"/sys/bus/vmbus/devices/vmbus_12/device_id",
		})
	})

	Describe("GetRealDevicePath", func() {
		Context("when path exists", func() {
			BeforeEach(func() {
				deviceIDPath := "/sys/bus/vmbus/devices/fake-vmbus-device/device_id"
				err := fs.WriteFileString(deviceIDPath, "fake-host-device-id")
				Expect(err).ToNot(HaveOccurred())

				err = fs.MkdirAll("fake-root/fake-vmbus-device/fake-base", os.FileMode(0750))
				Expect(err).ToNot(HaveOccurred())

				err = fs.Symlink("fake-root/fake-vmbus-device/fake-base", "/sys/class/block/sdc")
				Expect(err).ToNot(HaveOccurred())

				err = fs.MkdirAll("/dev/sdc", os.FileMode(0750))
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns the real path", func() {
				path, timeout, err := pathResolver.GetRealDevicePath(diskSettings)
				Expect(err).ToNot(HaveOccurred())

				Expect(path).To(Equal("/dev/sdc"))
				Expect(timeout).To(BeFalse())

				for _, host := range hosts {
					str, _ := fs.ReadFileString(host)
					Expect(str).To(Equal("- - -"))
				}
			})
		})

		Context("when the given host device id cannot be found", func() {
			BeforeEach(func() {
				deviceIDPath := "/sys/bus/vmbus/devices/fake-vmbus-device/device_id"
				err := fs.WriteFileString(deviceIDPath, "fake-wrong-host-device-id")
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				_, _, err := pathResolver.GetRealDevicePath(diskSettings)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when symlink does not exist", func() {
			It("returns an error", func() {
				deviceIDPath := "/sys/bus/vmbus/devices/fake-vmbus-device/device_id"
				err := fs.WriteFileString(deviceIDPath, "fake-host-device-id")
				Expect(err).ToNot(HaveOccurred())

				_, _, err = pathResolver.GetRealDevicePath(diskSettings)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when path does not exist", func() {
			BeforeEach(func() {
				deviceIDPath := "/sys/bus/vmbus/devices/fake-vmbus-device/device_id"
				err := fs.WriteFileString(deviceIDPath, "fake-host-device-id")
				Expect(err).ToNot(HaveOccurred())

				err = fs.Symlink("fake-root/fake-vmbus-device/fake-base", "/sys/class/block/sdc")
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				_, _, err := pathResolver.GetRealDevicePath(diskSettings)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when the disk path does not exist", func() {
			BeforeEach(func() {
				deviceIDPath := "/sys/bus/vmbus/devices/fake-vmbus-device/device_id"
				err := fs.WriteFileString(deviceIDPath, "fake-host-device-id")
				Expect(err).ToNot(HaveOccurred())

				err = fs.MkdirAll("fake-root/fake-vmbus-device/fake-base", os.FileMode(0750))
				Expect(err).ToNot(HaveOccurred())

				err = fs.Symlink("fake-root/fake-vmbus-device/fake-base", "/sys/class/block/sdc")
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				_, _, err := pathResolver.GetRealDevicePath(diskSettings)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when no matching device is found the first time", func() {
			Context("when the timeout has not expired", func() {
				BeforeEach(func() {
					deviceIDPath := "/sys/bus/vmbus/devices/fake-vmbus-device/device_id"
					err := fs.WriteFileString(deviceIDPath, "fake-host-device-id")
					Expect(err).ToNot(HaveOccurred())

					time.AfterFunc(100*time.Millisecond, func() {
						err := fs.MkdirAll("fake-root/fake-vmbus-device/fake-base", os.FileMode(0750))
						Expect(err).ToNot(HaveOccurred())

						err = fs.Symlink("fake-root/fake-vmbus-device/fake-base", "/sys/class/block/sdc")
						Expect(err).ToNot(HaveOccurred())

						err = fs.MkdirAll("/dev/sdc", os.FileMode(0750))
						Expect(err).ToNot(HaveOccurred())
					})
				})

				It("returns the real path", func() {
					path, timeout, err := pathResolver.GetRealDevicePath(diskSettings)
					Expect(err).ToNot(HaveOccurred())

					Expect(path).To(Equal("/dev/sdc"))
					Expect(timeout).To(BeFalse())
				})
			})

			Context("when the timeout has expired", func() {
				BeforeEach(func() {
					deviceIDPath := "/sys/bus/vmbus/devices/fake-vmbus-device/device_id"
					err := fs.WriteFileString(deviceIDPath, "fake-host-device-id")
					Expect(err).ToNot(HaveOccurred())

					fs.SetGlob("/sys/bus/scsi/devices/*:*:*:7/block/*", []string{})
				})

				It("returns an error", func() {
					path, timeout, err := pathResolver.GetRealDevicePath(boshsettings.DiskSettings{
						Lun:          "7",
						HostDeviceID: "fake-host-device-id",
					})
					Expect(err).To(HaveOccurred())

					Expect(path).To(Equal(""))
					Expect(timeout).To(BeTrue())
				})
			})
		})

		Context("when lun is empty", func() {
			BeforeEach(func() {
				diskSettings = boshsettings.DiskSettings{
					HostDeviceID: "fake-host-device-id",
				}
			})

			It("returns an error", func() {
				_, timeout, err := pathResolver.GetRealDevicePath(diskSettings)
				Expect(err).To(HaveOccurred())
				Expect(timeout).To(BeFalse())
			})
		})

		Context("when host_device_id is empty", func() {
			BeforeEach(func() {
				diskSettings = boshsettings.DiskSettings{
					Lun: "0",
				}
			})

			It("returns an error", func() {
				_, timeout, err := pathResolver.GetRealDevicePath(diskSettings)
				Expect(err).To(HaveOccurred())
				Expect(timeout).To(BeFalse())
			})
		})
	})
})
