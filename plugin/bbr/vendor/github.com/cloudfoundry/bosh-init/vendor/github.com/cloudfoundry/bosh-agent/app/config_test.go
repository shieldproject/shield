package app

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshinf "github.com/cloudfoundry/bosh-agent/infrastructure"
	boshplatform "github.com/cloudfoundry/bosh-agent/platform"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("LoadConfigFromPath", func() {
	var (
		fs *fakesys.FakeFileSystem
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
	})

	It("returns populates config", func() {
		fs.WriteFileString("/fake-config.conf", `{
			"Platform": {
				"Linux": {
					"UseDefaultTmpDir": true,
					"UsePreformattedPersistentDisk": true,
					"BindMountPersistentDisk": true,
					"SkipDiskSetup": true,
					"DevicePathResolutionType": "virtio"
				}
			},
			"Infrastructure": {
			  "Settings": {
				  "Sources": [
				  	{
					  	"Type": "HTTP",
					  	"URI": "http://fake-uri"
					  },
					  {
					  	"Type": "ConfigDrive",
					  	"DiskPaths": ["/fake-disk-path1", "/fake-disk-path2"],
					  	"MetaDataPath": "/fake-metadata-path",
					  	"UserDataPath": "/fake-userdata-path",
					  	"SettingsPath": "/fake-settings-path"
					  },
					  {
					  	"Type": "File",
					  	"MetaDataPath": "/fake-metadata-path",
					  	"UserDataPath": "/fake-userdata-path",
					  	"SettingsPath": "/fake-settings-path"
					  },
					  {
					  	"Type": "CDROM",
					  	"FileName": "/fake-file-name"
					  },
					  {
						"Type": "InstanceMetadata",
						"URI": "/fake-uri",
						"Headers": {"fake": "headers"},
						"SettingsPath": "/fake-settings-path"
					  }
				  ],
				  "UseServerName": true,
				  "UseRegistry": true
				}
			}
		}`)

		config, err := LoadConfigFromPath(fs, "/fake-config.conf")
		Expect(err).ToNot(HaveOccurred())
		Expect(config).To(Equal(Config{
			Platform: boshplatform.Options{
				Linux: boshplatform.LinuxOptions{
					UseDefaultTmpDir:              true,
					UsePreformattedPersistentDisk: true,
					BindMountPersistentDisk:       true,
					SkipDiskSetup:                 true,
					DevicePathResolutionType:      "virtio",
				},
			},
			Infrastructure: boshinf.Options{
				Settings: boshinf.SettingsOptions{
					Sources: []boshinf.SourceOptions{
						boshinf.HTTPSourceOptions{
							URI: "http://fake-uri",
						},
						boshinf.ConfigDriveSourceOptions{
							DiskPaths:    []string{"/fake-disk-path1", "/fake-disk-path2"},
							MetaDataPath: "/fake-metadata-path",
							UserDataPath: "/fake-userdata-path",
							SettingsPath: "/fake-settings-path",
						},
						boshinf.FileSourceOptions{
							MetaDataPath: "/fake-metadata-path",
							UserDataPath: "/fake-userdata-path",
							SettingsPath: "/fake-settings-path",
						},
						boshinf.CDROMSourceOptions{
							FileName: "/fake-file-name",
						},
						boshinf.InstanceMetadataSourceOptions{
							URI:          "/fake-uri",
							Headers:      map[string]string{"fake": "headers"},
							SettingsPath: "/fake-settings-path",
						},
					},
					UseServerName: true,
					UseRegistry:   true,
				},
			},
		}))
	})

	It("returns empty config if path is empty", func() {
		config, err := LoadConfigFromPath(fs, "")
		Expect(err).ToNot(HaveOccurred())
		Expect(config).To(Equal(Config{}))
	})

	It("returns error if file is not found", func() {
		_, err := LoadConfigFromPath(fs, "/something_not_exist")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not found"))
	})

	It("returns error if file cannot be parsed", func() {
		fs.WriteFileString("/fake-config.conf", `fake-invalid-json`)

		_, err := LoadConfigFromPath(fs, "/fake-config.conf")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid character"))
	})

	It("returns an error when the source options type is unknown", func() {
		fs.WriteFileString("/fake-config.conf", `{
			"Infrastructure": {
			  "Settings": {
				  "Sources": [{ "Type": "fake-type" }]
				}
			}
		}`)

		_, err := LoadConfigFromPath(fs, "/fake-config.conf")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Unknown source type 'fake-type'"))
	})

	It("returns an error when the source options do not have type", func() {
		fs.WriteFileString("/fake-config.conf", `{
			"Infrastructure": {
			  "Settings": {
				  "Sources": [{ }]
				}
			}
		}`)

		_, err := LoadConfigFromPath(fs, "/fake-config.conf")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Missing source type"))
	})

	It("returns empty settings sources if no sources are defined", func() {
		fs.WriteFileString("/fake-config.conf", `{
			"Infrastructure": {
			  "Settings": {
				  "Sources": []
				}
			}
		}`)

		config, err := LoadConfigFromPath(fs, "/fake-config.conf")
		Expect(err).ToNot(HaveOccurred())
		Expect(config).To(Equal(Config{
			Infrastructure: boshinf.Options{
				Settings: boshinf.SettingsOptions{},
			},
		}))
	})

	It("returns an error when the source options do not have type", func() {
		fs.WriteFileString("/fake-config.conf", `{
			"Infrastructure": {
			  "Settings": { "Sources": 1 }
			}
		}`)

		_, err := LoadConfigFromPath(fs, "/fake-config.conf")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Unmarshalling sources"))
	})

	It("returns errors if failed to decode HTTP source options", func() {
		fs.WriteFileString("/fake-config.conf", `{
			"Infrastructure": {
			  "Settings": {
				  "Sources": [{
				  	"Type": "HTTP",
				  	"URI": 1
					}]
				}
			}
		}`)

		_, err := LoadConfigFromPath(fs, "/fake-config.conf")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Unmarshalling source type 'HTTP'"))
	})

	It("returns errors if failed to decode InstanceMetadata source options", func() {
		fs.WriteFileString("/fake-config.conf", `{
			"Infrastructure": {
			  "Settings": {
				  "Sources": [{
				  	"Type": "InstanceMetadata",
					"URI": 1
					}]
				}
			}
		}`)

		_, err := LoadConfigFromPath(fs, "/fake-config.conf")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Unmarshalling source type 'InstanceMetadata'"))
	})

	It("returns errors if failed to decode ConfigDrive source options", func() {
		fs.WriteFileString("/fake-config.conf", `{
			"Infrastructure": {
			  "Settings": {
				  "Sources": [{
				  	"Type": "ConfigDrive",
				  	"DiskPaths": 1
				  }]
				}
			}
		}`)

		_, err := LoadConfigFromPath(fs, "/fake-config.conf")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Unmarshalling source type 'ConfigDrive'"))
	})

	It("returns errors if failed to decode File source options", func() {
		fs.WriteFileString("/fake-config.conf", `{
			"Infrastructure": {
			  "Settings": {
				  "Sources": [{
				  	"Type": "File",
				  	"MetaDataPath": 1
				  }]
				}
			}
		}`)

		_, err := LoadConfigFromPath(fs, "/fake-config.conf")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Unmarshalling source type 'File'"))
	})

	It("returns errors if failed to decode CDROM source options", func() {
		fs.WriteFileString("/fake-config.conf", `{
			"Infrastructure": {
			  "Settings": {
				  "Sources": [{
				  	"Type": "CDROM",
				  	"FileName": 1
				  }]
				}
			}
		}`)

		_, err := LoadConfigFromPath(fs, "/fake-config.conf")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Unmarshalling source type 'CDROM'"))
	})
})
