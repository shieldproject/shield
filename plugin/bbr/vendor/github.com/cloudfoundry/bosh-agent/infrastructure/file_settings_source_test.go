package infrastructure_test

import (
	"encoding/json"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/infrastructure"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
)

var _ = Describe("FileSettingsSource", func() {
	var (
		fs     *fakesys.FakeFileSystem
		source *FileSettingsSource
		logger boshlog.Logger
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		logger = boshlog.NewLogger(boshlog.LevelNone)
	})

	Describe("PublicSSHKeyForUsername", func() {
		It("returns an empty string", func() {
			publicKey, err := source.PublicSSHKeyForUsername("fake-username")
			Expect(err).ToNot(HaveOccurred())
			Expect(publicKey).To(Equal(""))
		})
	})

	Describe("Settings", func() {
		Context("when the settings file exists", func() {
			var (
				settingsFileName string
			)
			BeforeEach(func() {
				settingsFileName = "/fake-settings-file-path"
				source = NewFileSettingsSource(settingsFileName, fs, logger)
			})

			Context("settings have valid format", func() {
				var (
					expectedSettings boshsettings.Settings
				)

				BeforeEach(func() {
					expectedSettings = boshsettings.Settings{
						AgentID: "fake-agent-id",
					}

					settingsJSON, err := json.Marshal(expectedSettings)
					Expect(err).ToNot(HaveOccurred())
					fs.WriteFile(settingsFileName, settingsJSON)
				})

				It("returns settings read from the file", func() {
					settings, err := source.Settings()
					Expect(err).ToNot(HaveOccurred())
					Expect(settings).To(Equal(expectedSettings))
				})
			})

			Context("settings have invalid format", func() {
				BeforeEach(func() {
					fs.WriteFileString(settingsFileName, "bad-json")
				})
				It("returns settings read from the file", func() {
					_, err := source.Settings()
					Expect(err).To(HaveOccurred())
				})
			})

		})

		Context("when the registry file does not exist", func() {
			BeforeEach(func() {
				source = NewFileSettingsSource(
					"/missing-settings-file-path",
					fs, logger)
			})

			It("returns an error", func() {
				_, err := source.Settings()
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
