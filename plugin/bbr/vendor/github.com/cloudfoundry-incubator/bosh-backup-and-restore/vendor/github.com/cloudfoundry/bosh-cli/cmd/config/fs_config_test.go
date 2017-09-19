package config_test

import (
	"errors"

	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd/config"
)

var _ = Describe("NewFSConfigFromPath", func() {
	It("expands config path", func() {
		fs := fakesys.NewFakeFileSystem()
		fs.ExpandPathExpanded = "/expanded_config"

		config, err := NewFSConfigFromPath("/config", fs)
		Expect(err).ToNot(HaveOccurred())

		err = config.Save()
		Expect(err).ToNot(HaveOccurred())
		Expect(fs.FileExists("/expanded_config")).To(BeTrue())
	})

	It("returns empty config if file does not exist", func() {
		fs := fakesys.NewFakeFileSystem()

		config, err := NewFSConfigFromPath("/no_config", fs)
		Expect(err).ToNot(HaveOccurred())
		Expect(config.Environments()).To(BeEmpty())
	})

	It("returns error if expanding path fails", func() {
		fs := fakesys.NewFakeFileSystem()
		fs.ExpandPathErr = errors.New("fake-err")

		_, err := NewFSConfigFromPath("/config", fs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("fake-err"))
	})

	It("returns error if reading file fails", func() {
		fs := fakesys.NewFakeFileSystem()
		fs.WriteFileString("/config", "")
		fs.ReadFileError = errors.New("fake-err")

		_, err := NewFSConfigFromPath("/config", fs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("fake-err"))
	})

	It("returns error if config file cannot be unmarshaled", func() {
		fs := fakesys.NewFakeFileSystem()
		fs.WriteFileString("/config", "-")

		_, err := NewFSConfigFromPath("/config", fs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("line 1"))
	})
})

