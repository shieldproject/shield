package cmd_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	boshtbl "github.com/cloudfoundry/bosh-cli/ui/table"
	"time"
)

var _ = Describe("EventCmd", func() {
	var (
		ui       *fakeui.FakeUI
		director *fakedir.FakeDirector
		command  EventCmd
		event    boshdir.Event
		opts     EventOpts
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		director = &fakedir.FakeDirector{}
		command = NewEventCmd(ui, director)
		opts.Args.ID = "4"
	})

	Describe("Run", func() {
		It("outputs a table that should be transposed", func() {
			event = &fakedir.FakeEvent{}

			director.EventReturns(event, nil)

			err := command.Run(opts)
			Expect(err).ToNot(HaveOccurred())

			Expect(ui.Table.Transpose).To(Equal(true))
		})

		Context("when no optional values are present in event", func() {
			It("shows minimal information about event", func() {
				event = &fakedir.FakeEvent{
					IDStub: func() string {
						return "4"
					},

					TimestampStub: func() time.Time {
						return time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
					},
					ActionStub: func() string {
						return "action"
					},
					ObjectTypeStub: func() string {
						return "object-type"
					},
				}

				director.EventReturns(event, nil)

				err := command.Run(opts)
				Expect(err).ToNot(HaveOccurred())

				Expect(ui.Table.Header).To(Equal([]boshtbl.Header{
					boshtbl.NewHeader("ID"),
					boshtbl.NewHeader("Time"),
					boshtbl.NewHeader("Action"),
					boshtbl.NewHeader("Object Type"),
				}))
				Expect(ui.Table.Rows).To(Equal([][]boshtbl.Value{
					{
						boshtbl.NewValueString("4"),
						boshtbl.NewValueTime(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
						boshtbl.NewValueString("action"),
						boshtbl.NewValueString("object-type"),
					},
				}))
			})
		})

		Context("when all optional values are present in event", func() {
			It("shows full information about event with proper ordering", func() {
				event = &fakedir.FakeEvent{
					IDStub: func() string {
						return "4"
					},
					ParentIDStub: func() string {
						return "1"
					},
					TimestampStub: func() time.Time {
						return time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
					},

					UserStub: func() string {
						return "user"
					},

					ActionStub: func() string {
						return "action"
					},
					ObjectTypeStub: func() string {
						return "object-type"
					},
					ObjectNameStub: func() string {
						return "object-name"
					},
					TaskIDStub: func() string {
						return "task"
					},
					DeploymentNameStub: func() string {
						return "deployment"
					},
					InstanceStub: func() string {
						return "instance"
					},
					ContextStub: func() map[string]interface{} {
						return map[string]interface{}{"user": "bosh_z$", "test_variable": "test_value"}
					},
					ErrorStub: func() string {
						return "some-error"
					},
				}

				director.EventReturns(event, nil)

				err := command.Run(opts)
				Expect(err).ToNot(HaveOccurred())

				Expect(ui.Table.Header).To(Equal([]boshtbl.Header{
					boshtbl.NewHeader("ID"),
					boshtbl.NewHeader("Time"),
					boshtbl.NewHeader("User"),
					boshtbl.NewHeader("Action"),
					boshtbl.NewHeader("Object Type"),
					boshtbl.NewHeader("Object Name"),
					boshtbl.NewHeader("Task ID"),
					boshtbl.NewHeader("Deployment"),
					boshtbl.NewHeader("Instance"),
					boshtbl.NewHeader("Context"),
					boshtbl.NewHeader("Error"),
				}))
				Expect(ui.Table.Rows).To(Equal([][]boshtbl.Value{
					{
						boshtbl.NewValueString("4 <- 1"),
						boshtbl.NewValueTime(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
						boshtbl.NewValueString("user"),
						boshtbl.NewValueString("action"),
						boshtbl.NewValueString("object-type"),
						boshtbl.NewValueString("object-name"),
						boshtbl.NewValueString("task"),
						boshtbl.NewValueString("deployment"),
						boshtbl.NewValueString("instance"),
						boshtbl.NewValueStrings([]string{"test_variable: test_value", "user: bosh_z$"}),
						boshtbl.NewValueString("some-error"),
					},
				}))
			})
		})

		Describe("optional value rendering", func() {
			It("shows user when present", func() {
				event = &fakedir.FakeEvent{
					UserStub: func() string {
						return "fake-user"
					},
				}

				director.EventReturns(event, nil)

				err := command.Run(opts)
				Expect(err).ToNot(HaveOccurred())

				Expect(ui.Table.Header).To(ContainElement(boshtbl.NewHeader("User")))
				Expect(ui.Table.Rows).To(HaveLen(1))
				Expect(ui.Table.Rows[0]).To(ContainElement(boshtbl.NewValueString("fake-user")))
			})

			It("shows object name when present", func() {
				event = &fakedir.FakeEvent{
					ObjectNameStub: func() string {
						return "fake-object"
					},
				}

				director.EventReturns(event, nil)

				err := command.Run(opts)
				Expect(err).ToNot(HaveOccurred())

				Expect(ui.Table.Header).To(ContainElement(boshtbl.NewHeader("Object Name")))
				Expect(ui.Table.Rows).To(HaveLen(1))
				Expect(ui.Table.Rows[0]).To(ContainElement(boshtbl.NewValueString("fake-object")))
			})

			It("shows task id when present", func() {
				event = &fakedir.FakeEvent{
					TaskIDStub: func() string {
						return "fake-task"
					},
				}

				director.EventReturns(event, nil)

				err := command.Run(opts)
				Expect(err).ToNot(HaveOccurred())

				Expect(ui.Table.Header).To(ContainElement(boshtbl.NewHeader("Task ID")))
				Expect(ui.Table.Rows).To(HaveLen(1))
				Expect(ui.Table.Rows[0]).To(ContainElement(boshtbl.NewValueString("fake-task")))
			})

			It("shows deployment when present", func() {
				event = &fakedir.FakeEvent{
					DeploymentNameStub: func() string {
						return "fake-deployment"
					},
				}

				director.EventReturns(event, nil)

				err := command.Run(opts)
				Expect(err).ToNot(HaveOccurred())

				Expect(ui.Table.Header).To(ContainElement(boshtbl.NewHeader("Deployment")))
				Expect(ui.Table.Rows).To(HaveLen(1))
				Expect(ui.Table.Rows[0]).To(ContainElement(boshtbl.NewValueString("fake-deployment")))
			})

			It("shows instance when present", func() {
				event = &fakedir.FakeEvent{
					InstanceStub: func() string {
						return "fake-instance"
					},
				}

				director.EventReturns(event, nil)

				err := command.Run(opts)
				Expect(err).ToNot(HaveOccurred())

				Expect(ui.Table.Header).To(ContainElement(boshtbl.NewHeader("Instance")))
				Expect(ui.Table.Rows).To(HaveLen(1))
				Expect(ui.Table.Rows[0]).To(ContainElement(boshtbl.NewValueString("fake-instance")))
			})

			It("shows context when present", func() {
				event = &fakedir.FakeEvent{
					ContextStub: func() map[string]interface{} {
						return map[string]interface{}{"fake-key": "fake-value"}
					},
				}

				director.EventReturns(event, nil)

				err := command.Run(opts)
				Expect(err).ToNot(HaveOccurred())

				Expect(ui.Table.Header).To(ContainElement(boshtbl.NewHeader("Context")))
				Expect(ui.Table.Rows).To(HaveLen(1))
				Expect(ui.Table.Rows[0]).To(ContainElement(boshtbl.NewValueStrings([]string{"fake-key: fake-value"})))
			})

			It("shows error when present", func() {
				event = &fakedir.FakeEvent{
					ErrorStub: func() string {
						return "fake-error"
					},
				}

				director.EventReturns(event, nil)

				err := command.Run(opts)
				Expect(err).ToNot(HaveOccurred())

				Expect(ui.Table.Header).To(ContainElement(boshtbl.NewHeader("Error")))
				Expect(ui.Table.Rows).To(HaveLen(1))
				Expect(ui.Table.Rows[0]).To(ContainElement(boshtbl.NewValueString("fake-error")))
			})
		})
	})
})
