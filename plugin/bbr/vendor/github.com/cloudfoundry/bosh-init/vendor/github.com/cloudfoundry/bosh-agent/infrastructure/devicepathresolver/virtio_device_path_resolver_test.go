package devicepathresolver_test

import (
	"errors"

	fakedpresolv "github.com/cloudfoundry/bosh-agent/infrastructure/devicepathresolver/fakes"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/infrastructure/devicepathresolver"
)

var _ = Describe("VirtioDevicePathResolver", func() {
	var (
		pathResolver             DevicePathResolver
		idDevicePathResolver     *fakedpresolv.FakeDevicePathResolver
		mappedDevicePathResolver *fakedpresolv.FakeDevicePathResolver

		diskSettings boshsettings.DiskSettings
	)

	BeforeEach(func() {
		idDevicePathResolver = fakedpresolv.NewFakeDevicePathResolver()
		mappedDevicePathResolver = fakedpresolv.NewFakeDevicePathResolver()
		logger := boshlog.NewLogger(boshlog.LevelNone)
		pathResolver = NewVirtioDevicePathResolver(idDevicePathResolver, mappedDevicePathResolver, logger)

		diskSettings = boshsettings.DiskSettings{
			ID:       "fake-disk-id",
			VolumeID: "fake-volume-id",
			Path:     "fake-disk-path",
		}
	})

	Describe("GetRealDevicePath", func() {
		Context("when idDevicePathResolver returns path", func() {
			BeforeEach(func() {
				idDevicePathResolver.RealDevicePath = "fake-id-resolved-device-path"
			})

			It("returns the path", func() {
				realPath, timeout, err := pathResolver.GetRealDevicePath(diskSettings)
				Expect(err).ToNot(HaveOccurred())
				Expect(timeout).To(BeFalse())
				Expect(realPath).To(Equal("fake-id-resolved-device-path"))
			})
		})

		Context("when idDevicePathResolver errors", func() {
			BeforeEach(func() {
				idDevicePathResolver.GetRealDevicePathErr = errors.New("fake-id-error")
			})

			It("calls mappedDevicePathResolver", func() {
				mappedDevicePathResolver.RealDevicePath = "fake-mapped-resolved-device-path"

				realPath, timeout, err := pathResolver.GetRealDevicePath(diskSettings)
				Expect(err).ToNot(HaveOccurred())
				Expect(timeout).To(BeFalse())
				Expect(realPath).To(Equal("fake-mapped-resolved-device-path"))

				Expect(mappedDevicePathResolver.GetRealDevicePathDiskSettings).To(Equal(diskSettings))
			})

			Context("when mappedDevicePathResolver times out", func() {
				BeforeEach(func() {
					mappedDevicePathResolver.GetRealDevicePathErr = errors.New("fake-id-error")
					mappedDevicePathResolver.GetRealDevicePathTimedOut = true
				})

				It("returns timeout", func() {
					_, timeout, err := pathResolver.GetRealDevicePath(diskSettings)
					Expect(err).To(HaveOccurred())
					Expect(timeout).To(BeTrue())
				})
			})

			Context("when mappedDevicePathResolver errors", func() {
				BeforeEach(func() {
					mappedDevicePathResolver.GetRealDevicePathErr = errors.New("fake-mapped-error")
				})

				It("returns error", func() {
					_, timeout, err := pathResolver.GetRealDevicePath(diskSettings)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-mapped-error"))
					Expect(timeout).To(BeFalse())
				})
			})
		})
	})
})
