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

var _ = Describe("VMsCmd", func() {
	var (
		ui       *fakeui.FakeUI
		director *fakedir.FakeDirector
		command  VMsCmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		director = &fakedir.FakeDirector{}
		command = NewVMsCmd(ui, director)
	})

	Describe("Run", func() {
		var (
			opts  VMsOpts
			infos []boshdir.VMInfo
		)

		BeforeEach(func() {
			opts = VMsOpts{}
		})

		act := func() error { return command.Run(opts) }

		BeforeEach(func() {
			index1 := 1
			index2 := 2

			infos = []boshdir.VMInfo{
				{
					JobName:      "job-name",
					Index:        &index1,
					ProcessState: "in1-process-state",
					ResourcePool: "in1-rp",

					IPs: []string{"in1-ip1", "in1-ip2"},
					DNS: []string{"in1-dns1", "in1-dns2"},

					VMID:               "in1-cid",
					AgentID:            "in1-agent-id",
					ResurrectionPaused: false,
					Ignore:             false,
					DiskIDs:            []string{"diskcid1", "diskcid2"},

					Vitals: boshdir.VMInfoVitals{
						Load: []string{"0.02", "0.06", "0.11"},

						CPU:  boshdir.VMInfoVitalsCPU{Sys: "0.3", User: "1.2", Wait: "2.1"},
						Mem:  boshdir.VMInfoVitalsMemSize{Percent: "20", KB: "2000"},
						Swap: boshdir.VMInfoVitalsMemSize{Percent: "21", KB: "2100"},

						Disk: map[string]boshdir.VMInfoVitalsDiskSize{
							"system":     boshdir.VMInfoVitalsDiskSize{Percent: "35"},
							"ephemeral":  boshdir.VMInfoVitalsDiskSize{Percent: "45"},
							"persistent": boshdir.VMInfoVitalsDiskSize{Percent: "55"},
						},
					},
				},
				{
					JobName:      "job-name",
					Index:        &index2,
					ProcessState: "in2-process-state",
					AZ:           "in2-az",
					ResourcePool: "in2-rp",

					IPs: []string{"in2-ip1"},
					DNS: []string{"in2-dns1"},

					VMID:               "in2-cid",
					AgentID:            "in2-agent-id",
					ResurrectionPaused: true,
					Ignore:             true,
					DiskIDs:            []string{"diskcid1", "diskcid2"},

					Vitals: boshdir.VMInfoVitals{
						Load: []string{"0.52", "0.56", "0.51"},

						CPU:  boshdir.VMInfoVitalsCPU{Sys: "50.3", User: "51.2", Wait: "52.1"},
						Mem:  boshdir.VMInfoVitalsMemSize{Percent: "60", KB: "6000"},
						Swap: boshdir.VMInfoVitalsMemSize{Percent: "61", KB: "6100"},

						Disk: map[string]boshdir.VMInfoVitalsDiskSize{
							"system":     boshdir.VMInfoVitalsDiskSize{Percent: "75"},
							"ephemeral":  boshdir.VMInfoVitalsDiskSize{Percent: "85"},
							"persistent": boshdir.VMInfoVitalsDiskSize{Percent: "95"},
						},
					},
				},
				{
					JobName:      "",
					Index:        nil,
					ProcessState: "unresponsive agent",
					ResourcePool: "",
				},
			}
		})

		Context("when listing all deployments", func() {
			Context("when VMs are successfully retrieved", func() {
				BeforeEach(func() {
					deployments := []boshdir.Deployment{
						&fakedir.FakeDeployment{
							NameStub:    func() string { return "dep1" },
							VMInfosStub: func() ([]boshdir.VMInfo, error) { return infos, nil },
						},
					}

					director.DeploymentsReturns(deployments, nil)
				})

				It("lists VMs for the deployment", func() {
					Expect(act()).ToNot(HaveOccurred())

					Expect(ui.Table).To(Equal(boshtbl.Table{
						Title: "Deployment 'dep1'",

						Content: "vms",

						Header: []boshtbl.Header{
							boshtbl.NewHeader("Instance"),
							boshtbl.NewHeader("Process State"),
							boshtbl.NewHeader("AZ"),
							boshtbl.NewHeader("IPs"),
							boshtbl.NewHeader("VM CID"),
							boshtbl.NewHeader("VM Type"),
						},

						SortBy: []boshtbl.ColumnSort{{Column: 0, Asc: true}},

						Rows: [][]boshtbl.Value{
							{
								boshtbl.NewValueString("job-name"),
								boshtbl.NewValueFmt(boshtbl.NewValueString("in1-process-state"), true),
								boshtbl.ValueString{},
								boshtbl.NewValueStrings([]string{"in1-ip1", "in1-ip2"}),
								boshtbl.NewValueString("in1-cid"),
								boshtbl.NewValueString("in1-rp"),
							},
							{
								boshtbl.NewValueString("job-name"),
								boshtbl.NewValueFmt(boshtbl.NewValueString("in2-process-state"), true),
								boshtbl.NewValueString("in2-az"),
								boshtbl.NewValueStrings([]string{"in2-ip1"}),
								boshtbl.NewValueString("in2-cid"),
								boshtbl.NewValueString("in2-rp"),
							},
							{
								boshtbl.NewValueString("?"),
								boshtbl.NewValueFmt(boshtbl.NewValueString("unresponsive agent"), true),
								boshtbl.ValueString{},
								boshtbl.ValueStrings{},
								boshtbl.ValueString{},
								boshtbl.ValueString{},
							},
						},
					}))
				})

				It("lists VMs for the deployment including dns", func() {
					opts.DNS = true

					Expect(act()).ToNot(HaveOccurred())

					Expect(ui.Table).To(Equal(boshtbl.Table{
						Title: "Deployment 'dep1'",

						Content: "vms",

						Header: []boshtbl.Header{
							boshtbl.NewHeader("Instance"),
							boshtbl.NewHeader("Process State"),
							boshtbl.NewHeader("AZ"),
							boshtbl.NewHeader("IPs"),
							boshtbl.NewHeader("VM CID"),
							boshtbl.NewHeader("VM Type"),
							boshtbl.NewHeader("DNS A Records"),
						},

						SortBy: []boshtbl.ColumnSort{{Column: 0, Asc: true}},

						Rows: [][]boshtbl.Value{
							{
								boshtbl.NewValueString("job-name"),
								boshtbl.NewValueFmt(boshtbl.NewValueString("in1-process-state"), true),
								boshtbl.ValueString{},
								boshtbl.NewValueStrings([]string{"in1-ip1", "in1-ip2"}),
								boshtbl.NewValueString("in1-cid"),
								boshtbl.NewValueString("in1-rp"),
								boshtbl.NewValueStrings([]string{"in1-dns1", "in1-dns2"}),
							},
							{
								boshtbl.NewValueString("job-name"),
								boshtbl.NewValueFmt(boshtbl.NewValueString("in2-process-state"), true),
								boshtbl.NewValueString("in2-az"),
								boshtbl.NewValueStrings([]string{"in2-ip1"}),
								boshtbl.NewValueString("in2-cid"),
								boshtbl.NewValueString("in2-rp"),
								boshtbl.NewValueStrings([]string{"in2-dns1"}),
							},
							{
								boshtbl.NewValueString("?"),
								boshtbl.NewValueFmt(boshtbl.NewValueString("unresponsive agent"), true),
								boshtbl.ValueString{},
								boshtbl.ValueStrings{},
								boshtbl.ValueString{},
								boshtbl.ValueString{},
								boshtbl.ValueStrings{},
							},
						},
					}))
				})

				It("lists VMs for the deployment including vitals", func() {
					opts.Vitals = true

					Expect(act()).ToNot(HaveOccurred())

					Expect(ui.Table).To(Equal(boshtbl.Table{
						Title: "Deployment 'dep1'",

						Content: "vms",

						Header: []boshtbl.Header{
							boshtbl.NewHeader("Instance"),
							boshtbl.NewHeader("Process State"),
							boshtbl.NewHeader("AZ"),
							boshtbl.NewHeader("IPs"),
							boshtbl.NewHeader("VM CID"),
							boshtbl.NewHeader("VM Type"),
							boshtbl.NewHeader("Uptime"),
							boshtbl.NewHeader("Load\n(1m, 5m, 15m)"),
							boshtbl.NewHeader("CPU\nTotal"),
							boshtbl.NewHeader("CPU\nUser"),
							boshtbl.NewHeader("CPU\nSys"),
							boshtbl.NewHeader("CPU\nWait"),
							boshtbl.NewHeader("Memory\nUsage"),
							boshtbl.NewHeader("Swap\nUsage"),
							boshtbl.NewHeader("System\nDisk Usage"),
							boshtbl.NewHeader("Ephemeral\nDisk Usage"),
							boshtbl.NewHeader("Persistent\nDisk Usage"),
						},

						SortBy: []boshtbl.ColumnSort{{Column: 0, Asc: true}},

						Rows: [][]boshtbl.Value{
							{
								boshtbl.NewValueString("job-name"),
								boshtbl.NewValueFmt(boshtbl.NewValueString("in1-process-state"), true),
								boshtbl.ValueString{},
								boshtbl.NewValueStrings([]string{"in1-ip1", "in1-ip2"}),
								boshtbl.NewValueString("in1-cid"),
								boshtbl.NewValueString("in1-rp"),
								ValueUptime{},
								boshtbl.NewValueString("0.02, 0.06, 0.11"),
								ValueCPUTotal{},
								NewValueStringPercent("1.2"),
								NewValueStringPercent("0.3"),
								NewValueStringPercent("2.1"),
								ValueMemSize{boshdir.VMInfoVitalsMemSize{Percent: "20", KB: "2000"}},
								ValueMemSize{boshdir.VMInfoVitalsMemSize{Percent: "21", KB: "2100"}},
								ValueDiskSize{boshdir.VMInfoVitalsDiskSize{Percent: "35"}},
								ValueDiskSize{boshdir.VMInfoVitalsDiskSize{Percent: "45"}},
								ValueDiskSize{boshdir.VMInfoVitalsDiskSize{Percent: "55"}},
							},
							{
								boshtbl.NewValueString("job-name"),
								boshtbl.NewValueFmt(boshtbl.NewValueString("in2-process-state"), true),
								boshtbl.NewValueString("in2-az"),
								boshtbl.NewValueStrings([]string{"in2-ip1"}),
								boshtbl.NewValueString("in2-cid"),
								boshtbl.NewValueString("in2-rp"),
								ValueUptime{},
								boshtbl.NewValueString("0.52, 0.56, 0.51"),
								ValueCPUTotal{},
								NewValueStringPercent("51.2"),
								NewValueStringPercent("50.3"),
								NewValueStringPercent("52.1"),
								ValueMemSize{boshdir.VMInfoVitalsMemSize{Percent: "60", KB: "6000"}},
								ValueMemSize{boshdir.VMInfoVitalsMemSize{Percent: "61", KB: "6100"}},
								ValueDiskSize{boshdir.VMInfoVitalsDiskSize{Percent: "75"}},
								ValueDiskSize{boshdir.VMInfoVitalsDiskSize{Percent: "85"}},
								ValueDiskSize{boshdir.VMInfoVitalsDiskSize{Percent: "95"}},
							},
							{
								boshtbl.NewValueString("?"),
								boshtbl.NewValueFmt(boshtbl.NewValueString("unresponsive agent"), true),
								boshtbl.ValueString{},
								boshtbl.ValueStrings{},
								boshtbl.ValueString{},
								boshtbl.ValueString{},
								ValueUptime{},
								boshtbl.ValueString{},
								ValueCPUTotal{},
								NewValueStringPercent(""),
								NewValueStringPercent(""),
								NewValueStringPercent(""),
								ValueMemSize{},
								ValueMemSize{},
								ValueDiskSize{},
								ValueDiskSize{},
								ValueDiskSize{},
							},
						},
					}))
				})
			})

			It("returns error if VMs cannot be retrieved", func() {
				deployments := []boshdir.Deployment{
					&fakedir.FakeDeployment{
						NameStub:    func() string { return "dep1" },
						VMInfosStub: func() ([]boshdir.VMInfo, error) { return nil, errors.New("fake-err") },
					},
				}

				director.DeploymentsReturns(deployments, nil)

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("returns error if deployments cannot be retrieved", func() {
				director.DeploymentsReturns(nil, errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})
		})

		Context("when listing single deployment", func() {
			BeforeEach(func() {
				opts.Deployment = "dep1"
			})

			It("lists VMs for the deployment", func() {
				deployment := &fakedir.FakeDeployment{
					NameStub:    func() string { return "dep1" },
					VMInfosStub: func() ([]boshdir.VMInfo, error) { return infos, nil },
				}

				director.FindDeploymentReturns(deployment, nil)

				Expect(act()).ToNot(HaveOccurred())

				Expect(ui.Table).To(Equal(boshtbl.Table{
					Title: "Deployment 'dep1'",

					Content: "vms",

					Header: []boshtbl.Header{
						boshtbl.NewHeader("Instance"),
						boshtbl.NewHeader("Process State"),
						boshtbl.NewHeader("AZ"),
						boshtbl.NewHeader("IPs"),
						boshtbl.NewHeader("VM CID"),
						boshtbl.NewHeader("VM Type"),
					},

					SortBy: []boshtbl.ColumnSort{{Column: 0, Asc: true}},

					Rows: [][]boshtbl.Value{
						{
							boshtbl.NewValueString("job-name"),
							boshtbl.NewValueFmt(boshtbl.NewValueString("in1-process-state"), true),
							boshtbl.ValueString{},
							boshtbl.NewValueStrings([]string{"in1-ip1", "in1-ip2"}),
							boshtbl.NewValueString("in1-cid"),
							boshtbl.NewValueString("in1-rp"),
						},
						{
							boshtbl.NewValueString("job-name"),
							boshtbl.NewValueFmt(boshtbl.NewValueString("in2-process-state"), true),
							boshtbl.NewValueString("in2-az"),
							boshtbl.NewValueStrings([]string{"in2-ip1"}),
							boshtbl.NewValueString("in2-cid"),
							boshtbl.NewValueString("in2-rp"),
						},
						{
							boshtbl.NewValueString("?"),
							boshtbl.NewValueFmt(boshtbl.NewValueString("unresponsive agent"), true),
							boshtbl.ValueString{},
							boshtbl.ValueStrings{},
							boshtbl.ValueString{},
							boshtbl.ValueString{},
						},
					},
				}))
			})

			It("returns error if VMs cannot be retrieved", func() {
				deployment := &fakedir.FakeDeployment{
					NameStub:    func() string { return "dep1" },
					VMInfosStub: func() ([]boshdir.VMInfo, error) { return nil, errors.New("fake-err") },
				}

				director.FindDeploymentReturns(deployment, nil)

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("returns error if finding deployment fails", func() {
				director.FindDeploymentReturns(nil, errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})
		})
	})
})
