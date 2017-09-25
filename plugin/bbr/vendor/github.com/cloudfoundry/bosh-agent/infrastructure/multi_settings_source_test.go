package infrastructure_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/infrastructure"
	fakeinf "github.com/cloudfoundry/bosh-agent/infrastructure/fakes"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
)

var _ = Describe("MultiSettingsSource", func() {
	var (
		source boshsettings.Source
	)

	Context("when there are no sources", func() {
		It("returns an error when there are no sources", func() {
			_, err := NewMultiSettingsSource([]boshsettings.Source{}...)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("MultiSettingsSource requires to have at least one source"))
		})
	})

	Context("when there is at least one source", func() {
		var (
			source1 fakeinf.FakeSettingsSource
			source2 fakeinf.FakeSettingsSource
		)

		BeforeEach(func() {
			source1 = fakeinf.FakeSettingsSource{
				PublicKey:    "fake-public-key-1",
				PublicKeyErr: errors.New("fake-public-key-err-1"),

				SettingsValue: boshsettings.Settings{AgentID: "fake-settings-1"},
				SettingsErr:   errors.New("fake-settings-err-1"),
			}

			source2 = fakeinf.FakeSettingsSource{
				PublicKey:    "fake-public-key-2",
				PublicKeyErr: errors.New("fake-public-key-err-2"),

				SettingsValue: boshsettings.Settings{AgentID: "fake-settings-2"},
				SettingsErr:   errors.New("fake-settings-err-2"),
			}
		})

		JustBeforeEach(func() {
			var err error
			source, err = NewMultiSettingsSource(source1, source2)
			Expect(err).ToNot(HaveOccurred())
		})

		Describe("PublicSSHKeyForUsername", func() {
			Context("when the first source returns public key", func() {
				BeforeEach(func() {
					source1.PublicKeyErr = nil
				})

				It("returns public key and public key error from the first source", func() {
					publicKey, err := source.PublicSSHKeyForUsername("fake-username")
					Expect(err).ToNot(HaveOccurred())
					Expect(publicKey).To(Equal("fake-public-key-1"))
				})
			})

			Context("when the second source returns public key", func() {
				BeforeEach(func() {
					source2.PublicKeyErr = nil
				})

				It("returns public key from the second source", func() {
					publicKey, err := source.PublicSSHKeyForUsername("fake-username")
					Expect(err).ToNot(HaveOccurred())
					Expect(publicKey).To(Equal("fake-public-key-2"))
				})
			})

			Context("when both sources fail to get ssh key", func() {
				It("returns error from the second source", func() {
					_, err := source.PublicSSHKeyForUsername("fake-username")
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-public-key-err-2"))
				})
			})
		})

		Describe("Settings", func() {
			Context("when the first source returns settings", func() {
				BeforeEach(func() {
					source1.SettingsErr = nil
				})

				It("returns settings from the first source", func() {
					settings, err := source.Settings()
					Expect(err).ToNot(HaveOccurred())
					Expect(settings).To(Equal(boshsettings.Settings{AgentID: "fake-settings-1"}))
				})
			})

			Context("when both sources do not have settings", func() {
				It("returns error from the second source", func() {
					_, err := source.Settings()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-settings-err-2"))
				})
			})

			Context("when the second source returns settings", func() {
				BeforeEach(func() {
					source2.SettingsErr = nil
				})

				It("returns settings from the second source", func() {
					settings, err := source.Settings()
					Expect(err).ToNot(HaveOccurred())
					Expect(settings).To(Equal(boshsettings.Settings{AgentID: "fake-settings-2"}))
				})
			})
		})
	})
})