var _ = Describe("FSConfig", func() {
	var (
		fs     *fakesys.FakeFileSystem
		config Config
	)

	readConfig := func() FSConfig {
		config, err := NewFSConfigFromPath("/dir/sub-dir/config", fs)
		Expect(err).ToNot(HaveOccurred())

		return config
	}

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		config = readConfig()
	})

	Describe("Environments", func() {
		It("returns empty list if there are no remembered environments", func() {
			Expect(config.Environments()).To(BeEmpty())
		})

		It("returns list of previously remembered environments", func() {
			updatedConfig, err := config.AliasEnvironment("url1", "alias1", "")
			Expect(err).ToNot(HaveOccurred())

			updatedConfig, err = updatedConfig.AliasEnvironment("url2", "alias2", "")
			Expect(err).ToNot(HaveOccurred())

			updatedConfig, err = updatedConfig.AliasEnvironment("url3", "alias3", "")
			Expect(err).ToNot(HaveOccurred())

			Expect(updatedConfig.Environments()).To(Equal([]Environment{
				Environment{URL: "url1", Alias: "alias1"},
				Environment{URL: "url2", Alias: "alias2"},
				Environment{URL: "url3", Alias: "alias3"},
			}))

			err = updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(reloadedConfig.Environments()).To(Equal([]Environment{
				Environment{URL: "url1", Alias: "alias1"},
				Environment{URL: "url2", Alias: "alias2"},
				Environment{URL: "url3", Alias: "alias3"},
			}))
		})
	})

	Describe("AliasEnvironment/CACert", func() {
		It("returns empty if file does not exist", func() {
			Expect(config.CACert("url")).To(Equal(""))
		})

		It("returns error if url is empty", func() {
			_, err := config.AliasEnvironment("", "alias", "ca-cert")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected non-empty environment URL"))
		})

		It("overwrites when an entry with the given url is already present", func() {
			config, err := config.AliasEnvironment("url", "alias", "")
			Expect(err).ToNot(HaveOccurred())

			config, err = config.AliasEnvironment("url", "different-alias", "")
			Expect(err).ToNot(HaveOccurred())

			Expect(config.Environments()).To(HaveLen(1))
			Expect(config.Environments()[0].Alias).To(Equal("different-alias"))
			Expect(config.Environments()[0].URL).To(Equal("url"))

		})

		It("overwrites whent an entry with the given alias is already present", func() {
			config, err := config.AliasEnvironment("url", "alias", "ca")
			Expect(err).ToNot(HaveOccurred())

			config, err = config.AliasEnvironment("different-url", "alias", "diff-ca")
			Expect(err).ToNot(HaveOccurred())

			Expect(config.Environments()).To(HaveLen(1))
			Expect(config.Environments()[0].Alias).To(Equal("alias"))
			Expect(config.Environments()[0].URL).To(Equal("different-url"))
		})

		It("returns error if alias is empty", func() {
			_, err := config.AliasEnvironment("url", "", "ca-cert")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected non-empty environment alias"))
		})

		It("returns saved url", func() {
			updatedConfig, err := config.AliasEnvironment("url", "alias", "")
			Expect(err).ToNot(HaveOccurred())

			Expect(updatedConfig.ResolveEnvironment("url")).To(Equal("url"))

			err = updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(reloadedConfig.ResolveEnvironment("url")).To(Equal("url"))
		})

		It("returns saved url based on the alias", func() {
			updatedConfig, err := config.AliasEnvironment("url1", "alias1", "")
			Expect(err).ToNot(HaveOccurred())

			updatedConfig, err = updatedConfig.AliasEnvironment("url2", "alias2", "")
			Expect(err).ToNot(HaveOccurred())

			Expect(updatedConfig.ResolveEnvironment("alias1")).To(Equal("url1"))

			err = updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()

			Expect(reloadedConfig.ResolveEnvironment("alias1")).To(Equal("url1"))
		})

		It("saves empty CA certificate", func() {
			updatedConfig, err := config.AliasEnvironment("url", "alias", "")
			Expect(err).ToNot(HaveOccurred())

			Expect(updatedConfig.ResolveEnvironment("url")).To(Equal("url"))
			Expect(updatedConfig.CACert("url")).To(Equal(""))

			err = updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()

			Expect(reloadedConfig.CACert("url")).To(Equal(""))
		})

		It("saves non-empty CA certificate and then unsets it", func() {
			updatedConfig, err := config.AliasEnvironment("url", "alias", "ca-cert")
			Expect(err).ToNot(HaveOccurred())

			Expect(updatedConfig.ResolveEnvironment("url")).To(Equal("url"))
			Expect(updatedConfig.CACert("url")).To(Equal("ca-cert"))

			err = updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(reloadedConfig.CACert("url")).To(Equal("ca-cert"))

			updatedConfig, err = reloadedConfig.AliasEnvironment("url", "alias", "")
			Expect(err).ToNot(HaveOccurred())

			Expect(updatedConfig.ResolveEnvironment("url")).To(Equal("url"))
			Expect(updatedConfig.CACert("url")).To(Equal(""))

			err = updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig = readConfig()
			Expect(reloadedConfig.CACert("url")).To(Equal(""))
		})

		It("returns CA cert for alias", func() {
			updatedConfig, err := config.AliasEnvironment("url", "alias", "ca-cert")
			Expect(err).ToNot(HaveOccurred())

			Expect(updatedConfig.ResolveEnvironment("url")).To(Equal("url"))
			Expect(updatedConfig.CACert("alias")).To(Equal("ca-cert"))

			err = updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(reloadedConfig.CACert("alias")).To(Equal("ca-cert"))

			updatedConfig, err = reloadedConfig.AliasEnvironment("url", "alias", "")
			Expect(err).ToNot(HaveOccurred())

			Expect(updatedConfig.ResolveEnvironment("url")).To(Equal("url"))
			Expect(updatedConfig.CACert("alias")).To(Equal(""))

			err = updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig = readConfig()
			Expect(reloadedConfig.CACert("alias")).To(Equal(""))
		})
	})

	Describe("ResolveEnvironment", func() {
		It("returns url if it's a known url", func() {
			updatedConfig, err := config.AliasEnvironment("url", "alias", "")
			Expect(err).ToNot(HaveOccurred())

			Expect(updatedConfig.ResolveEnvironment("url")).To(Equal("url"))
		})

		It("returns aliased url", func() {
			updatedConfig, err := config.AliasEnvironment("url", "alias", "")
			Expect(err).ToNot(HaveOccurred())

			updatedConfig, err = updatedConfig.AliasEnvironment("url2", "alias2", "")
			Expect(err).ToNot(HaveOccurred())

			Expect(updatedConfig.ResolveEnvironment("alias")).To(Equal("url"))
		})

		It("returns input when it's not an alias or url", func() {
			Expect(config.ResolveEnvironment("unknown")).To(Equal("unknown"))
		})

		It("returns empty when alias or url is empty", func() {
			Expect(config.ResolveEnvironment("")).To(Equal(""))
		})
	})

	Describe("SetCredentials/Credentials/UnsetCredentials", func() {
		It("returns empty if environment is not found", func() {
			Expect(config.Credentials("url")).To(Equal(Creds{}))
		})

		It("returns empty if environment is found but creds are not set", func() {
			updatedConfig, err := config.AliasEnvironment("url", "alias", "")
			Expect(err).ToNot(HaveOccurred())

			Expect(updatedConfig.Credentials("url")).To(Equal(Creds{}))

			err = updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(reloadedConfig.Credentials("url")).To(Equal(Creds{}))

			updatedConfig = reloadedConfig.UnsetCredentials("url")
			Expect(updatedConfig.Credentials("url")).To(Equal(Creds{}))

			err = updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig = readConfig()
			Expect(reloadedConfig.Credentials("url")).To(Equal(Creds{}))
		})

		It("returns creds with username/password if environment is found and basic creds are set", func() {
			updatedConfig, err := config.AliasEnvironment("url", "alias", "")
			Expect(err).ToNot(HaveOccurred())

			updatedConfig = config.SetCredentials("url", Creds{Client: "user", ClientSecret: "pass"})
			Expect(updatedConfig.Credentials("url")).To(Equal(Creds{Client: "user", ClientSecret: "pass"}))

			err = updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(reloadedConfig.Credentials("url")).To(Equal(Creds{Client: "user", ClientSecret: "pass"}))

			updatedConfig = reloadedConfig.UnsetCredentials("url")
			Expect(updatedConfig.Credentials("url")).To(Equal(Creds{}))

			err = updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig = readConfig()
			Expect(reloadedConfig.Credentials("url")).To(Equal(Creds{}))
		})

		It("returns creds with token if environment is found and token is set", func() {
			updatedConfig, err := config.AliasEnvironment("url", "alias", "")
			Expect(err).ToNot(HaveOccurred())

			updatedConfig = config.SetCredentials("url", Creds{RefreshToken: "token"})
			Expect(updatedConfig.Credentials("url")).To(Equal(Creds{RefreshToken: "token"}))

			err = updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(reloadedConfig.Credentials("url")).To(Equal(Creds{RefreshToken: "token"}))

			updatedConfig = reloadedConfig.UnsetCredentials("url")
			Expect(updatedConfig.Credentials("url")).To(Equal(Creds{}))

			err = updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig = readConfig()
			Expect(reloadedConfig.Credentials("url")).To(Equal(Creds{}))
		})

		It("returns creds for alias if environment is found and token is set", func() {
			updatedConfig, err := config.AliasEnvironment("url", "alias", "")
			Expect(err).ToNot(HaveOccurred())

			updatedConfig = config.SetCredentials("alias", Creds{RefreshToken: "token"})
			Expect(updatedConfig.Credentials("alias")).To(Equal(Creds{RefreshToken: "token"}))

			err = updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(reloadedConfig.Credentials("alias")).To(Equal(Creds{RefreshToken: "token"}))

			updatedConfig = reloadedConfig.UnsetCredentials("alias")
			Expect(updatedConfig.Credentials("alias")).To(Equal(Creds{}))

			err = updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig = readConfig()
			Expect(reloadedConfig.Credentials("alias")).To(Equal(Creds{}))
		})

		It("does not update existing config when creds are set", func() {
			updatedConfig, err := config.AliasEnvironment("url", "alias", "")
			Expect(err).ToNot(HaveOccurred())

			updatedConfig = config.SetCredentials("url", Creds{Client: "user"})
			Expect(updatedConfig.Credentials("url")).To(Equal(Creds{Client: "user"}))

			err = updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			Expect(config.Credentials("url")).To(Equal(Creds{}))
		})
	})

	Describe("Save", func() {
		It("returns error if writing file fails", func() {
			fs.WriteFileError = errors.New("fake-err")

			config := readConfig()
			err := config.Save()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
