package cmd_test

import (
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	boshtbl "github.com/cloudfoundry/bosh-cli/ui/table"
)

var _ = Describe("SnapshotsCmd", func() {
	var (
		ui         *fakeui.FakeUI
		deployment *fakedir.FakeDeployment
		command    SnapshotsCmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		deployment = &fakedir.FakeDeployment{}
		command = NewSnapshotsCmd(ui, deployment)
	})

	Describe("Run", func() {
		act := func() error { return command.Run(SnapshotsOpts{}) }

		It("lists current snapshots", func() {
			jobIndex := 10

			snapshots := []boshdir.Snapshot{
				{
					Job:   "some-job",
					Index: &jobIndex,

					CID:       "some-cid",
					CreatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),

					Clean: true,
				},
			}

			deployment.SnapshotsReturns(snapshots, nil)

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(ui.Table).To(Equal(boshtbl.Table{
				Content: "snapshots",

				Header: []boshtbl.Header{
					boshtbl.NewHeader("Instance"),
					boshtbl.NewHeader("CID"),
					boshtbl.NewHeader("Created At"),
					boshtbl.NewHeader("Clean"),
				},

				Rows: [][]boshtbl.Value{
					{
						boshtbl.NewValueString("some-job/10"),
						boshtbl.NewValueString("some-cid"),
						boshtbl.NewValueTime(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
						boshtbl.NewValueBool(true),
					},
				},
			}))
		})

		It("returns error if snapshots cannot be retrieved", func() {
			deployment.SnapshotsReturns(nil, errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
