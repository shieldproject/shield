package cmd_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	fakecmd "github.com/cloudfoundry/bosh-cli/cmd/cmdfakes"
	cmdconf "github.com/cloudfoundry/bosh-cli/cmd/config"
	fakecmdconf "github.com/cloudfoundry/bosh-cli/cmd/config/configfakes"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	boshtbl "github.com/cloudfoundry/bosh-cli/ui/table"
)

var _ = Describe("DeploymentCmd", func() {
	var (
		sessions map[cmdconf.Config]*fakecmd.FakeSession
		config   *fakecmdconf.FakeConfig
		ui       *fakeui.FakeUI
		command  DeploymentCmd
	)

	BeforeEach(func() {
		sessions = map[cmdconf.Config]*fakecmd.FakeSession{}
		sessionFactory := func(config cmdconf.Config) Session {
			return sessions[config]
		}
		config = &fakecmdconf.FakeConfig{}
		ui = &fakeui.FakeUI{}
		command = NewDeploymentCmd(sessionFactory, config, ui)
	})

	Describe("Run", func() {
		var (
			initialSession *fakecmd.FakeSession
			deployment     *fakedir.FakeDeployment
		)

		BeforeEach(func() {
			initialSession = &fakecmd.FakeSession{}
			sessions[config] = initialSession

			initialSession.EnvironmentReturns("environment-url")
		})

		act := func() error { return command.Run() }

		It("shows current deployment name when director finds deployment", func() {
			deployment = &fakedir.FakeDeployment{
				NameStub: func() string { return "deployment-name" },
			}
			initialSession.DeploymentReturns(deployment, nil)

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(ui.Table).To(Equal(boshtbl.Table{
				Content: "deployments",

				Header: []boshtbl.Header{
					boshtbl.NewHeader("Name"),
					boshtbl.NewHeader("Release(s)"),
					boshtbl.NewHeader("Stemcell(s)"),
					boshtbl.NewHeader("Team(s)"),
					boshtbl.NewHeader("Cloud Config"),
				},

				SortBy: []boshtbl.ColumnSort{
					{Column: 0, Asc: true},
				},

				Rows: [][]boshtbl.Value{
					{
						boshtbl.NewValueString("deployment-name"),
						boshtbl.NewValueStrings(nil),
						boshtbl.NewValueStrings(nil),
						boshtbl.NewValueStrings(nil),
						boshtbl.NewValueString(""),
					},
				},
			}))
		})

		It("returns an error when director does not find deployment", func() {
			initialSession.DeploymentReturns(nil, errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			Expect(ui.Tables).To(BeEmpty())
		})
	})
})
