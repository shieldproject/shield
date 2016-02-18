package supervisor_test

import (
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	. "github.com/starkandwayne/shield/supervisor"
)

var _ = Describe("/v1/tasks API", func() {
	var API http.Handler

	// running
	TASK1 := `9211dc07-6c39-4028-997c-fdc4bbf7c5de`
	JOB1 := `07d8475d-0d62-4b59-9f05-8d2173226ad1`

	// completed
	TASK2 := `b7c35c17-b61b-4541-abea-d93d1837f971`
	ARCHIVE2 := `df4625db-52be-4b55-9ea8-f10214a041bf`
	JOB2 := `0f61e2e1-9293-438f-b2f7-c69a382010ec`

	// canceled
	TASK3 := `524753f0-4f24-4b63-929c-026d20cf07b1`
	JOB3 := `5f04aef7-69cc-40e1-9736-4b3ee4caef50`

	PURGE1 := uuid.NewRandom().String()

	NIL := `00000000-0000-0000-0000-000000000000`

	BeforeEach(func() {

		unixtime := func(t string) string {
			utc, err := time.LoadLocation("UTC")
			Ω(err).ShouldNot(HaveOccurred())
			tempt, err := time.ParseInLocation("2006-01-02 15:04:05", t, utc)
			Ω(err).ShouldNot(HaveOccurred())
			return fmt.Sprintf("%d", tempt.Unix())
		}

		data, err := Database(
			// need a job
			`INSERT INTO jobs (uuid, store_uuid, target_uuid, schedule_uuid, retention_uuid)
				VALUES ("`+JOB1+`", "`+NIL+`", "`+NIL+`", "`+NIL+`", "`+NIL+`")`,
			`INSERT INTO jobs (uuid, store_uuid, target_uuid, schedule_uuid, retention_uuid)
				VALUES ("`+JOB2+`", "`+NIL+`", "`+NIL+`", "`+NIL+`", "`+NIL+`")`,
			`INSERT INTO jobs (uuid, store_uuid, target_uuid, schedule_uuid, retention_uuid)
				VALUES ("`+JOB3+`", "`+NIL+`", "`+NIL+`", "`+NIL+`", "`+NIL+`")`,

			// need an archive
			`INSERT INTO archives (uuid, target_uuid, store_uuid, store_key, taken_at, expires_at)
				VALUES ("`+ARCHIVE2+`", "`+NIL+`", "`+NIL+`", "re-st-ore", 1447900000, 1448000000)`,

			// need a running task
			`INSERT INTO tasks (uuid, owner, op, job_uuid,
				status, requested_at, started_at, log)
				VALUES (
					"`+TASK1+`", "system", "backup", "`+JOB1+`", "running",
					`+unixtime("2015-04-15 06:00:00")+`,
					`+unixtime("2015-04-15 06:00:01")+`,
					"this is the log"
				)`,

			// need a completed task
			`INSERT INTO tasks (uuid, owner, op, job_uuid, archive_uuid,
				status, requested_at, started_at, stopped_at, log)
				VALUES (
					"`+TASK2+`", "joe", "restore", "`+JOB2+`", "`+ARCHIVE2+`", "done",
					`+unixtime("2015-04-10 17:35:00")+`,
					`+unixtime("2015-04-10 17:35:01")+`,
					`+unixtime("2015-04-10 18:19:45")+`,
					"restore complete"
				)`,

			// need a canceled task
			`INSERT INTO tasks (uuid, owner, op, job_uuid,
				status, requested_at, started_at, stopped_at, log)
				VALUES (
					"`+TASK3+`", "joe", "backup", "`+JOB3+`", "canceled",
					`+unixtime("2015-04-18 19:12:03")+`,
					`+unixtime("2015-04-18 19:12:05")+`,
					`+unixtime("2015-04-18 19:13:55")+`,
					"cancel!"
				)`,

			`INSERT INTO tasks (uuid, owner, op, archive_uuid, store_uuid, status, log, requested_at, "job_uuid")
				VALUES ("`+PURGE1+`", "system", "purge", "`+NIL+`", "`+NIL+`", "valid", "", 0, "")`,
		)
		Ω(err).ShouldNot(HaveOccurred())
		API = TaskAPI{Data: data}
	})

	It("should retrieve all tasks, sorted properly", func() {
		res := GET(API, "/v1/tasks")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid": "` + TASK3 + `",
					"owner": "joe",
					"type": "backup",
					"job_uuid": "` + JOB3 + `",
					"archive_uuid": "",
					"status": "canceled",
					"started_at": "2015-04-18 19:12:05",
					"stopped_at": "2015-04-18 19:13:55",
					"log": "cancel!"
				},
				{
					"uuid": "` + TASK1 + `",
					"owner": "system",
					"type": "backup",
					"job_uuid": "` + JOB1 + `",
					"archive_uuid": "",
					"status": "running",
					"started_at": "2015-04-15 06:00:01",
					"stopped_at": "",
					"log": "this is the log"
				},
				{
					"uuid": "` + TASK2 + `",
					"owner": "joe",
					"type": "restore",
					"job_uuid": "` + JOB2 + `",
					"archive_uuid": "` + ARCHIVE2 + `",
					"status": "done",
					"started_at": "2015-04-10 17:35:01",
					"stopped_at": "2015-04-10 18:19:45",
					"log": "restore complete"
				},
				{
					"uuid": "` + PURGE1 + `",
					"owner": "system",
					"type": "purge",
					"job_uuid": "",
					"archive_uuid": "` + NIL + `",
					"status": "valid",
					"started_at": "",
					"stopped_at": "",
					"log": ""
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve tasks based on status", func() {
		res := GET(API, "/v1/tasks?status=done")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid": "` + TASK2 + `",
					"owner": "joe",
					"type": "restore",
					"job_uuid": "` + JOB2 + `",
					"archive_uuid": "` + ARCHIVE2 + `",
					"status": "done",
					"started_at": "2015-04-10 17:35:01",
					"stopped_at": "2015-04-10 18:19:45",
					"log": "restore complete"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should limit qty of retrieved tasks for valid limit", func() {
		res := GET(API, "/v1/tasks?limit=1")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid": "524753f0-4f24-4b63-929c-026d20cf07b1",
					"owner": "joe",
					"type": "backup",
					"job_uuid": "5f04aef7-69cc-40e1-9736-4b3ee4caef50",
					"archive_uuid": "",
					"status": "canceled",
					"started_at": "2015-04-18 19:12:05",
					"stopped_at": "2015-04-18 19:13:55",
					"log": "cancel!"
				}
			]`))
	})

	It("should only retrieved stopped tasks for active=f", func() {
		res := GET(API, "/v1/tasks?active=f")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid": "` + TASK3 + `",
					"owner": "joe",
					"type": "backup",
					"job_uuid": "` + JOB3 + `",
					"archive_uuid": "",
					"status": "canceled",
					"started_at": "2015-04-18 19:12:05",
					"stopped_at": "2015-04-18 19:13:55",
					"log": "cancel!"
				},
				{
					"uuid": "` + TASK2 + `",
					"owner": "joe",
					"type": "restore",
					"job_uuid": "` + JOB2 + `",
					"archive_uuid": "` + ARCHIVE2 + `",
					"status": "done",
					"started_at": "2015-04-10 17:35:01",
					"stopped_at": "2015-04-10 18:19:45",
					"log": "restore complete"
				}
			]`))
	})

	It("should can limit with active=f", func() {
		res := GET(API, "/v1/tasks?active=f&limit=1")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid": "` + TASK3 + `",
					"owner": "joe",
					"type": "backup",
					"job_uuid": "` + JOB3 + `",
					"archive_uuid": "",
					"status": "canceled",
					"started_at": "2015-04-18 19:12:05",
					"stopped_at": "2015-04-18 19:13:55",
					"log": "cancel!"
				}
			]`))
	})

	It("should only retrieved running tasks for active=t", func() {
		res := GET(API, "/v1/tasks?active=t")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid": "` + TASK1 + `",
					"owner": "system",
					"type": "backup",
					"job_uuid": "` + JOB1 + `",
					"archive_uuid": "",
					"status": "running",
					"started_at": "2015-04-15 06:00:01",
					"stopped_at": "",
					"log": "this is the log"
				},
				{
					"uuid": "` + PURGE1 + `",
					"owner": "system",
					"type": "purge",
					"job_uuid": "",
					"archive_uuid": "` + NIL + `",
					"status": "valid",
					"started_at": "",
					"stopped_at": "",
					"log": ""
				}
			]`))
	})

	It("should error for invalid limit value", func() {
		res := GET(API, "/v1/tasks?limit=n")
		Ω(res.Code).Should(Equal(400))
		Ω(res.Body.String()).Should(MatchJSON(`{"error":"invalid limit supplied"}`))

		res = GET(API, "/v1/tasks?limit=-1")
		Ω(res.Code).Should(Equal(400))
		Ω(res.Body.String()).Should(MatchJSON(`{"error":"invalid limit supplied"}`))
	})

	It("can retrieve a single task by UUID", func() {
		res := GET(API, "/v1/task/"+TASK2)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{
				"uuid": "` + TASK2 + `",
				"owner": "joe",
				"type": "restore",
				"job_uuid": "` + JOB2 + `",
				"archive_uuid": "` + ARCHIVE2 + `",
				"status": "done",
				"started_at": "2015-04-10 17:35:01",
				"stopped_at": "2015-04-10 18:19:45",
				"log": "restore complete"
			}`))
	})

	It("returns a 404 for unknown UUIDs", func() {
		res := GET(API, "/v1/task/"+ARCHIVE2) // it's an archive...
		Ω(res.Code).Should(Equal(404))
	})

	It("can cancel tasks", func() {
		res := GET(API, "/v1/tasks?status=running")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid": "` + TASK1 + `",
					"owner": "system",
					"type": "backup",
					"job_uuid": "` + JOB1 + `",
					"archive_uuid": "",
					"status": "running",
					"started_at": "2015-04-15 06:00:01",
					"stopped_at": "",
					"log": "this is the log"
				}
			]`))

		res = DELETE(API, "/v1/task/"+TASK1)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{"ok":"canceled"}`))

		//FIXME: this should change to status=running, and have tests added to make
		// sure what it got back was expected, since it's not failing, despite 'state'
		// not being the right parameter
		res = GET(API, "/v1/tasks?state=running")
		Ω(res.Code).Should(Equal(200))
	})

	It("ignores other HTTP methods", func() {
		for _, method := range []string{"PUT", "DELETE", "PATCH", "OPTIONS", "TRACE"} {
			NotImplemented(API, method, "/v1/tasks", nil)
		}

		for _, method := range []string{"GET", "HEAD", "POST", "PATCH", "OPTIONS", "TRACE"} {
			NotImplemented(API, method, "/v1/tasks/sub/requests", nil)
			NotImplemented(API, method, "/v1/task/sub/requests", nil)
		}
	})

	It("ignores malformed UUIDs", func() {
		for _, id := range []string{"malformed-uuid-01234", "(abcdef-01234-56-789)"} {
			NotImplemented(API, "PUT", fmt.Sprintf("/v1/task/%s", id), nil)
		}
	})
})
