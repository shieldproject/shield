package devicepathresolver_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"

	. "github.com/cloudfoundry/bosh-agent/infrastructure/devicepathresolver"
)

var _ = Describe("SCSIVolumeIDDevicePathResolver", func() {
	var (
		fs           *fakesys.FakeFileSystem
		resolver     DevicePathResolver
		diskSettings boshsettings.DiskSettings
	)

	const sleepInterval = time.Millisecond * 1

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		resolver = NewSCSIVolumeIDDevicePathResolver(sleepInterval, fs)

		fs.SetGlob("/sys/bus/scsi/devices/*:0:0:0/block/*", []string{
			"/sys/bus/scsi/devices/0:0:0:0/block/sr0",
			"/sys/bus/scsi/devices/6:0:0:0/block/sdd",
			"/sys/bus/scsi/devices/fake-host-id:0:0:0/block/sda",
		})

		fs.SetGlob("/sys/bus/scsi/devices/fake-host-id:0:fake-disk-id:0/block/*", []string{
			"/sys/bus/scsi/devices/fake-host-id:0:fake-disk-id:0/block/sdf",
		})

		diskSettings = boshsettings.DiskSettings{
			VolumeID: "fake-disk-id",
		}
	})

	Describe("GetRealDevicePath", func() {
		It("rescans the devices attached to the root disks scsi controller", func() {
			resolver.GetRealDevicePath(diskSettings)

			scanContents, err := fs.ReadFileString("/sys/class/scsi_host/hostfake-host-id/scan")
			Expect(err).NotTo(HaveOccurred())
			Expect(scanContents).To(Equal("- - -"))
		})

		It("detects device", func() {
			devicePath, timedOut, err := resolver.GetRealDevicePath(diskSettings)
			Expect(err).NotTo(HaveOccurred())
			Expect(timedOut).To(BeFalse())
			Expect(devicePath).To(Equal("/dev/sdf"))
		})

		Context("when device does not immediately appear", func() {
			It("retries detection of device", func() {
				fs.SetGlob("/sys/bus/scsi/devices/fake-host-id:0:fake-disk-id:0/block/*",
					[]string{},
					[]string{},
					[]string{},
					[]string{},
					[]string{},
					[]string{"/sys/bus/scsi/devices/fake-host-id:0:fake-disk-id:0/block/sdf"},
				)

				startTime := time.Now()
				devicePath, timedOut, err := resolver.GetRealDevicePath(diskSettings)
				runningTime := time.Since(startTime)
				Expect(err).NotTo(HaveOccurred())
				Expect(timedOut).To(BeFalse())
				Expect(runningTime >= sleepInterval).To(BeTrue())
				Expect(devicePath).To(Equal("/dev/sdf"))
			})
		})

		Context("when device is found", func() {
			It("does not retry detection of device", func() {
				fs.SetGlob("/sys/bus/scsi/devices/fake-host-id:0:fake-disk-id:0/block/*",
					[]string{"/sys/bus/scsi/devices/fake-host-id:0:fake-disk-id:0/block/sdf"},
					[]string{},
					[]string{},
					[]string{},
					[]string{},
					[]string{"/sys/bus/scsi/devices/fake-host-id:0:fake-disk-id:0/block/bla"},
				)

				devicePath, timedOut, err := resolver.GetRealDevicePath(diskSettings)
				Expect(err).NotTo(HaveOccurred())
				Expect(timedOut).To(BeFalse())
				Expect(devicePath).To(Equal("/dev/sdf"))
			})
		})

		Context("when device never appears", func() {
			It("returns not err", func() {
				fs.SetGlob("/sys/bus/scsi/devices/fake-host-id:0:fake-disk-id:0/block/*", []string{})
				_, _, err := resolver.GetRealDevicePath(diskSettings)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
