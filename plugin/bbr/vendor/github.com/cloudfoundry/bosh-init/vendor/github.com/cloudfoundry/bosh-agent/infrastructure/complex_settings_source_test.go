package infrastructure_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/infrastructure"
	fakeinf "github.com/cloudfoundry/bosh-agent/infrastructure/fakes"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("ComplexSettingsSource", func() {
	var (
		metadataService  *fakeinf.FakeMetadataService
		registryProvider *fakeinf.FakeRegistryProvider
		source           ComplexSettingsSource
	)

	BeforeEach(func() {
		metadataService = &fakeinf.FakeMetadataService{}
		registryProvider = &fakeinf.FakeRegistryProvider{}
		logger := boshlog.NewLogger(boshlog.LevelNone)
		source = NewComplexSettingsSource(metadataService, registryProvider, logger)
	})

	Describe("PublicSSHKeyForUsername", func() {
		It("returns an empty string", func() {
			metadataService.PublicKey = "public-key"

			publicKey, err := source.PublicSSHKeyForUsername("fake-username")
			Expect(err).ToNot(HaveOccurred())
			Expect(publicKey).To(Equal("public-key"))
		})

		It("returns an error if string", func() {
			metadataService.GetPublicKeyErr = errors.New("fake-public-key-error")

			_, err := source.PublicSSHKeyForUsername("fake-username")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-public-key-error"))
		})
	})

	Describe("Settings", func() {
		It("returns settings read from the registry", func() {
			registryProvider.GetRegistryRegistry = &fakeinf.FakeRegistry{
				Settings: boshsettings.Settings{
					AgentID: "fake-agent-id",
				},
			}

			settings, err := source.Settings()
			Expect(err).ToNot(HaveOccurred())
			Expect(settings.AgentID).To(Equal("fake-agent-id"))
		})

		It("returns an error if cannot get registry", func() {
			registryProvider.GetRegistryErr = errors.New("fake-get-registry-error")

			_, err := source.Settings()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-get-registry-error"))
		})

		It("returns an error if registry returns an error while getting settings", func() {
			registryProvider.GetRegistryRegistry = &fakeinf.FakeRegistry{
				GetSettingsErr: errors.New("fake-get-settings-error"),
			}

			_, err := source.Settings()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-get-settings-error"))
		})
	})
})
