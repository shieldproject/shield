package cmd_test

import (
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	cmdconf "github.com/cloudfoundry/bosh-cli/cmd/config"
	fakeconf "github.com/cloudfoundry/bosh-cli/cmd/config/configfakes"
)

var _ = Describe("SessionContextImpl", func() {
	var (
		opts    BoshOpts
		config  *fakeconf.FakeConfig
		fs      *fakesys.FakeFileSystem
		context *SessionContextImpl
	)

	BeforeEach(func() {
		opts = BoshOpts{}
		config = &fakeconf.FakeConfig{
			ResolveEnvironmentStub: func(in string) string { return in },
		}
		fs = fakesys.NewFakeFileSystem()
		context = nil
	})

	build := func() *SessionContextImpl { return NewSessionContextImpl(opts, config, fs) }

	Describe("Environment", func() {
		It("returns resolved global option if provided", func() {
			config.ResolveEnvironmentStub = func(in string) string {
				Expect(in).To(Equal("opt-alias"))
				return "resolved-url"
			}

			opts.EnvironmentOpt = "opt-alias"

			Expect(build().Environment()).To(Equal("resolved-url"))
		})

		It("returns empty string if global option is not set", func() {
			Expect(build().Environment()).To(Equal(""))
		})
	})

	Describe("Credentials", func() {
		It("defaults to config credentials for environment global option", func() {
			config.CredentialsStub = func(environment string) cmdconf.Creds {
				Expect(environment).To(Equal("opt-alias"))
				return cmdconf.Creds{Client: "config-username"}
			}

			opts.EnvironmentOpt = "opt-alias"

			Expect(build().Credentials()).To(Equal(cmdconf.Creds{Client: "config-username"}))
		})

		It("overrides uaa client and resets secret if uaa client global option is provided", func() {
			config.CredentialsReturns(cmdconf.Creds{
				Client:       "config-client",
				ClientSecret: "config-client-secret",
			})

			opts.ClientOpt = "opt-client"

			Expect(build().Credentials()).To(Equal(cmdconf.Creds{
				Client:       "opt-client",
				ClientSecret: "",
			}))
		})

		It("overrides uaa client and secret if uaa client global option is provided", func() {
			config.CredentialsReturns(cmdconf.Creds{
				Client:       "config-client",
				ClientSecret: "config-client-secret",
			})

			opts.ClientOpt = "opt-client"
			opts.ClientSecretOpt = "opt-client-secret"

			Expect(build().Credentials()).To(Equal(cmdconf.Creds{
				Client:       "opt-client",
				ClientSecret: "opt-client-secret",
			}))
		})
	})

	Describe("CACert", func() {
		BeforeEach(func() {
			opts.EnvironmentOpt = "opt-url"
		})

		It("returns global option if provided", func() {
			config.CACertReturns("config-cert")
			opts.CACertOpt = CACertArg{Content: "opt-cert"}
			Expect(build().CACert()).To(Equal("opt-cert"))
		})

		It("returns config value if global option is not set", func() {
			config.CACertReturns("config-cert")
			opts.CACertOpt = CACertArg{}
			Expect(build().CACert()).To(Equal("config-cert"))
		})

		It("returns empty string if global option or config value is not set", func() {
			Expect(build().CACert()).To(Equal(""))
		})
	})

	Describe("Deployment", func() {
		It("returns global option if provided", func() {
			opts.DeploymentOpt = "opt-dep"
			Expect(build().Deployment()).To(Equal("opt-dep"))
		})

		It("returns empty string if global option is not set", func() {
			Expect(build().Deployment()).To(Equal(""))
		})
	})
})
