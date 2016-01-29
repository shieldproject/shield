package supervisor_test

import (
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/starkandwayne/goutils/log"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	. "github.com/starkandwayne/shield/supervisor"
)

var _ = Describe("HTTP API /v1/schedule", func() {
	var API http.Handler
	var resyncChan chan int

	WEEKLY := `51e69607-eb48-4679-afd2-bc3b4c92e691`
	DAILY := `647bc775-b07b-4f87-bb67-d84cccac34a7`

	NIL := `00000000-0000-0000-0000-000000000000`

	BeforeEach(func() {
		log.SetupLogging(log.LogConfig{Type: "file", Level: "EMERG", File: "/dev/null"})
		data, err := Database(
			`INSERT INTO schedules (uuid, name, summary, timespec) VALUES
				("`+WEEKLY+`",
				 "Weekly Backups",
				 "A schedule for weekly bosh-blobs, during normal maintenance windows",
				 "sundays at 3:15am")`,

			`INSERT INTO schedules (uuid, name, summary, timespec) VALUES
				("`+DAILY+`",
				 "Daily Backups",
				 "Use for daily (11-something-at-night) bosh-blobs",
				 "daily at 11:24pm")`,

			`INSERT INTO jobs (uuid, store_uuid, target_uuid, schedule_uuid, retention_uuid)
				VALUES ("abc-def", "`+NIL+`", "`+NIL+`", "`+WEEKLY+`", "`+NIL+`")`,
		)
		Ω(err).ShouldNot(HaveOccurred())

		resyncChan = make(chan int, 1)
		API = ScheduleAPI{
			Data:       data,
			ResyncChan: resyncChan,
		}
	})

	AfterEach(func() {
		close(resyncChan)
		resyncChan = nil
	})

	It("should retrieve all schedules", func() {
		res := GET(API, "/v1/schedules")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "` + DAILY + `",
					"name"    : "Daily Backups",
					"summary" : "Use for daily (11-something-at-night) bosh-blobs",
					"when"    : "daily at 11:24pm"
				},
				{
					"uuid"    : "` + WEEKLY + `",
					"name"    : "Weekly Backups",
					"summary" : "A schedule for weekly bosh-blobs, during normal maintenance windows",
					"when"    : "sundays at 3:15am"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve all schedules with matching names", func() {
		res := GET(API, "/v1/schedules?name=daily")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "` + DAILY + `",
					"name"    : "Daily Backups",
					"summary" : "Use for daily (11-something-at-night) bosh-blobs",
					"when"    : "daily at 11:24pm"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve only unused schedules for ?unused=t", func() {
		res := GET(API, "/v1/schedules?unused=t")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "` + DAILY + `",
					"name"    : "Daily Backups",
					"summary" : "Use for daily (11-something-at-night) bosh-blobs",
					"when"    : "daily at 11:24pm"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve only used schedules for ?unused=f", func() {
		res := GET(API, "/v1/schedules?unused=f")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "` + WEEKLY + `",
					"name"    : "Weekly Backups",
					"summary" : "A schedule for weekly bosh-blobs, during normal maintenance windows",
					"when"    : "sundays at 3:15am"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve only unused weekly schedules for ?unused=t and ?name=weekly", func() {
		res := GET(API, "/v1/schedules?unused=t&name=weekly")
		Ω(res.Body.String()).Should(MatchJSON(`[]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("can retrieve a single schedule by UUID", func() {
		res := GET(API, "/v1/schedule/"+WEEKLY)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{
				"uuid"    : "` + WEEKLY + `",
				"name"    : "Weekly Backups",
				"summary" : "A schedule for weekly bosh-blobs, during normal maintenance windows",
				"when"    : "sundays at 3:15am"
			}`))
	})

	It("returns a 404 for unknown UUIDs", func() {
		res := GET(API, "/v1/schedule/3d650864-0578-42c6-b9c8-883c8a2b1887")
		Ω(res.Code).Should(Equal(404))
	})

	It("can create new schedules", func() {
		res := POST(API, "/v1/schedules", WithJSON(`{
			"name"    : "My New Schedule",
			"summary" : "A new schedule",
			"when"    : "daily 2pm"
		}`))
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchRegexp(`{"ok":"created","uuid":"[a-z0-9-]+"}`))
		Eventually(resyncChan).Should(Receive())
	})

	It("Fails to create invalid timespec schedules", func() {
		res := POST(API, "/v1/schedules", WithJSON(`{
			"name"    : "My New Schedule",
			"summary" : "A new schedule",
			"when"    : "this should fail"
		}`))
		Expect(res.Code).Should(Equal(500))
		Expect(res.Body.String()).Should(Equal(""))
	})

	It("requires the `name' and `when' keys to create a new schedule", func() {
		res := POST(API, "/v1/schedules", "{}")
		Ω(res.Code).Should(Equal(400))
		Ω(res.Body.String()).Should(Equal(`{"missing":["name","when"]}`))
	})

	It("can update existing schedules", func() {
		res := PUT(API, "/v1/schedule/"+DAILY, WithJSON(`{
			"name"    : "Daily Backup Schedule",
			"summary" : "UPDATED!",
			"when"    : "daily at 2:05pm"
		}`))
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{"ok":"updated"}`))
		Eventually(resyncChan).Should(Receive())

		res = GET(API, "/v1/schedules")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "` + DAILY + `",
					"name"    : "Daily Backup Schedule",
					"summary" : "UPDATED!",
					"when"    : "daily at 2:05pm"
				},
				{
					"uuid"    : "` + WEEKLY + `",
					"name"    : "Weekly Backups",
					"summary" : "A schedule for weekly bosh-blobs, during normal maintenance windows",
					"when"    : "sundays at 3:15am"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})
	It("Fails to update schedules with bad timespecs", func() {
		res := PUT(API, "/v1/schedule/"+DAILY, WithJSON(`{
			"name"    : "Daily Backup Schedule",
			"summary" : "UPDATED?",
			"when"    : "this should fail"
		}`))
		Expect(res.Code).Should(Equal(500))
		Expect(res.Body.String()).Should(Equal(""))
	})

	It("requires the `name' and `when' keys to update an existing schedule", func() {
		res := PUT(API, "/v1/schedule/"+DAILY, "{}")
		Ω(res.Code).Should(Equal(400))
		Ω(res.Body.String()).Should(Equal(`{"missing":["name","when"]}`))
	})

	It("can delete unused schedules", func() {
		res := DELETE(API, "/v1/schedule/"+DAILY)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{"ok":"deleted"}`))
		Eventually(resyncChan).Should(Receive())

		res = GET(API, "/v1/schedules")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "` + WEEKLY + `",
					"name"    : "Weekly Backups",
					"summary" : "A schedule for weekly bosh-blobs, during normal maintenance windows",
					"when"    : "sundays at 3:15am"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("refuses to delete a schedule that is in use", func() {
		res := DELETE(API, "/v1/schedule/"+WEEKLY)
		Ω(res.Code).Should(Equal(403))
		Ω(res.Body.String()).Should(Equal(""))
	})

	It("validates JSON payloads", func() {
		JSONValidated(API, "POST", "/v1/schedules")
		JSONValidated(API, "PUT", "/v1/schedule/"+WEEKLY)
	})

	It("ignores other HTTP methods", func() {
		for _, method := range []string{"PUT", "DELETE", "PATCH", "OPTIONS", "TRACE"} {
			NotImplemented(API, method, "/v1/schedules", nil)
		}

		for _, method := range []string{"GET", "HEAD", "POST", "PATCH", "OPTIONS", "TRACE"} {
			NotImplemented(API, method, "/v1/schedules/sub/requests", nil)
			NotImplemented(API, method, "/v1/schedule/sub/requests", nil)
		}
	})

	It("ignores malformed UUIDs", func() {
		for _, id := range []string{"malformed-uuid-01234", "", "(abcdef-01234-56-789)"} {
			NotImplemented(API, "GET", fmt.Sprintf("/v1/schedule/%s", id), nil)
			NotImplemented(API, "PUT", fmt.Sprintf("/v1/schedule/%s", id), nil)
		}
	})
})
