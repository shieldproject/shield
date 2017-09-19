package director_test

import (
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/cloudfoundry/bosh-cli/director"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
)

var _ = Describe("Director", func() {
	var (
		director Director
		server   *ghttp.Server
	)

	BeforeEach(func() {
		director, server = BuildServer()
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("CurrentTasks", func() {
		It("returns tasks", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks", "state=processing,cancelling,queued&verbose=1"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[
	{
		"id": 165,
		"started_at": 1440318199,
		"timestamp": 1440318200,
		"state": "state1",
		"user": "user1",
		"deployment": "deployment1",
		"description": "desc1",
		"result": "result1"
	},
	{
		"id": 166,
		"started_at": 1440318199,
		"timestamp": 1440318200,
		"state": "state2",
		"user": "user2",
		"deployment": "deployment2",
		"description": "desc2",
		"result": "result2"
	}
]`),
				),
			)

			tasks, err := director.CurrentTasks(TasksFilter{})
			Expect(err).ToNot(HaveOccurred())
			Expect(tasks).To(HaveLen(2))

			Expect(tasks[0].ID()).To(Equal(165))
			Expect(tasks[0].StartedAt()).To(Equal(time.Date(2015, time.August, 23, 8, 23, 19, 0, time.UTC)))
			Expect(tasks[0].LastActivityAt()).To(Equal(time.Date(2015, time.August, 23, 8, 23, 20, 0, time.UTC)))
			Expect(tasks[0].State()).To(Equal("state1"))
			Expect(tasks[0].User()).To(Equal("user1"))
			Expect(tasks[0].DeploymentName()).To(Equal("deployment1"))
			Expect(tasks[0].Description()).To(Equal("desc1"))
			Expect(tasks[0].Result()).To(Equal("result1"))

			Expect(tasks[1].ID()).To(Equal(166))
			Expect(tasks[1].StartedAt()).To(Equal(time.Date(2015, time.August, 23, 8, 23, 19, 0, time.UTC)))
			Expect(tasks[1].LastActivityAt()).To(Equal(time.Date(2015, time.August, 23, 8, 23, 20, 0, time.UTC)))
			Expect(tasks[1].State()).To(Equal("state2"))
			Expect(tasks[1].User()).To(Equal("user2"))
			Expect(tasks[1].DeploymentName()).To(Equal("deployment2"))
			Expect(tasks[1].Description()).To(Equal("desc2"))
			Expect(tasks[1].Result()).To(Equal("result2"))
		})

		It("includes all tasks when requested", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks", "state=processing,cancelling,queued&verbose=2"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, "[]"),
				),
			)

			_, err := director.CurrentTasks(TasksFilter{All: true})
			Expect(err).ToNot(HaveOccurred())
		})

		It("includes tasks for specific deployment when requested", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks", "state=processing,cancelling,queued&verbose=1&deployment=deployment"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, "[]"),
				),
			)

			_, err := director.CurrentTasks(TasksFilter{Deployment: "deployment"})
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("GET", "/tasks"), server)

			_, err := director.CurrentTasks(TasksFilter{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Finding current tasks: Director responded with non-successful status code"))
		})

		It("returns error if response cannot be unmarshalled", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks"),
					ghttp.RespondWith(http.StatusOK, ``),
				),
			)

			_, err := director.CurrentTasks(TasksFilter{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Finding current tasks: Unmarshaling Director response"))
		})
	})

	Describe("RecentTasks", func() {
		It("returns tasks", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks", "limit=10&verbose=1"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[
	{
		"id": 165,
		"started_at": 1440318199,
		"timestamp": 1440318200,
		"state": "state1",
		"user": "user1",
		"deployment": "deployment1",
		"description": "desc1",
		"result": "result1"
	},
	{
		"id": 166,
		"started_at": 1440318199,
		"timestamp": 1440318200,
		"state": "state2",
		"user": "user2",
		"deployment": "deployment2",
		"description": "desc2",
		"result": "result2"
	}
]`),
				),
			)

			tasks, err := director.RecentTasks(10, TasksFilter{})
			Expect(err).ToNot(HaveOccurred())
			Expect(tasks).To(HaveLen(2))

			Expect(tasks[0].ID()).To(Equal(165))
			Expect(tasks[0].StartedAt()).To(Equal(time.Date(2015, time.August, 23, 8, 23, 19, 0, time.UTC)))
			Expect(tasks[0].LastActivityAt()).To(Equal(time.Date(2015, time.August, 23, 8, 23, 20, 0, time.UTC)))
			Expect(tasks[0].State()).To(Equal("state1"))
			Expect(tasks[0].User()).To(Equal("user1"))
			Expect(tasks[0].DeploymentName()).To(Equal("deployment1"))
			Expect(tasks[0].Description()).To(Equal("desc1"))
			Expect(tasks[0].Result()).To(Equal("result1"))

			Expect(tasks[1].ID()).To(Equal(166))
			Expect(tasks[1].StartedAt()).To(Equal(time.Date(2015, time.August, 23, 8, 23, 19, 0, time.UTC)))
			Expect(tasks[1].LastActivityAt()).To(Equal(time.Date(2015, time.August, 23, 8, 23, 20, 0, time.UTC)))
			Expect(tasks[1].State()).To(Equal("state2"))
			Expect(tasks[1].User()).To(Equal("user2"))
			Expect(tasks[1].DeploymentName()).To(Equal("deployment2"))
			Expect(tasks[1].Description()).To(Equal("desc2"))
			Expect(tasks[1].Result()).To(Equal("result2"))
		})

		It("includes all tasks when requested", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks", "limit=10&verbose=2"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, "[]"),
				),
			)

			_, err := director.RecentTasks(10, TasksFilter{All: true})
			Expect(err).ToNot(HaveOccurred())
		})

		It("includes tasks for specific deployment when requested", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks", "limit=10&verbose=1&deployment=deployment"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, "[]"),
				),
			)

			_, err := director.RecentTasks(10, TasksFilter{Deployment: "deployment"})
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("GET", "/tasks"), server)

			_, err := director.RecentTasks(10, TasksFilter{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Finding recent tasks: Director responded with non-successful status code"))
		})

		It("returns error if response cannot be unmarshalled", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks"),
					ghttp.RespondWith(http.StatusOK, ``),
				),
			)

			_, err := director.RecentTasks(10, TasksFilter{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Finding recent tasks: Unmarshaling Director response"))
		})
	})

	Describe("FindTask", func() {
		It("returns task", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `{
	"id": 123,
	"started_at": 1440318199,
	"timestamp": 1440318200,
	"state": "state1",
	"user": "user1",
	"deployment": "deployment1",
	"description": "desc1",
	"result": "result1",
	"context_id": "context_id1"
}`),
				),
			)

			task, err := director.FindTask(123)
			Expect(err).ToNot(HaveOccurred())

			Expect(task.ID()).To(Equal(123))
			Expect(task.StartedAt()).To(Equal(time.Date(2015, time.August, 23, 8, 23, 19, 0, time.UTC)))
			Expect(task.LastActivityAt()).To(Equal(time.Date(2015, time.August, 23, 8, 23, 20, 0, time.UTC)))
			Expect(task.State()).To(Equal("state1"))
			Expect(task.IsError()).To(BeFalse())
			Expect(task.User()).To(Equal("user1"))
			Expect(task.DeploymentName()).To(Equal("deployment1"))
			Expect(task.Description()).To(Equal("desc1"))
			Expect(task.Result()).To(Equal("result1"))
			Expect(task.ContextID()).To(Equal("context_id1"))
		})

		for _, state := range []string{"error", "timeout", "cancelled"} {
			state := state

			It(fmt.Sprintf("returns task in error state when state is '%s'", state), func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/tasks/123"),
						ghttp.VerifyBasicAuth("username", "password"),
						ghttp.RespondWith(http.StatusOK, fmt.Sprintf(`{"state":"%s"}`, state)),
					),
				)

				task, err := director.FindTask(123)
				Expect(err).ToNot(HaveOccurred())
				Expect(task.IsError()).To(BeTrue())
			})
		}

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("GET", "/tasks/123"), server)

			_, err := director.FindTask(123)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Finding task '123': Director responded with non-successful status code"))
		})

		It("returns error if response cannot be unmarshalled", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks/123"),
					ghttp.RespondWith(http.StatusOK, ``),
				),
			)

			_, err := director.FindTask(123)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Finding task '123': Unmarshaling Director response"))
		})
	})

	Describe("FindTasksByContextId", func() {
		It("returns task", func() {
			contextId := "example-context-id"
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/tasks", fmt.Sprintf("context_id=%s", contextId)),
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.RespondWith(http.StatusOK, `[{
	"id": 123,
	"started_at": 1440318199,
	"timestamp": 1440318200,
	"state": "state1",
	"user": "user1",
	"deployment": "deployment1",
	"description": "desc1",
	"result": "result1",
	"context_id": "`+contextId+`"
}]`),
				),
			)

			tasks, err := director.FindTasksByContextId(contextId)
			Expect(err).ToNot(HaveOccurred())
			Expect(tasks).To(HaveLen(1))
			Expect(tasks[0].ID()).To(Equal(123))
			Expect(tasks[0].ContextID()).To(Equal(contextId))
		})
	})
})

