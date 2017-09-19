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

var _ = Describe("EventsCmd", func() {
	var (
		ui       *fakeui.FakeUI
		director *fakedir.FakeDirector
		command  EventsCmd
		events   []boshdir.Event
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		director = &fakedir.FakeDirector{}
		command = NewEventsCmd(ui, director)
		events = []boshdir.Event{
			&fakedir.FakeEvent{
				IDStub:        func() string { return "4" },
				ParentIDStub:  func() string { return "1" },
				TimestampStub: func() time.Time { return time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC) },

				UserStub: func() string { return "user" },

				ActionStub:         func() string { return "action" },
				ObjectTypeStub:     func() string { return "object-type" },
				ObjectNameStub:     func() string { return "object-name" },
				TaskIDStub:         func() string { return "task" },
				DeploymentNameStub: func() string { return "deployment" },
				InstanceStub:       func() string { return "instance" },
				ContextStub:        func() map[string]interface{} { return map[string]interface{}{"user": "bosh_z$"} },
				ErrorStub:          func() string { return "" },
			},
			&fakedir.FakeEvent{
				IDStub:        func() string { return "5" },
				TimestampStub: func() time.Time { return time.Date(2090, time.November, 10, 23, 0, 0, 0, time.UTC) },

				UserStub: func() string { return "user2" },

				ActionStub:         func() string { return "action2" },
				ObjectTypeStub:     func() string { return "object-type2" },
				ObjectNameStub:     func() string { return "object-name2" },
				TaskIDStub:         func() string { return "task2" },
				DeploymentNameStub: func() string { return "deployment2" },
				InstanceStub:       func() string { return "instance2" },
				ContextStub:        func() map[string]interface{} { return make(map[string]interface{}) },
				ErrorStub:          func() string { return "some-error" },
			},
		}
	})

	Describe("Run", func() {
		var (
			opts EventsOpts
		)

		It("lists events", func() {
			director.EventsReturns(events, nil)

			err := command.Run(opts)
			Expect(err).ToNot(HaveOccurred())

			Expect(director.EventsArgsForCall(0)).To(Equal(boshdir.EventsFilter{}))

			Expect(ui.Table).To(Equal(boshtbl.Table{
				Content: "events",

				Header: []boshtbl.Header{
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
				},

				Rows: [][]boshtbl.Value{
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
						boshtbl.NewValueInterface(map[string]interface{}{"user": "bosh_z$"}),
						boshtbl.NewValueString(""),
					},
					{
						boshtbl.NewValueString("5"),
						boshtbl.NewValueTime(time.Date(2090, time.November, 10, 23, 0, 0, 0, time.UTC)),
						boshtbl.NewValueString("user2"),
						boshtbl.NewValueString("action2"),
						boshtbl.NewValueString("object-type2"),
						boshtbl.NewValueString("object-name2"),
						boshtbl.NewValueString("task2"),
						boshtbl.NewValueString("deployment2"),
						boshtbl.NewValueString("instance2"),
						boshtbl.NewValueInterface(map[string]interface{}{}),
						boshtbl.NewValueString("some-error"),
					},
				},
			}))
		})

		It("filters events based on options", func() {
			opts.BeforeID = "0"
			opts.Before = time.Date(2050, time.November, 10, 23, 0, 0, 0, time.UTC).String()
			opts.After = time.Date(3055, time.November, 10, 23, 0, 0, 0, time.UTC).String()
			opts.Deployment = "deployment"
			opts.Task = "task"
			opts.Instance = "instance2"
			opts.User = "user2"
			opts.Action = "action2"
			opts.ObjectName = "object-name2"
			opts.ObjectType = "object-type2"

			director.EventsReturns(nil, nil)

			err := command.Run(opts)
			Expect(err).ToNot(HaveOccurred())

			Expect(director.EventsArgsForCall(0)).To(Equal(boshdir.EventsFilter{
				BeforeID:   "0",
				Before:     time.Date(2050, time.November, 10, 23, 0, 0, 0, time.UTC).String(),
				After:      time.Date(3055, time.November, 10, 23, 0, 0, 0, time.UTC).String(),
				Deployment: "deployment",
				Task:       "task",
				Instance:   "instance2",
				User:       "user2",
				Action:     "action2",
				ObjectName: "object-name2",
				ObjectType: "object-type2",
			}))
		})

		It("returns error if events cannot be retrieved", func() {
			director.EventsReturns(nil, errors.New("fake-err"))

			err := command.Run(opts)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
