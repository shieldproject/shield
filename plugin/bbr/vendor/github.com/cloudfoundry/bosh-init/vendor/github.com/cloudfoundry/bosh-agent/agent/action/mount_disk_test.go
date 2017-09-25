package action_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent/action"
	fakeplatform "github.com/cloudfoundry/bosh-agent/platform/fakes"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshdirs "github.com/cloudfoundry/bosh-agent/settings/directories"
	fakesettings "github.com/cloudfoundry/bosh-agent/settings/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("MountDiskAction", func() {
	var (
		settingsService *fakesettings.FakeSettingsService
		platform        *fakeplatform.FakePlatform
		action          MountDiskAction
		logger          boshlog.Logger
	)

	BeforeEach(func() {
		settingsService = &fakesettings.FakeSettingsService{}
		platform = fakeplatform.NewFakePlatform()
		dirProvider := boshdirs.NewProvider("/fake-base-dir")
		logger = boshlog.NewLogger(boshlog.LevelNone)
		action = NewMountDisk(settingsService, platform, dirProvider, logger)
	})

	It("is asynchronous", func() {
		Expect(action.IsAsynchronous()).To(BeTrue())
	})

	It("is not persistent", func() {
		Expect(action.IsPersistent()).To(BeFalse())
	})

	Describe("Run", func() {
		Context("when settings can be loaded", func() {
			Context("when disk cid can be resolved to a device path from infrastructure settings", func() {
				BeforeEach(func() {
					settingsService.Settings.Disks.Persistent = map[string]interface{}{
						"fake-disk-cid": map[string]interface{}{
							"path":      "fake-device-path",
							"volume_id": "fake-volume-id",
						},
					}
				})

				Context("when mounting succeeds", func() {
					It("returns without an error after mounting store directory", func() {
						result, err := action.Run("fake-disk-cid")
						Expect(err).NotTo(HaveOccurred())
						Expect(result).To(Equal(map[string]string{}))

						Expect(platform.MountPersistentDiskSettings).To(Equal(boshsettings.DiskSettings{
							ID:       "fake-disk-cid",
							VolumeID: "fake-volume-id",
							Path:     "fake-device-path",
						}))
						Expect(platform.MountPersistentDiskMountPoint).To(Equal("/fake-base-dir/store"))
					})
				})

				Context("when mounting fails", func() {
					It("returns error after trying to mount store directory", func() {
						platform.MountPersistentDiskErr = errors.New("fake-mount-persistent-disk-err")

						_, err := action.Run("fake-disk-cid")
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("fake-mount-persistent-disk-err"))
					})
				})
			})

			Context("when disk cid cannot be resolved to a device path from infrastructure settings", func() {
				BeforeEach(func() {
					settingsService.Settings.Disks.Persistent = map[string]interface{}{
						"fake-known-disk-cid": "/dev/sdf",
					}
				})

				It("returns error", func() {
					_, err := action.Run("fake-unknown-disk-cid")
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("Persistent disk with volume id 'fake-unknown-disk-cid' could not be found"))
				})
			})
		})

		Context("when settings cannot be loaded", func() {
			It("returns error", func() {
				settingsService.LoadSettingsError = errors.New("fake-load-settings-err")

				_, err := action.Run("fake-disk-cid")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-load-settings-err"))
			})
		})
	})
})
