package cmd_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	fakecmd "github.com/cloudfoundry/bosh-cli/cmd/cmdfakes"
	cmdconf "github.com/cloudfoundry/bosh-cli/cmd/config"
	fakecmdconf "github.com/cloudfoundry/bosh-cli/cmd/config/configfakes"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	boshtbl "github.com/cloudfoundry/bosh-cli/ui/table"
)

var _ = Describe("AliasEnvCmd", func() {
	var (
		sessions map[*fakecmdconf.FakeConfig2]*fakecmd.FakeSession
		config   *fakecmdconf.FakeConfig2
		ui       *fakeui.FakeUI
		command  AliasEnvCmd
	)

	BeforeEach(func() {
		sessions = map[*fakecmdconf.FakeConfig2]*fakecmd.FakeSession{}

		sessionFactory := func(config cmdconf.Config) Session {
			typedConfig, ok := config.(*fakecmdconf.FakeConfig2)
			if !ok {
				panic("Expected to find FakeConfig2")
			}

			for c, sess := range sessions {
				if c.Existing == typedConfig.Existing {
					return sess
				}
			}

			panic("Expected to find fake session")
		}

		config = &fakecmdconf.FakeConfig2{
			Existing: fakecmdconf.ConfigContents{
				EnvironmentURL:    "curr-environment-url",
				EnvironmentCACert: "curr-ca-cert",
			},
		}

		ui = &fakeui.FakeUI{}

		command = NewAliasEnvCmd(sessionFactory, config, ui)
	})

	Describe("Run", func() {
		var (
			opts            AliasEnvOpts
			updatedSession  *fakecmd.FakeSession
			updatedConfig   *fakecmdconf.FakeConfig2
			updatedDirector *fakedir.FakeDirector
		)

		BeforeEach(func() {
			opts = AliasEnvOpts{}

			opts.URL = "environment-url"
			opts.Args.Alias = "environment-alias"
			opts.CACert = CACertArg{Content: "environment-ca-cert"}

			updatedConfig = &fakecmdconf.FakeConfig2{
				Existing: fakecmdconf.ConfigContents{
					EnvironmentURL:    "environment-url",
					EnvironmentAlias:  "environment-alias",
					EnvironmentCACert: "environment-ca-cert",
				},
			}

			updatedDirector = &fakedir.FakeDirector{}

			updatedSession = &fakecmd.FakeSession{}
			updatedSession.DirectorReturns(updatedDirector, nil)
			updatedSession.EnvironmentReturns("environment-url")

			sessions[updatedConfig] = updatedSession
		})

		act := func() error { return command.Run(opts) }

		It("returns error if aliasing fails", func() {
			config.AliasEnvironmentErr = errors.New("fake-err")

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			Expect(config.Saved.Called).To(BeFalse())
		})

		It("saves environment and shows director info if director is reachable", func() {
			info := boshdir.Info{
				Name:    "director-name",
				UUID:    "director-uuid",
				Version: "director-version",
			}
			updatedDirector.InfoReturns(info, nil)

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(config.Saved.Called).To(BeTrue())
			Expect(config.Saved.EnvironmentURL).To(Equal("environment-url"))
			Expect(config.Saved.EnvironmentAlias).To(Equal("environment-alias"))
			Expect(config.Saved.EnvironmentCACert).To(Equal("environment-ca-cert"))

			Expect(ui.Table).To(Equal(boshtbl.Table{
				Header: []boshtbl.Header{
					boshtbl.NewHeader("Name"),
					boshtbl.NewHeader("UUID"),
					boshtbl.NewHeader("Version"),
					boshtbl.NewHeader("User"),
				},
				Rows: [][]boshtbl.Value{
					{
						boshtbl.NewValueString("director-name"),
						boshtbl.NewValueString("director-uuid"),
						boshtbl.NewValueString("director-version"),
						boshtbl.NewValueString("(not logged in)"),
					},
				},
				Transpose: true,
			}))
		})

		It("returns an error and does not save environment if director is not reachable", func() {
			updatedDirector.InfoReturns(boshdir.Info{}, errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			Expect(config.Saved.Called).To(BeFalse())
		})
	})
})
