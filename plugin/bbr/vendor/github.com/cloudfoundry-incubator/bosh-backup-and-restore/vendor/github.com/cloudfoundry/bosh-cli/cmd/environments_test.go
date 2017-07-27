package cmd_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	cmdconf "github.com/cloudfoundry/bosh-cli/cmd/config"
	fakecmdconf "github.com/cloudfoundry/bosh-cli/cmd/config/configfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	boshtbl "github.com/cloudfoundry/bosh-cli/ui/table"
)

var _ = Describe("EnvironmentsCmd", func() {
	var (
		config  *fakecmdconf.FakeConfig
		ui      *fakeui.FakeUI
		command EnvironmentsCmd
	)

	BeforeEach(func() {
		config = &fakecmdconf.FakeConfig{}
		ui = &fakeui.FakeUI{}
		command = NewEnvironmentsCmd(config, ui)
	})

	Describe("Run", func() {
		act := func() error { return command.Run() }

		It("lists environments", func() {
			config.EnvironmentsReturns([]cmdconf.Environment{
				{Alias: "environment1-alias", URL: "environment1-url"},
				{Alias: "environment2-alias", URL: "environment2-url"},
			})

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(ui.Table).To(Equal(boshtbl.Table{
				Content: "environments",

				Header: []boshtbl.Header{
					boshtbl.NewHeader("URL"),
					boshtbl.NewHeader("Alias"),
				},

				SortBy: []boshtbl.ColumnSort{{Column: 0, Asc: true}},

				Rows: [][]boshtbl.Value{
					{
						boshtbl.NewValueString("environment1-url"),
						boshtbl.NewValueString("environment1-alias"),
					},
					{
						boshtbl.NewValueString("environment2-url"),
						boshtbl.NewValueString("environment2-alias"),
					},
				},
			}))
		})
	})
})
