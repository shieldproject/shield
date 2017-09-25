package devicepathresolver_test

import (
	"errors"
	"os"
	"time"

	fakeudev "github.com/cloudfoundry/bosh-agent/platform/udevdevice/fakes"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/infrastructure/devicepathresolver"
)

var _ = Describe("IDDevicePathResolver", func() {
	var (
		fs           *fakesys.FakeFileSystem
		udev         *fakeudev.FakeUdevDevice
		diskSettings boshsettings.DiskSettings
		pathResolver DevicePathResolver
	)

	BeforeEach(func() {
		udev = fakeudev.NewFakeUdevDevice()
		fs = fakesys.NewFakeFileSystem()
		diskSettings = boshsettings.DiskSettings{
			ID: "fake-disk-id-include-truncate",
		}
	})

	JustBeforeEach(func() {
		pathResolver = NewIDDevicePathResolver(500*time.Millisecond, udev, fs)
	})

	Describe("GetRealDevicePath", func() {
		It("refreshes udev", func() {
			pathResolver.GetRealDevicePath(diskSettings)
			Expect(udev.Triggered).To(Equal(true))
			Expect(udev.Settled).To(Equal(true))
		})

		Context("when path exists", func() {
			BeforeEach(func() {
				err := fs.MkdirAll("/dev/fake-device-path", os.FileMode(0750))
				Expect(err).ToNot(HaveOccurred())

				err = fs.Symlink("/dev/fake-device-path", "/dev/intermediate/fake-device-path")
				Expect(err).ToNot(HaveOccurred())

				err = fs.Symlink("/dev/intermediate/fake-device-path", "/dev/disk/by-id/virtio-fake-disk-id-include")
				Expect(err).ToNot(HaveOccurred())

				fs.SetGlob("/dev/disk/by-id/*fake-disk-id-include", []string{"/dev/disk/by-id/virtio-fake-disk-id-include"})
			})

			It("returns fully resolved the path (not potentially relative symlink target)", func() {
				path, timeout, err := pathResolver.GetRealDevicePath(diskSettings)
				Expect(err).ToNot(HaveOccurred())

				Expect(path).To(Equal("/dev/fake-device-path"))
				Expect(timeout).To(BeFalse())
			})
		})

		Context("when disks with the same ID but different virtio prefixes exist ", func() {
			BeforeEach(func() {
				err := fs.MkdirAll("fake-device-path-1", os.FileMode(0750))
				Expect(err).ToNot(HaveOccurred())
				err = fs.MkdirAll("fake-device-path-2", os.FileMode(0750))
				Expect(err).ToNot(HaveOccurred())

				err = fs.Symlink("fake-device-path-1", "/dev/disk/by-id/virtio-fake-disk-id-include")
				Expect(err).ToNot(HaveOccurred())
				err = fs.Symlink("fake-device-path-2", "/dev/disk/by-id/customprefix-fake-disk-id-include")
				Expect(err).ToNot(HaveOccurred())

				fs.SetGlob("/dev/disk/by-id/*fake-disk-id-include", []string{
					"/dev/disk/by-id/virtio-fake-disk-id-include",
					"/dev/disk/by-id/customprefix-fake-disk-id-include",
				})
			})
			It("returns an error", func() {
				_, _, err := pathResolver.GetRealDevicePath(diskSettings)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("More than one disk matched"))
			})
		})

		Context("when path does not exist", func() {
			BeforeEach(func() {
				err := fs.Symlink("fake-device-path", "/dev/disk/by-id/virtio-fake-disk-id-include")
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				_, _, err := pathResolver.GetRealDevicePath(diskSettings)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Timed out getting real device path for 'fake-disk-id-include'"))
			})
		})

		Context("when symlink does not exist", func() {
			It("returns an error", func() {
				_, _, err := pathResolver.GetRealDevicePath(diskSettings)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Timed out getting real device path for 'fake-disk-id-include'"))
			})
		})

		Context("when no matching device is found the first time", func() {
			Context("when the timeout has not expired", func() {
				BeforeEach(func() {
					err := fs.MkdirAll("fake-device-path", os.FileMode(0750))
					Expect(err).ToNot(HaveOccurred())

					err = fs.Symlink("fake-device-path", "/dev/disk/by-id/virtio-fake-disk-id-include")
					Expect(err).ToNot(HaveOccurred())

					fs.GlobStub = func(pattern string) ([]string, error) {
						fs.SetGlob("/dev/disk/by-id/*fake-disk-id-include", []string{
							"/dev/disk/by-id/virtio-fake-disk-id-include",
						})

						fs.GlobStub = nil

						return nil, errors.New("new error")
					}
				})

				It("returns the real path", func() {
					path, timeout, err := pathResolver.GetRealDevicePath(diskSettings)
					Expect(err).ToNot(HaveOccurred())

					Expect(path).To(Equal("fake-device-path"))
					Expect(timeout).To(BeFalse())
				})
			})
		})

		Context("when triggering udev fails", func() {
			BeforeEach(func() {
				udev.TriggerErr = errors.New("fake-udev-trigger-error")
			})

			It("returns an error", func() {
				_, timeout, err := pathResolver.GetRealDevicePath(diskSettings)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-udev-trigger-error"))
				Expect(timeout).To(BeFalse())
			})
		})

		Context("when settling udev fails", func() {
			BeforeEach(func() {
				udev.SettleErr = errors.New("fake-udev-settle-error")
			})

			It("returns an error", func() {
				_, timeout, err := pathResolver.GetRealDevicePath(diskSettings)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-udev-settle-error"))
				Expect(timeout).To(BeFalse())
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

		Context("when id is not the correct format", func() {
			BeforeEach(func() {
				diskSettings = boshsettings.DiskSettings{
					ID: "too-short",
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
