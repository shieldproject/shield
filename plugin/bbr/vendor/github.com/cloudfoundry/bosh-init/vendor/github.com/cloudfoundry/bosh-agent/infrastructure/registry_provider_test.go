package infrastructure_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/infrastructure"
	fakeinf "github.com/cloudfoundry/bosh-agent/infrastructure/fakes"
	fakeplat "github.com/cloudfoundry/bosh-agent/platform/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("RegistryProvider", func() {
	var (
		metadataService  *fakeinf.FakeMetadataService
		platform         *fakeplat.FakePlatform
		useServerName    bool
		fs               *fakesys.FakeFileSystem
		registryProvider RegistryProvider
	)

	BeforeEach(func() {
		metadataService = &fakeinf.FakeMetadataService{}
		platform = &fakeplat.FakePlatform{}
		useServerName = false
		fs = fakesys.NewFakeFileSystem()
	})

	JustBeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		registryProvider = NewRegistryProvider(metadataService, platform, useServerName, fs, logger)
	})

	Describe("GetRegistry", func() {
		Context("when metadata service returns registry http endpoint", func() {
			BeforeEach(func() {
				metadataService.RegistryEndpoint = "http://registry-endpoint"
			})

			Context("when registry is configured to not use server name as id", func() {
				BeforeEach(func() { useServerName = false })

				It("returns an http registry that does not use server name as id", func() {
					registry, err := registryProvider.GetRegistry()
					Expect(err).ToNot(HaveOccurred())
					Expect(registry).To(Equal(NewHTTPRegistry(metadataService, platform, false)))
				})
			})

			Context("when registry is configured to use server name as id", func() {
				BeforeEach(func() { useServerName = true })

				It("returns an http registry that uses server name as id", func() {
					registry, err := registryProvider.GetRegistry()
					Expect(err).ToNot(HaveOccurred())
					Expect(registry).To(Equal(NewHTTPRegistry(metadataService, platform, true)))
				})
			})
		})

		Context("when metadata service returns registry file endpoint", func() {
			BeforeEach(func() {
				metadataService.RegistryEndpoint = "/tmp/registry-endpoint"
			})

			It("returns a file registry", func() {
				registry, err := registryProvider.GetRegistry()
				Expect(err).ToNot(HaveOccurred())
				Expect(registry).To(Equal(NewFileRegistry("/tmp/registry-endpoint", fs)))
			})
		})

		Context("when metadata service returns an error", func() {
			BeforeEach(func() {
				metadataService.GetRegistryEndpointErr = errors.New("fake-get-registry-endpoint-error")
			})

			It("returns error", func() {
				_, err := registryProvider.GetRegistry()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-get-registry-endpoint-error"))
			})
		})
	})
})
