package action_test

import (
	fakeplatform "github.com/cloudfoundry/bosh-agent/platform/fakes"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	fakesettings "github.com/cloudfoundry/bosh-agent/settings/fakes"
	boshassert "github.com/cloudfoundry/bosh-utils/assert"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent/action"
)

var _ = Describe("UnmountDiskAction", func() {
	var (
		platform *fakeplatform.FakePlatform
		action   UnmountDiskAction

		expectedDiskSettings boshsettings.DiskSettings
	)

	BeforeEach(func() {
		platform = fakeplatform.NewFakePlatform()

		settingsService := &fakesettings.FakeSettingsService{
			Settings: boshsettings.Settings{
				Disks: boshsettings.Disks{
					Persistent: map[string]interface{}{
						"vol-123": map[string]interface{}{
							"volume_id":      "2",
							"path":           "/dev/sdf",
							"lun":            "0",
							"host_device_id": "fake-host-device-id",
						},
					},
				},
				Env: boshsettings.Env{
					PersistentDiskFS: "ext4",
				},
			},
		}
		action = NewUnmountDisk(settingsService, platform)

		expectedDiskSettings = boshsettings.DiskSettings{
			ID:             "vol-123",
			VolumeID:       "2",
			Path:           "/dev/sdf",
			FileSystemType: "ext4",
			Lun:            "0",
			HostDeviceID:   "fake-host-device-id",
		}
	})

	It("is asynchronous", func() {
		Expect(action.IsAsynchronous()).To(BeTrue())
	})

	It("is not persistent", func() {
		Expect(action.IsPersistent()).To(BeFalse())
	})

	It("unmount disk when the disk is mounted", func() {
		platform.UnmountPersistentDiskDidUnmount = true

		result, err := action.Run("vol-123")
		Expect(err).ToNot(HaveOccurred())
		boshassert.MatchesJSONString(GinkgoT(), result, `{"message":"Unmounted partition of {ID:vol-123 DeviceID: VolumeID:2 Lun:0 HostDeviceID:fake-host-device-id Path:/dev/sdf FileSystemType:ext4}"}`)

		Expect(platform.UnmountPersistentDiskSettings).To(Equal(expectedDiskSettings))
	})

	It("unmount disk when the disk is not mounted", func() {
		platform.UnmountPersistentDiskDidUnmount = false

		result, err := action.Run("vol-123")
		Expect(err).ToNot(HaveOccurred())
		boshassert.MatchesJSONString(GinkgoT(), result, `{"message":"Partition of {ID:vol-123 DeviceID: VolumeID:2 Lun:0 HostDeviceID:fake-host-device-id Path:/dev/sdf FileSystemType:ext4} is not mounted"}`)

		Expect(platform.UnmountPersistentDiskSettings).To(Equal(expectedDiskSettings))
	})

	It("unmount disk when device path not found", func() {
		_, err := action.Run("vol-456")
		Expect(err).To(HaveOccurred())
	})
})
