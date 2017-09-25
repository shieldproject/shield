package devicepathresolver_test

import (
	"os"
	"strings"
	"time"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/infrastructure/devicepathresolver"
)

var _ = Describe("ScsiIDDevicePathResolver", func() {
	var (
		fs           *fakesys.FakeFileSystem
		diskSettings boshsettings.DiskSettings
		pathResolver DevicePathResolver
		id           string
		hosts        []string
	)

	BeforeEach(func() {
		deviceID := "ab1b46b5-bf22-4332-bddd-12a05ea1a5fc"
		id = strings.Replace(deviceID, "-", "", -1)
		fs = fakesys.NewFakeFileSystem()
		pathResolver = NewSCSIIDDevicePathResolver(500*time.Millisecond, fs, boshlog.NewLogger(boshlog.LevelNone))
		diskSettings = boshsettings.DiskSettings{
			DeviceID: deviceID,
		}

		hosts = []string{
			"/sys/class/scsi_host/host0/scan",
			"/sys/class/scsi_host/host1/scan",
			"/sys/class/scsi_host/host2/scan",
		}
		fs.SetGlob("/sys/class/scsi_host/host*/scan", hosts)
		fs.SetGlob("/dev/disk/by-id/*"+id, []string{
			"/dev/disk/by-id/scsi-3" + id,
		})
	})

	Describe("GetRealDevicePath", func() {
		Context("when path exists", func() {
			BeforeEach(func() {
				err := fs.MkdirAll("fake-device-path", os.FileMode(0750))
				Expect(err).ToNot(HaveOccurred())

				err = fs.Symlink("fake-device-path", "/dev/disk/by-id/scsi-3"+id)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns the path", func() {
				path, timeout, err := pathResolver.GetRealDevicePath(diskSettings)
				Expect(err).ToNot(HaveOccurred())

				Expect(path).To(Equal("fake-device-path"))
				Expect(timeout).To(BeFalse())

				for _, host := range hosts {
					str, _ := fs.ReadFileString(host)
					Expect(str).To(Equal("- - -"))
				}
			})
		})

		Context("when path does not exist", func() {
			BeforeEach(func() {
				err := fs.Symlink("fake-device-path", "/dev/disk/by-id/scsi-3"+id)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				_, _, err := pathResolver.GetRealDevicePath(diskSettings)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when symlink does not exist", func() {
			It("returns an error", func() {
				_, _, err := pathResolver.GetRealDevicePath(diskSettings)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when no matching device is found the first time", func() {
			Context("when the timeout has not expired", func() {
				BeforeEach(func() {
					time.AfterFunc(100*time.Millisecond, func() {
						err := fs.MkdirAll("fake-device-path", os.FileMode(0750))
						Expect(err).ToNot(HaveOccurred())

						err = fs.Symlink("fake-device-path", "/dev/disk/by-id/scsi-3"+id)
						Expect(err).ToNot(HaveOccurred())
					})
				})

				It("returns the real path", func() {
					path, timeout, err := pathResolver.GetRealDevicePath(diskSettings)
					Expect(err).ToNot(HaveOccurred())

					Expect(path).To(Equal("fake-device-path"))
					Expect(timeout).To(BeFalse())
				})
			})
		})

		Context("when id is empty", func() {
			BeforeEach(func() {
				diskSettings = boshsettings.DiskSettings{}
			})

			It("returns an error", func() {
				_, timeout, err := pathResolver.GetRealDevicePath(diskSettings)
				Expect(err).To(HaveOccurred())
				Expect(timeout).To(BeFalse())
			})
		})
	})
})
