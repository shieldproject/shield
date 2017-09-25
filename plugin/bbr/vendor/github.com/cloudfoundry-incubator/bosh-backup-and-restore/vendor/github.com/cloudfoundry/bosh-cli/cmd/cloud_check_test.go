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

var _ = Describe("CloudCheckCmd", func() {
	skipResolutionName := "ignore"
	recreateResolutionName := "recreate_vm"
	rebootResolutionName := "reboot_vm"
	deleteVmReference := "delete_vm_reference"
	deleteDiskReference := "delete_disk_reference"

	var (
		deployment *fakedir.FakeDeployment
		ui         *fakeui.FakeUI
		command    CloudCheckCmd
	)

	BeforeEach(func() {
		deployment = &fakedir.FakeDeployment{}
		ui = &fakeui.FakeUI{}
		command = NewCloudCheckCmd(deployment, ui)
	})

	Describe("Run", func() {
		var (
			opts         CloudCheckOpts
			severalProbs []boshdir.Problem
		)

		BeforeEach(func() {
			opts = CloudCheckOpts{}

			severalProbs = []boshdir.Problem{
				{
					ID: 3,

					Type:        "unresponsive_agent",
					Description: "problem1-desc",

					Resolutions: []boshdir.ProblemResolution{
						{Name: &skipResolutionName, Plan: "Skip for now"},
						{Name: &recreateResolutionName, Plan: "Recreate VM"},
						{Name: &deleteVmReference, Plan: "Delete VM reference"},
					},
				},
				{
					ID: 4,

					Type:        "missing_vm",
					Description: "problem2-desc",

					Resolutions: []boshdir.ProblemResolution{
						{Name: &skipResolutionName, Plan: "Skip for now"},
						{Name: &recreateResolutionName, Plan: "Recreate VM"},
						{Name: &rebootResolutionName, Plan: "Reboot VM"},
						{Name: &deleteDiskReference, Plan: "Delete disk reference (DANGEROUS!)"},
					},
				},
			}
		})

		act := func() error { return command.Run(opts) }

		Context("when trying to resolve problems (not just reporting)", func() {
			Context("when auto resolution is disabled", func() {
				Context("when several problems were found", func() {
					BeforeEach(func() {
						deployment.ScanForProblemsReturns(severalProbs, nil)
						ui.AskedChoiceChosens = []int{1, 0}
						ui.AskedChoiceErrs = []error{nil, nil}
					})

					It("shows problems", func() {
						err := act()
						Expect(err).ToNot(HaveOccurred())

						Expect(ui.Table).To(Equal(boshtbl.Table{
							Content: "problems",

							Header: []boshtbl.Header{
								boshtbl.NewHeader("#"),
								boshtbl.NewHeader("Type"),
								boshtbl.NewHeader("Description"),
							},

							SortBy: []boshtbl.ColumnSort{{Column: 0, Asc: true}},

							Rows: [][]boshtbl.Value{
								{
									boshtbl.NewValueInt(3),
									boshtbl.NewValueString("unresponsive_agent"),
									boshtbl.NewValueString("problem1-desc"),
								},
								{
									boshtbl.NewValueInt(4),
									boshtbl.NewValueString("missing_vm"),
									boshtbl.NewValueString("problem2-desc"),
								},
							},
						}))
					})

					It("resolves problems based on asked answers", func() {
						ui.AskedChoiceChosens = []int{1, 2}

						err := act()
						Expect(err).ToNot(HaveOccurred())

						Expect(ui.AskedChoiceCalled).To(BeTrue())

						Expect(deployment.ResolveProblemsCallCount()).To(Equal(1))

						problemAnswers := deployment.ResolveProblemsArgsForCall(0)
						Expect(len(problemAnswers)).To(Equal(2))

						problemAnswer0 := problemAnswers[0]
						Expect(problemAnswer0.ProblemID).To(Equal(3))
						Expect(*problemAnswer0.Resolution.Name).To(Equal("recreate_vm"))
						Expect(problemAnswer0.Resolution.Plan).To(Equal("Recreate VM"))

						problemAnswer1 := problemAnswers[1]
						Expect(problemAnswer1.ProblemID).To(Equal(4))
						Expect(*problemAnswer1.Resolution.Name).To(Equal("reboot_vm"))
						Expect(problemAnswer1.Resolution.Plan).To(Equal("Reboot VM"))
					})

					It("does not resolve problems if confirmation is rejected", func() {
						ui.AskedConfirmationErr = errors.New("stop")

						err := act()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("stop"))

						Expect(deployment.ResolveProblemsCallCount()).To(Equal(0))
					})

					It("returns error if failed asking", func() {
						ui.AskedChoiceErrs = []error{nil, errors.New("fake-err")}

						err := act()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("fake-err"))

						Expect(deployment.ResolveProblemsCallCount()).To(Equal(0))
					})
				})

				Context("when no problems were found", func() {
					BeforeEach(func() {
						deployment.ScanForProblemsReturns(nil, nil)
					})

					It("does try to resolve any problem", func() {
						err := act()
						Expect(err).ToNot(HaveOccurred())

						Expect(ui.Tables).To(Equal([]boshtbl.Table{
							{
								Content: "problems",
								Header: []boshtbl.Header{
									boshtbl.NewHeader("#"),
									boshtbl.NewHeader("Type"),
									boshtbl.NewHeader("Description"),
								},
								SortBy: []boshtbl.ColumnSort{{Column: 0, Asc: true}},
							},
						}))

						Expect(deployment.ResolveProblemsCallCount()).To(Equal(0))
					})

					It("does not ask for confirmation or with choices", func() {
						err := act()
						Expect(err).ToNot(HaveOccurred())

						Expect(ui.AskedChoiceCalled).To(BeFalse())
						Expect(ui.AskedConfirmationCalled).To(BeFalse())
					})
				})

				It("returns error if scannig for problems failed", func() {
					deployment.ScanForProblemsReturns(nil, errors.New("fake-err"))

					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-err"))

					Expect(deployment.ResolveProblemsCallCount()).To(Equal(0))
				})
			})

			Context("when auto resolution is enabled", func() {
				BeforeEach(func() {
					opts.Auto = true
				})

				Context("when several problems were found", func() {
					BeforeEach(func() {
						deployment.ScanForProblemsReturns(severalProbs, nil)
					})

					It("shows problems", func() {
						err := act()
						Expect(err).ToNot(HaveOccurred())

						Expect(ui.Table).To(Equal(boshtbl.Table{
							Content: "problems",

							Header: []boshtbl.Header{
								boshtbl.NewHeader("#"),
								boshtbl.NewHeader("Type"),
								boshtbl.NewHeader("Description"),
							},

							SortBy: []boshtbl.ColumnSort{{Column: 0, Asc: true}},

							Rows: [][]boshtbl.Value{
								{
									boshtbl.NewValueInt(3),
									boshtbl.NewValueString("unresponsive_agent"),
									boshtbl.NewValueString("problem1-desc"),
								},
								{
									boshtbl.NewValueInt(4),
									boshtbl.NewValueString("missing_vm"),
									boshtbl.NewValueString("problem2-desc"),
								},
							},
						}))
					})

					It("automatically resolves problems without asking", func() {
						err := act()
						Expect(err).ToNot(HaveOccurred())

						Expect(deployment.ResolveProblemsCallCount()).To(Equal(1))
						Expect(deployment.ResolveProblemsArgsForCall(0)).To(Equal([]boshdir.ProblemAnswer{
							{
								ProblemID:  3,
								Resolution: boshdir.ProblemResolutionDefault,
							},
							{
								ProblemID:  4,
								Resolution: boshdir.ProblemResolutionDefault,
							},
						}))

						Expect(ui.AskedChoiceCalled).To(BeFalse())
					})

					It("does not automatically resolve problems if confirmation is rejected", func() {
						ui.AskedConfirmationErr = errors.New("stop")

						err := act()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("stop"))

						Expect(deployment.ResolveProblemsCallCount()).To(Equal(0))
					})
				})

				Context("when no problems were found", func() {
					BeforeEach(func() {
						deployment.ScanForProblemsReturns(nil, nil)
					})

					It("does try to resolve any problem", func() {
						err := act()
						Expect(err).ToNot(HaveOccurred())

						Expect(ui.Tables).To(Equal([]boshtbl.Table{
							{
								Content: "problems",
								Header: []boshtbl.Header{
									boshtbl.NewHeader("#"),
									boshtbl.NewHeader("Type"),
									boshtbl.NewHeader("Description"),
								},
								SortBy: []boshtbl.ColumnSort{{Column: 0, Asc: true}},
							},
						}))

						Expect(deployment.ResolveProblemsCallCount()).To(Equal(0))
					})

					It("does not ask for confirmation or with choices", func() {
						err := act()
						Expect(err).ToNot(HaveOccurred())

						Expect(ui.AskedChoiceCalled).To(BeFalse())
						Expect(ui.AskedConfirmationCalled).To(BeFalse())
					})
				})

				It("returns error if scannig for problems failed", func() {
					deployment.ScanForProblemsReturns(nil, errors.New("fake-err"))

					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-err"))

					Expect(deployment.ResolveProblemsCallCount()).To(Equal(0))
				})
			})

			Context("when resolutions are provided", func() {
				BeforeEach(func() {
					opts.Auto = false
					opts.Resolutions = []string{"delete_disk_reference", "delete_vm_reference"}
				})

				Context("when every problems has a matching resolution", func() {
					BeforeEach(func() {
						deployment.ScanForProblemsReturns(severalProbs, nil)
					})

					It("automatically resolves problems without asking", func() {
						err := act()
						Expect(err).ToNot(HaveOccurred())

						Expect(deployment.ResolveProblemsCallCount()).To(Equal(1))
						Expect(deployment.ResolveProblemsArgsForCall(0)).To(Equal([]boshdir.ProblemAnswer{
							{
								ProblemID: 3,
								Resolution: boshdir.ProblemResolution{
									Name: &deleteVmReference,
									Plan: "Delete VM reference",
								},
							},
							{
								ProblemID: 4,
								Resolution: boshdir.ProblemResolution{
									Name: &deleteDiskReference,
									Plan: "Delete disk reference (DANGEROUS!)",
								},
							},
						}))

						Expect(ui.AskedChoiceCalled).To(BeFalse())
					})

				})
				Context("when some problems do not have a matching resolution", func() {
					BeforeEach(func() {
						severalProbs = append(severalProbs, boshdir.Problem{
							ID:          5,
							Type:        "unresponsive_agent",
							Description: "problem3-desc",

							Resolutions: []boshdir.ProblemResolution{
								{Name: &skipResolutionName, Plan: "Skip for now"},
							},
						})

						deployment.ScanForProblemsReturns(severalProbs, nil)
					})

					It("ignores/skips problems it cannot resolve", func() {
						err := act()
						Expect(err).ToNot(HaveOccurred())

						Expect(deployment.ResolveProblemsCallCount()).To(Equal(1))
						Expect(deployment.ResolveProblemsArgsForCall(0)).To(Equal([]boshdir.ProblemAnswer{
							{
								ProblemID: 3,
								Resolution: boshdir.ProblemResolution{
									Name: &deleteVmReference,
									Plan: "Delete VM reference",
								},
							},
							{
								ProblemID: 4,
								Resolution: boshdir.ProblemResolution{
									Name: &deleteDiskReference,
									Plan: "Delete disk reference (DANGEROUS!)",
								},
							},
							{
								ProblemID: 5,
								Resolution: boshdir.ProblemResolution{
									Name: &skipResolutionName,
									Plan: "Skip for now",
								},
							},
						}))

						Expect(ui.AskedChoiceCalled).To(BeFalse())
					})

					It("does not automatically resolve problems if confirmation is rejected", func() {
						ui.AskedConfirmationErr = errors.New("stop")

						err := act()
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("stop"))

						Expect(deployment.ResolveProblemsCallCount()).To(Equal(0))
					})
				})

				Context("when no problems were found", func() {
					BeforeEach(func() {
						deployment.ScanForProblemsReturns(nil, nil)
					})

					It("does try to resolve any problem", func() {
						err := act()
						Expect(err).ToNot(HaveOccurred())

						Expect(ui.Tables).To(Equal([]boshtbl.Table{
							{
								Content: "problems",
								Header: []boshtbl.Header{
									boshtbl.NewHeader("#"),
									boshtbl.NewHeader("Type"),
									boshtbl.NewHeader("Description"),
								},
								SortBy: []boshtbl.ColumnSort{{Column: 0, Asc: true}},
							},
						}))

						Expect(deployment.ResolveProblemsCallCount()).To(Equal(0))
					})

					It("does not ask for confirmation or with choices", func() {
						err := act()
						Expect(err).ToNot(HaveOccurred())

						Expect(ui.AskedChoiceCalled).To(BeFalse())
						Expect(ui.AskedConfirmationCalled).To(BeFalse())
					})
				})

				It("returns error if scannig for problems failed", func() {
					deployment.ScanForProblemsReturns(nil, errors.New("fake-err"))

					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-err"))

					Expect(deployment.ResolveProblemsCallCount()).To(Equal(0))
				})
			})
		})

		Context("when only reporting", func() {
			BeforeEach(func() {
				opts.Report = true
			})

			It("returns error if there are any problems found by the scan", func() {
				probs := []boshdir.Problem{
					{
						ID: 3,

						Type:        "unresponsive_agent",
						Description: "problem1-desc",

						Data: nil,
						Resolutions: []boshdir.ProblemResolution{
							{
								Name: &skipResolutionName,
								Plan: "Skip for now",
							},
						},
					},
					{
						ID: 4,

						Type:        "missing_vm",
						Description: "problem2-desc",

						Data: nil,
						Resolutions: []boshdir.ProblemResolution{
							{
								Name: &recreateResolutionName,
								Plan: "Recreate VM",
							},
						},
					},
				}

				deployment.ScanForProblemsReturns(probs, nil)

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("2 problem(s) found"))

				Expect(deployment.ResolveProblemsCallCount()).To(Equal(0))

				Expect(ui.Table).To(Equal(boshtbl.Table{
					Content: "problems",

					Header: []boshtbl.Header{
						boshtbl.NewHeader("#"),
						boshtbl.NewHeader("Type"),
						boshtbl.NewHeader("Description"),
					},

					SortBy: []boshtbl.ColumnSort{{Column: 0, Asc: true}},

					Rows: [][]boshtbl.Value{
						{
							boshtbl.NewValueInt(3),
							boshtbl.NewValueString("unresponsive_agent"),
							boshtbl.NewValueString("problem1-desc"),
						},
						{
							boshtbl.NewValueInt(4),
							boshtbl.NewValueString("missing_vm"),
							boshtbl.NewValueString("problem2-desc"),
						},
					},
				}))
			})

			It("does not return error if no problems found", func() {
				deployment.ScanForProblemsReturns([]boshdir.Problem{}, nil)

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(deployment.ResolveProblemsCallCount()).To(Equal(0))

				Expect(ui.Tables).ToNot(BeEmpty())
			})

			It("returns error if scannig for problems failed", func() {
				deployment.ScanForProblemsReturns(nil, errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))

				Expect(deployment.ResolveProblemsCallCount()).To(Equal(0))
			})
		})
	})
})
