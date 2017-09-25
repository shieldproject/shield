package devicepathresolver_test

import (
	fakedpresolv "github.com/cloudfoundry/bosh-agent/infrastructure/devicepathresolver/fakes"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/infrastructure/devicepathresolver"
)

var _ = Describe("scsiDevicePathResolver", func() {
	var (
		scsiDevicePathResolver         DevicePathResolver
		scsiIDDevicePathResolver       *fakedpresolv.FakeDevicePathResolver
		scsiVolumeIDDevicePathResolver *fakedpresolv.FakeDevicePathResolver
		scsiLunDevicePathResolver      *fakedpresolv.FakeDevicePathResolver

		diskSettings boshsettings.DiskSettings
	)

	BeforeEach(func() {
		scsiIDDevicePathResolver = fakedpresolv.NewFakeDevicePathResolver()
		scsiVolumeIDDevicePathResolver = fakedpresolv.NewFakeDevicePathResolver()
		scsiLunDevicePathResolver = fakedpresolv.NewFakeDevicePathResolver()
		scsiDevicePathResolver = NewScsiDevicePathResolver(scsiVolumeIDDevicePathResolver, scsiIDDevicePathResolver, scsiLunDevicePathResolver)
	})

	Describe("GetRealDevicePath", func() {
		Context("when diskSettings provides device id", func() {
			BeforeEach(func() {
				diskSettings = boshsettings.DiskSettings{
					DeviceID: "fake-disk-id",
				}
			})

			It("returns the path using SCSIIDDevicePathResolver", func() {
				scsiIDDevicePathResolver.RealDevicePath = "fake-id-resolved-device-path"
				realPath, timeout, err := scsiDevicePathResolver.GetRealDevicePath(diskSettings)
				Expect(err).ToNot(HaveOccurred())
				Expect(timeout).To(BeFalse())
				Expect(realPath).To(Equal("fake-id-resolved-device-path"))

				Expect(scsiIDDevicePathResolver.GetRealDevicePathDiskSettings).To(Equal(diskSettings))
			})
		})

		Context("when diskSettings does not provides id but volume_id", func() {
			BeforeEach(func() {
				diskSettings = boshsettings.DiskSettings{
					VolumeID: "fake-disk-id",
				}
			})

			It("returns the path using SCSIVolumeIDDevicePathResolver", func() {
				scsiVolumeIDDevicePathResolver.RealDevicePath = "fake-volume-id-resolved-device-path"
				realPath, timeout, err := scsiDevicePathResolver.GetRealDevicePath(diskSettings)
				Expect(err).ToNot(HaveOccurred())
				Expect(timeout).To(BeFalse())
				Expect(realPath).To(Equal("fake-volume-id-resolved-device-path"))

				Expect(scsiVolumeIDDevicePathResolver.GetRealDevicePathDiskSettings).To(Equal(diskSettings))
			})
		})

		Context("when diskSettings does not provides id nor volume_id but lun", func() {
			Context("When both lun and scsi_host_id are provided", func() {
				BeforeEach(func() {
					diskSettings = boshsettings.DiskSettings{
						Lun:          "fake-lun-id",
						HostDeviceID: "fake-host-device-id",
					}
				})

				It("returns the path using SCSILunDevicePathResolver", func() {
					scsiLunDevicePathResolver.RealDevicePath = "fake-lun-resolved-device-path"
					realPath, timeout, err := scsiDevicePathResolver.GetRealDevicePath(diskSettings)
					Expect(err).ToNot(HaveOccurred())
					Expect(timeout).To(BeFalse())
					Expect(realPath).To(Equal("fake-lun-resolved-device-path"))

					Expect(scsiLunDevicePathResolver.GetRealDevicePathDiskSettings).To(Equal(diskSettings))
				})
			})

			Context("When scsi_host_id is not provided", func() {
				BeforeEach(func() {
					diskSettings = boshsettings.DiskSettings{
						Lun: "fake-lun-id",
					}
				})

				It("returns an error", func() {
					realPath, timeout, err := scsiDevicePathResolver.GetRealDevicePath(diskSettings)
					Expect(err).To(HaveOccurred())
					Expect(timeout).To(BeFalse())
					Expect(realPath).To(Equal(""))
				})
			})
		})

		Context("when diskSettings does not provides id, volume_id nor (lun, scsi_host_id)", func() {
			BeforeEach(func() {
				diskSettings = boshsettings.DiskSettings{}
			})

			It("returns the path using SCSIVolumeIDDevicePathResolver", func() {
				realPath, timeout, err := scsiDevicePathResolver.GetRealDevicePath(diskSettings)
				Expect(err).To(HaveOccurred())
				Expect(timeout).To(BeFalse())
				Expect(realPath).To(Equal(""))
			})
		})
	})
})
