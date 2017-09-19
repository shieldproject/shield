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

var _ = Describe("ErrandsCmd", func() {
	var (
		ui         *fakeui.FakeUI
		deployment *fakedir.FakeDeployment
		command    ErrandsCmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		deployment = &fakedir.FakeDeployment{}
		command = NewErrandsCmd(ui, deployment)
	})

	Describe("Run", func() {
		act := func() error { return command.Run() }

		It("lists current errands", func() {
			errands := []boshdir.Errand{
				{
					Name: "some-errand",
				},
			}

			deployment.ErrandsReturns(errands, nil)

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(ui.Table).To(Equal(boshtbl.Table{
				Content: "errands",

				Header: []boshtbl.Header{boshtbl.NewHeader("Name")},

				SortBy: []boshtbl.ColumnSort{{Column: 0, Asc: true}},

				Rows: [][]boshtbl.Value{
					{boshtbl.NewValueString("some-errand")},
				},
			}))
		})

		It("returns error if errands cannot be retrieved", func() {
			deployment.ErrandsReturns(nil, errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
