package cmd_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
)

var _ = Describe("TaskCmd", func() {
	var (
		eventsRep *fakedir.FakeTaskReporter
		plainRep  *fakedir.FakeTaskReporter
		director  *fakedir.FakeDirector
		command   TaskCmd
	)

	BeforeEach(func() {
		eventsRep = &fakedir.FakeTaskReporter{}
		plainRep = &fakedir.FakeTaskReporter{}
		director = &fakedir.FakeDirector{}
		command = NewTaskCmd(eventsRep, plainRep, director)
	})

	Describe("Run", func() {
		var (
			opts TaskOpts
			task *fakedir.FakeTask
		)

		BeforeEach(func() {
			opts = TaskOpts{}
			task = &fakedir.FakeTask{}
			director.FindTaskReturns(task, nil)
		})

		act := func() error { return command.Run(opts) }

		Context("when id is specified", func() {
			BeforeEach(func() {
				opts.Args.ID = 123
			})

			It("fetches given task", func() {
				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(director.FindTaskCallCount()).To(Equal(1))
				Expect(director.FindTaskArgsForCall(0)).To(Equal(123))
			})

			It("shows task's 'event' output", func() {
				err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(task.EventOutputArgsForCall(0)).To(Equal(eventsRep))

				task.EventOutputStub = func(boshdir.TaskReporter) error { return errors.New("fake-err") }

				err = act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("shows task's 'event' output if requested", func() {
				opts.Event = true

				err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(task.EventOutputArgsForCall(0)).To(Equal(plainRep))

				task.EventOutputStub = func(boshdir.TaskReporter) error { return errors.New("fake-err") }

				err = act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("shows task's 'cpi' output if requested", func() {
				opts.CPI = true

				err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(task.CPIOutputArgsForCall(0)).To(Equal(plainRep))

				task.CPIOutputStub = func(boshdir.TaskReporter) error { return errors.New("fake-err") }

				err = act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("shows task's 'debug' output if requested", func() {
				opts.Debug = true

				err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(task.DebugOutputArgsForCall(0)).To(Equal(plainRep))

				task.DebugOutputStub = func(boshdir.TaskReporter) error { return errors.New("fake-err") }

				err = act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("shows task's 'result' output if requested", func() {
				opts.Result = true

				err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(task.ResultOutputArgsForCall(0)).To(Equal(plainRep))

				task.ResultOutputStub = func(boshdir.TaskReporter) error { return errors.New("fake-err") }

				err = act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("returns error if task cannot be retrieved", func() {
				director.FindTaskReturns(nil, errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})
		})

		Context("when id is not specified", func() {
			BeforeEach(func() {
				task.IDStub = func() int { return 5 }

				tasks := []boshdir.Task{
					task,
					&fakedir.FakeTask{IDStub: func() int { return 4 }},
				}

				director.CurrentTasksReturns(tasks, nil)
			})

			It("shows task's 'event' output", func() {

				err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(task.EventOutputArgsForCall(0)).To(Equal(eventsRep))

				task.EventOutputStub = func(boshdir.TaskReporter) error { return errors.New("fake-err") }

				err = act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("shows task's 'event' output if requested", func() {
				opts.Event = true

				err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(task.EventOutputArgsForCall(0)).To(Equal(plainRep))

				task.EventOutputStub = func(boshdir.TaskReporter) error { return errors.New("fake-err") }

				err = act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("shows task's 'cpi' output if requested", func() {
				opts.CPI = true

				err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(task.CPIOutputArgsForCall(0)).To(Equal(plainRep))

				task.CPIOutputStub = func(boshdir.TaskReporter) error { return errors.New("fake-err") }

				err = act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("shows task's 'debug' output if requested", func() {
				opts.Debug = true

				err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(task.DebugOutputArgsForCall(0)).To(Equal(plainRep))

				task.DebugOutputStub = func(boshdir.TaskReporter) error { return errors.New("fake-err") }

				err = act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("shows task's 'result' output if requested", func() {
				opts.Result = true

				err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(task.ResultOutputArgsForCall(0)).To(Equal(plainRep))

				task.ResultOutputStub = func(boshdir.TaskReporter) error { return errors.New("fake-err") }

				err = act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("filters tasks based on 'all' option", func() {
				err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(director.CurrentTasksArgsForCall(0)).To(Equal(boshdir.TasksFilter{}))

				opts.All = true

				err = act()
				Expect(err).ToNot(HaveOccurred())
				Expect(director.CurrentTasksArgsForCall(1)).To(Equal(boshdir.TasksFilter{All: true}))
			})

			It("returns error if there are no current tasks", func() {
				director.CurrentTasksReturns(nil, nil)

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("No task found"))
			})

			It("returns error if current tasks cannot be retrieved", func() {
				director.CurrentTasksReturns(nil, errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})
		})
	})
})
