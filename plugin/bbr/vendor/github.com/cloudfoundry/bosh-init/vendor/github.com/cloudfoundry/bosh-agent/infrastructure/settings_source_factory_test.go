package infrastructure_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/infrastructure"
	fakeplat "github.com/cloudfoundry/bosh-agent/platform/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("SettingsSourceFactory", func() {
	Describe("New", func() {
		var (
			options  SettingsOptions
			platform *fakeplat.FakePlatform
			logger   boshlog.Logger
			factory  SettingsSourceFactory
		)

		BeforeEach(func() {
			options = SettingsOptions{}
			platform = fakeplat.NewFakePlatform()
			logger = boshlog.NewLogger(boshlog.LevelNone)
		})

		JustBeforeEach(func() {
			factory = NewSettingsSourceFactory(options, platform, logger)
		})

		Context("when UseRegistry is set to true", func() {
			BeforeEach(func() {
				options.UseRegistry = true
			})

			ItConfiguresSourcesToUseRegistry := func(useServerName bool) {
				Context("when using HTTP source", func() {
					BeforeEach(func() {
						options.Sources = []SourceOptions{
							HTTPSourceOptions{URI: "http://fake-url"},
						}
					})

					It("returns a settings source that uses HTTP to fetch settings", func() {
						resolver := NewRegistryEndpointResolver(NewDigDNSResolver(platform.GetRunner(), logger))
						httpMetadataService := NewHTTPMetadataService("http://fake-url", nil, "", "", "", resolver, platform, logger)
						multiSourceMetadataService := NewMultiSourceMetadataService(httpMetadataService)
						registryProvider := NewRegistryProvider(multiSourceMetadataService, platform, useServerName, platform.GetFs(), logger)
						httpSettingsSource := NewComplexSettingsSource(multiSourceMetadataService, registryProvider, logger)

						settingsSource, err := factory.New()
						Expect(err).ToNot(HaveOccurred())
						Expect(settingsSource).To(Equal(httpSettingsSource))
					})
				})

				Context("when using ConfigDrive source", func() {
					BeforeEach(func() {
						options.Sources = []SourceOptions{
							ConfigDriveSourceOptions{
								DiskPaths: []string{"/fake-disk-path"},

								MetaDataPath: "fake-meta-data-path",
								UserDataPath: "fake-user-data-path",

								SettingsPath: "fake-settings-path",
							},
						}
					})

					It("returns a settings source that uses config drive to fetch settings", func() {
						resolver := NewRegistryEndpointResolver(NewDigDNSResolver(platform.GetRunner(), logger))
						configDriveMetadataService := NewConfigDriveMetadataService(
							resolver,
							platform,
							[]string{"/fake-disk-path"},
							"fake-meta-data-path",
							"fake-user-data-path",
							logger,
						)
						multiSourceMetadataService := NewMultiSourceMetadataService(configDriveMetadataService)
						registryProvider := NewRegistryProvider(multiSourceMetadataService, platform, useServerName, platform.GetFs(), logger)
						configDriveSettingsSource := NewComplexSettingsSource(multiSourceMetadataService, registryProvider, logger)

						settingsSource, err := factory.New()
						Expect(err).ToNot(HaveOccurred())
						Expect(settingsSource).To(Equal(configDriveSettingsSource))
					})
				})

				Context("when using File source", func() {
					BeforeEach(func() {
						options.Sources = []SourceOptions{
							FileSourceOptions{
								MetaDataPath: "fake-meta-data-path",
								UserDataPath: "fake-user-data-path",

								SettingsPath: "fake-settings-path",
							},
						}
					})

					It("returns a settings source that uses file to fetch settings", func() {
						fileMetadataService := NewFileMetadataService(
							"fake-meta-data-path",
							"fake-user-data-path",
							"fake-settings-path",
							platform.GetFs(),
							logger,
						)
						multiSourceMetadataService := NewMultiSourceMetadataService(fileMetadataService)
						registryProvider := NewRegistryProvider(multiSourceMetadataService, platform, useServerName, platform.GetFs(), logger)
						fileSettingsSource := NewComplexSettingsSource(multiSourceMetadataService, registryProvider, logger)

						settingsSource, err := factory.New()
						Expect(err).ToNot(HaveOccurred())
						Expect(settingsSource).To(Equal(fileSettingsSource))
					})
				})

				Context("when using CDROM source", func() {
					BeforeEach(func() {
						options.Sources = []SourceOptions{
							CDROMSourceOptions{
								FileName: "fake-file-name",
							},
						}
					})

					It("returns error because it is not supported", func() {
						_, err := factory.New()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("CDROM source is not supported when registry is used"))
					})
				})
			}

			Context("when UseServerName is set to true", func() {
				BeforeEach(func() { options.UseServerName = true })
				ItConfiguresSourcesToUseRegistry(true)
			})

			Context("when UseServerName is set to false", func() {
				BeforeEach(func() { options.UseServerName = false })
				ItConfiguresSourcesToUseRegistry(false)
			})
		})

		Context("when UseRegistry is set to false", func() {
			Context("when using HTTP source", func() {
				BeforeEach(func() {
					options = SettingsOptions{
						Sources: []SourceOptions{
							HTTPSourceOptions{},
						},
					}
				})

				It("returns error because it is not supported", func() {
					_, err := factory.New()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("HTTP source is not supported without registry"))
				})
			})

			Context("when using ConfigDrive source", func() {
				BeforeEach(func() {
					options = SettingsOptions{
						Sources: []SourceOptions{
							ConfigDriveSourceOptions{
								DiskPaths: []string{"/fake-disk-path"},

								MetaDataPath: "fake-meta-data-path",

								SettingsPath: "fake-settings-path",
							},
						},
					}
				})

				It("returns a settings source that uses config drive to fetch settings", func() {
					configDriveSettingsSource := NewConfigDriveSettingsSource(
						[]string{"/fake-disk-path"},
						"fake-meta-data-path",
						"fake-settings-path",
						platform,
						logger,
					)

					multiSettingsSource, err := NewMultiSettingsSource(configDriveSettingsSource)
					Expect(err).ToNot(HaveOccurred())

					settingsSource, err := factory.New()
					Expect(err).ToNot(HaveOccurred())
					Expect(settingsSource).To(Equal(multiSettingsSource))
				})
			})

			Context("when using File source", func() {
				BeforeEach(func() {
					options = SettingsOptions{
						Sources: []SourceOptions{
							FileSourceOptions{},
						},
					}
				})

				It("returns error because it is not supported", func() {
					_, err := factory.New()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("File source is not supported without registry"))
				})
			})

			Context("when using CDROM source", func() {
				BeforeEach(func() {
					options = SettingsOptions{
						Sources: []SourceOptions{
							CDROMSourceOptions{
								FileName: "fake-file-name",
							},
						},
					}
				})

				It("returns a settings source that uses the CDROM to fetch settings", func() {
					cdromSettingsSource := NewCDROMSettingsSource(
						"fake-file-name",
						platform,
						logger,
					)

					multiSettingsSource, err := NewMultiSettingsSource(cdromSettingsSource)
					Expect(err).ToNot(HaveOccurred())

					settingsSource, err := factory.New()
					Expect(err).ToNot(HaveOccurred())
					Expect(settingsSource).To(Equal(multiSettingsSource))
				})
			})
		})
	})
})
