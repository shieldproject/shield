package infrastructure_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/infrastructure"
	fakeplatform "github.com/cloudfoundry/bosh-agent/platform/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("ConfigDriveSettingsSource", func() {
	var (
		platform *fakeplatform.FakePlatform
		source   *CDROMSettingsSource
	)

	BeforeEach(func() {
		settingsFileName := "fake-settings-file-name"
		platform = fakeplatform.NewFakePlatform()
		logger := boshlog.NewLogger(boshlog.LevelNone)
		source = NewCDROMSettingsSource(settingsFileName, platform, logger)
	})

	Describe("PublicSSHKeyForUsername", func() {
		It("returns an empty string", func() {
			publicKey, err := source.PublicSSHKeyForUsername("fake-username")
			Expect(err).ToNot(HaveOccurred())
			Expect(publicKey).To(Equal(""))
		})
	})

	Describe("Settings", func() {
		It("returns settings read from the CDROM", func() {
			platform.GetFileContentsFromCDROMContents = []byte(`{"agent_id": "123"}`)

			settings, err := source.Settings()
			Expect(err).ToNot(HaveOccurred())

			Expect(platform.GetFileContentsFromCDROMPath).To(Equal("fake-settings-file-name"))
			Expect(settings.AgentID).To(Equal("123"))
		})

		It("returns an error if reading from the CDROM fails", func() {
			platform.GetFileContentsFromCDROMErr = errors.New("fake-read-disk-error")

			_, err := source.Settings()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-read-disk-error"))
		})
	})
})
