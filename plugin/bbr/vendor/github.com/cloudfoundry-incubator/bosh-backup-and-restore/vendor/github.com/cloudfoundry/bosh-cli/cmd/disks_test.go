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

var _ = Describe("DisksCmd", func() {
	var (
		ui       *fakeui.FakeUI
		director *fakedir.FakeDirector
		command  DisksCmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		director = &fakedir.FakeDirector{}
		command = NewDisksCmd(ui, director)
	})

	Describe("Run", func() {
		var (
			opts DisksOpts
		)

		BeforeEach(func() {
			opts = DisksOpts{}
		})

		act := func() error { return command.Run(opts) }

		Context("when orphaned disks requested", func() {
			BeforeEach(func() {
				opts.Orphaned = true
			})

			It("lists disks", func() {
				disks := []boshdir.OrphanDisk{
					&fakedir.FakeOrphanDisk{
						CIDStub:  func() string { return "cid" },
						SizeStub: func() uint64 { return 100 },

						DeploymentStub: func() boshdir.Deployment {
							return &fakedir.FakeDeployment{
								NameStub: func() string { return "deployment" },
							}
						},
						InstanceNameStub: func() string { return "instance" },
						AZNameStub:       func() string { return "az" },

						OrphanedAtStub: func() time.Time {
							return time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
						},
					},
				}

				director.OrphanDisksReturns(disks, nil)

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(ui.Table).To(Equal(boshtbl.Table{
					Content: "disks",

					Header: []boshtbl.Header{
						boshtbl.NewHeader("Disk CID"),
						boshtbl.NewHeader("Size"),
						boshtbl.NewHeader("Deployment"),
						boshtbl.NewHeader("Instance"),
						boshtbl.NewHeader("AZ"),
						boshtbl.NewHeader("Orphaned At"),
					},

					SortBy: []boshtbl.ColumnSort{{Column: 5}},

					Rows: [][]boshtbl.Value{
						{
							boshtbl.NewValueString("cid"),
							boshtbl.NewValueMegaBytes(100),
							boshtbl.NewValueString("deployment"),
							boshtbl.NewValueString("instance"),
							boshtbl.NewValueString("az"),
							boshtbl.NewValueTime(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
						},
					},
				}))
			})

			It("returns error if orphaned disks cannot be retrieved", func() {
				director.OrphanDisksReturns(nil, errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})
		})

		It("returns error if orphaned disks were not requested", func() {
			Expect(act()).To(Equal(errors.New("Only --orphaned is supported")))
		})
	})
})
