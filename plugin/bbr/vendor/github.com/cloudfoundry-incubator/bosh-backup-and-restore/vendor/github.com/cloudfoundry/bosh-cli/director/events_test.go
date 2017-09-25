package director_test

import (
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/cloudfoundry/bosh-cli/director"
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

	Describe("Events", func() {
		It("returns events", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/events", ""),
					ghttp.RespondWith(http.StatusOK, `[
						{
						  "id": "1",
						  "timestamp": 1440318199,
						  "user": "fake-user-1",
						  "action": "fake-action",
						  "object_type": "fake-object-type",
						  "object_name": "fake-object-name",
						  "task": "fake-task",
						  "deployment": "fake-deployment",
						  "instance": "fake-instance",
						  "context": {"fake-context-key":"fake-context-value"}
						},
						{
						  "id": "2",
						  "parent_id": "1",
						  "timestamp": 1440318200,
						  "user": "fake-user-2",
						  "action": "fake-action-2",
						  "object_type": "fake-object-type-2",
						  "object_name": "fake-object-name-2",
						  "task": "fake-task-2",
						  "deployment": "fake-deployment-2",
						  "instance": "fake-instance-2"
						}
					]`),
				),
			)

			events, err := director.Events(EventsFilter{})
			Expect(err).ToNot(HaveOccurred())

			Expect(events[0].ID()).To(Equal("1"))
			Expect(events[0].Timestamp()).To(Equal(time.Date(2015, time.August, 23, 8, 23, 19, 0, time.UTC)))
			Expect(events[0].User()).To(Equal("fake-user-1"))
			Expect(events[0].Action()).To(Equal("fake-action"))
			Expect(events[0].ObjectType()).To(Equal("fake-object-type"))
			Expect(events[0].ObjectName()).To(Equal("fake-object-name"))
			Expect(events[0].TaskID()).To(Equal("fake-task"))
			Expect(events[0].DeploymentName()).To(Equal("fake-deployment"))
			Expect(events[0].Instance()).To(Equal("fake-instance"))
			Expect(events[0].Context()).To(Equal(map[string]interface{}{"fake-context-key": "fake-context-value"}))

			Expect(events[1].ID()).To(Equal("2"))
			Expect(events[1].ParentID()).To(Equal("1"))
			Expect(events[1].Timestamp()).To(Equal(time.Date(2015, time.August, 23, 8, 23, 20, 0, time.UTC)))
			Expect(events[1].User()).To(Equal("fake-user-2"))
			Expect(events[1].Action()).To(Equal("fake-action-2"))
			Expect(events[1].ObjectType()).To(Equal("fake-object-type-2"))
			Expect(events[1].ObjectName()).To(Equal("fake-object-name-2"))
			Expect(events[1].TaskID()).To(Equal("fake-task-2"))
			Expect(events[1].DeploymentName()).To(Equal("fake-deployment-2"))
			Expect(events[1].Instance()).To(Equal("fake-instance-2"))
			Expect(events[1].Context()).To(BeNil())
		})

		It("returns event with error", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/events", ""),
					ghttp.RespondWith(http.StatusOK, `[
					  {
						"id": "3",
						"timestamp": 1455635708,
						"user": "admin",
						"action": "rename",
						"error": "Something went wrong",
						"object_type": "deployment",
						"object_name": "depl1",
						"task": "6",
						"context": {"new name": "depl2"}
					  }
					]`),
				),
			)

			events, err := director.Events(EventsFilter{})
			Expect(err).ToNot(HaveOccurred())

			Expect(events[0].Error()).To(Equal("Something went wrong"))
		})

		It("filters events based on 'before-id' option", func() {
			opts := EventsFilter{BeforeID: "3"}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/events", "before_id=3"),
					ghttp.RespondWith(http.StatusOK, `[]`),
				),
			)

			_, err := director.Events(opts)
			Expect(err).ToNot(HaveOccurred())
		})

		It("filters events based on 'before' option", func() {
			opts := EventsFilter{Before: "1440318200"}
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/events", "before_time=1440318200"),
					ghttp.RespondWith(http.StatusOK, `[]`),
				),
			)

			_, err := director.Events(opts)
			Expect(err).ToNot(HaveOccurred())
		})

		It("filters events based on 'after' option", func() {
			opts := EventsFilter{After: "1440318200"}
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/events", "after_time=1440318200"),
					ghttp.RespondWith(http.StatusOK, `[]`),
				),
			)

			_, err := director.Events(opts)
			Expect(err).ToNot(HaveOccurred())
		})

		It("filters events based on 'deployment' option", func() {
			opts := EventsFilter{Deployment: "fake-deployment-2"}
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/events", "deployment=fake-deployment-2"),
					ghttp.RespondWith(http.StatusOK, `[]`),
				),
			)

			_, err := director.Events(opts)

			Expect(err).ToNot(HaveOccurred())
		})

		It("filters events based on 'task' option", func() {
			opts := EventsFilter{Task: "fake-task"}
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/events", "task=fake-task"),
					ghttp.RespondWith(http.StatusOK, `[]`),
				),
			)

			_, err := director.Events(opts)
			Expect(err).ToNot(HaveOccurred())
		})

		It("filters events based on 'instance' option", func() {
			opts := EventsFilter{Instance: "fake-instance-2"}
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/events", "instance=fake-instance-2"),
					ghttp.RespondWith(http.StatusOK, `[]`),
				),
			)

			_, err := director.Events(opts)
			Expect(err).ToNot(HaveOccurred())
		})

		It("filters events based on 'user' option", func() {
			opts := EventsFilter{User: "user2"}
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/events", "user=user2"),
					ghttp.RespondWith(http.StatusOK, `[]`),
				),
			)

			_, err := director.Events(opts)
			Expect(err).ToNot(HaveOccurred())
		})

		It("filters events based on 'action' option", func() {
			opts := EventsFilter{Action: "action2"}
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/events", "action=action2"),
					ghttp.RespondWith(http.StatusOK, `[]`),
				),
			)

			_, err := director.Events(opts)
			Expect(err).ToNot(HaveOccurred())
		})

		It("filters events based on 'object-type' option", func() {
			opts := EventsFilter{ObjectType: "object-type2"}
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/events", "object_type=object-type2"),
					ghttp.RespondWith(http.StatusOK, `[]`),
				),
			)

			_, err := director.Events(opts)
			Expect(err).ToNot(HaveOccurred())
		})

		It("filters events based on 'object-id' option", func() {
			opts := EventsFilter{ObjectName: "object-name2"}
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/events", "object_name=object-name2"),
					ghttp.RespondWith(http.StatusOK, `[]`),
				),
			)

			_, err := director.Events(opts)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns a single event based on multiple options", func() {
			opts := EventsFilter{
				Instance:   "fake-instance-2",
				Deployment: "fake-deployment-2",
			}
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/events", "instance=fake-instance-2&deployment=fake-deployment-2"),
					ghttp.RespondWith(http.StatusOK, `[]`),
				),
			)

			_, err := director.Events(opts)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if response is non-200", func() {
			AppendBadRequest(ghttp.VerifyRequest("GET", "/events", ""), server)

			_, err := director.Events(EventsFilter{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Finding events: Director responded with non-successful status code"))
		})

		It("returns error if response cannot be unmarshalled", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/events", ""),
					ghttp.RespondWith(http.StatusOK, ``),
				),
			)

			_, err := director.Events(EventsFilter{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Finding events: Unmarshaling Director response"))
		})

		It("returns event", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/events/1", ""),
					ghttp.RespondWith(http.StatusOK, `
						{
						  "id": "1",
						  "timestamp": 1440318199,
						  "user": "fake-user-1",
						  "action": "fake-action",
						  "object_type": "fake-object-type",
						  "object_name": "fake-object-name",
						  "task": "fake-task",
						  "deployment": "fake-deployment",
						  "instance": "fake-instance",
						  "context": {"fake-context-key":"fake-context-value"}
						}
					`),
				),
			)

			event, err := director.Event("1")
			Expect(err).ToNot(HaveOccurred())

			Expect(event.ID()).To(Equal("1"))
			Expect(event.Timestamp()).To(Equal(time.Date(2015, time.August, 23, 8, 23, 19, 0, time.UTC)))
			Expect(event.User()).To(Equal("fake-user-1"))
			Expect(event.Action()).To(Equal("fake-action"))
			Expect(event.ObjectType()).To(Equal("fake-object-type"))
			Expect(event.ObjectName()).To(Equal("fake-object-name"))
			Expect(event.TaskID()).To(Equal("fake-task"))
			Expect(event.DeploymentName()).To(Equal("fake-deployment"))
			Expect(event.Instance()).To(Equal("fake-instance"))
			Expect(event.Context()).To(Equal(map[string]interface{}{"fake-context-key": "fake-context-value"}))
		})
	})
})
