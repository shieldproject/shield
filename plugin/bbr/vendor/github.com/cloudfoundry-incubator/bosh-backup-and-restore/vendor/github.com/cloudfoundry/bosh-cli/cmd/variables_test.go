package cmd_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	boshtbl "github.com/cloudfoundry/bosh-cli/ui/table"
)

var _ = Describe("VariablesCmd", func() {
	var (
		ui         *fakeui.FakeUI
		deployment *fakedir.FakeDeployment
		command    VariablesCmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		deployment = &fakedir.FakeDeployment{}
		command = NewVariablesCmd(ui, deployment)
	})

	Describe("Run", func() {
		act := func() error { return command.Run() }

		It("lists variables", func() {
			variables := []boshdir.VariableResult{
				{ID: "1", Name: "foo-1"},
				{ID: "2", Name: "foo-2"},
				{ID: "3", Name: "foo-3"},
				{ID: "4", Name: "foo-4"},
			}
			deployment.VariablesReturns(variables, nil)

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(ui.Table).To(Equal(boshtbl.Table{
				Content: "variables",

				Header: []boshtbl.Header{boshtbl.NewHeader("ID"), boshtbl.NewHeader("Name")},

				SortBy: []boshtbl.ColumnSort{
					{Column: 1, Asc: true},
				},

				Rows: [][]boshtbl.Value{
					{
						boshtbl.NewValueString("1"),
						boshtbl.NewValueString("foo-1"),
					},
					{
						boshtbl.NewValueString("2"),
						boshtbl.NewValueString("foo-2"),
					},
					{
						boshtbl.NewValueString("3"),
						boshtbl.NewValueString("foo-3"),
					},
					{
						boshtbl.NewValueString("4"),
						boshtbl.NewValueString("foo-4"),
					},
				},
			}))
		})

		It("returns error if getting variables fails", func() {
			deployment.VariablesReturns(nil, errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