var _ = Describe("Task", func() {
	var (
		director Director
		task     Task
		server   *ghttp.Server
	)

	BeforeEach(func() {
		director, server = BuildServer()

		var err error

		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/tasks/123"),
				ghttp.RespondWith(http.StatusOK, `{"id":123}`),
			),
		)

		task, err = director.FindTask(123)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("TaskOutput", func() {
		var (
			reporter *fakedir.FakeTaskReporter
		)

		BeforeEach(func() {
			reporter = &fakedir.FakeTaskReporter{}
		})

		types := map[string]func(Task) error{
			"event":  func(t Task) error { return task.EventOutput(reporter) },
			"cpi":    func(t Task) error { return task.CPIOutput(reporter) },
			"debug":  func(t Task) error { return task.DebugOutput(reporter) },
			"result": func(t Task) error { return task.ResultOutput(reporter) },
		}

		for type_, typeFunc := range types {
			type_ := type_
			typeFunc := typeFunc

			It(fmt.Sprintf("reports task '%s' output", type_), func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/tasks/123"),
						ghttp.VerifyBasicAuth("username", "password"),
						ghttp.RespondWith(http.StatusOK, `{"id":123, "state":"done"}`),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/tasks/123/output", fmt.Sprintf("type=%s", type_)),
						ghttp.VerifyBasicAuth("username", "password"),
						ghttp.RespondWith(http.StatusOK, "chunk"),
					),
				)

				Expect(typeFunc(task)).ToNot(HaveOccurred())

				taskID := reporter.TaskStartedArgsForCall(0)
				Expect(taskID).To(Equal(123))

				taskID, chunk := reporter.TaskOutputChunkArgsForCall(0)
				Expect(taskID).To(Equal(123))
				Expect(chunk).To(Equal([]byte("chunk")))

				taskID, state := reporter.TaskFinishedArgsForCall(0)
				Expect(taskID).To(Equal(123))
				Expect(state).To(Equal("done"))
			})

			It(fmt.Sprintf("returns error if task '%s' response is non-200", type_), func() {
				AppendBadRequest(ghttp.VerifyRequest("GET", "/tasks/123"), server)

				err := typeFunc(task)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Capturing task '123' output"))
			})
		}
	})

	Describe("Cancel", func() {
		It("cancels task", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("DELETE", "/task/123"),
					ghttp.RespondWith(http.StatusOK, ``),
				),
			)

			Expect(task.Cancel()).ToNot(HaveOccurred())
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("DELETE", "/task/123"), server)

			err := task.Cancel()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Cancelling task '123': Director responded with non-successful status code"))
		})
	})
})
