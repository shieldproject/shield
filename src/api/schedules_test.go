package api_test

import (
	. "api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	"db"

	"fmt"
	"net/http"
)

var _ = Describe("HTTP API /v1/schedule", func() {
	var orm *db.ORM
	var API http.Handler

	BeforeEach(func() {
		var err error
		orm, err = setupORM(
			`INSERT INTO schedules (uuid, name, summary, timespec) VALUES
				("51e69607-eb48-4679-afd2-bc3b4c92e691",
				 "Weekly Backups",
				 "A schedule for weekly bosh-blobs, during normal maintenance windows",
				 "sundays at 3:15am")`,

			`INSERT INTO schedules (uuid, name, summary, timespec) VALUES
				("647bc775-b07b-4f87-bb67-d84cccac34a7",
				 "Daily Backups",
				 "Use for daily (11-something-at-night) bosh-blobs",
				 "daily at 11:24pm")`,

			`INSERT INTO jobs (uuid, schedule_uuid) VALUES ("abc-def", "51e69607-eb48-4679-afd2-bc3b4c92e691")`,
		)
		Ω(err).ShouldNot(HaveOccurred())

		API = ScheduleAPI{Data: orm}
	})

	It("should retrieve all schedules", func() {
		res := GET(API, "/v1/schedules")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "647bc775-b07b-4f87-bb67-d84cccac34a7",
					"name"    : "Daily Backups",
					"summary" : "Use for daily (11-something-at-night) bosh-blobs",
					"when"    : "daily at 11:24pm"
				},
				{
					"uuid"    : "51e69607-eb48-4679-afd2-bc3b4c92e691",
					"name"    : "Weekly Backups",
					"summary" : "A schedule for weekly bosh-blobs, during normal maintenance windows",
					"when"    : "sundays at 3:15am"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve only unused schedules for ?unused=t", func() {
		res := GET(API, "/v1/schedules?unused=t")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "647bc775-b07b-4f87-bb67-d84cccac34a7",
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
					"uuid"    : "51e69607-eb48-4679-afd2-bc3b4c92e691",
					"name"    : "Weekly Backups",
					"summary" : "A schedule for weekly bosh-blobs, during normal maintenance windows",
					"when"    : "sundays at 3:15am"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("can create new schedules", func() {
		res := POST(API, "/v1/schedules", `{
			"name"    : "My New Schedule",
			"summary" : "A new schedule",
			"when"    : "daily 2pm"
		}`)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchRegexp(`{"ok":"created","uuid":"[a-z0-9-]+"}`))
	})

	It("requires the `name' and `when' keys in POST'ed data", func() {
		res := POST(API, "/v1/schedules", "{}")
		Ω(res.Code).Should(Equal(400))
	})

	It("can update existing schedules", func() {
		res := PUT(API, "/v1/schedule/647bc775-b07b-4f87-bb67-d84cccac34a7", `{
			"name"    : "Daily Backup Schedule",
			"summary" : "UPDATED!",
			"when"    : "daily at 2:05pm"
		}`)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{"ok":"updated","uuid":"647bc775-b07b-4f87-bb67-d84cccac34a7"}`))

		res = GET(API, "/v1/schedules")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "647bc775-b07b-4f87-bb67-d84cccac34a7",
					"name"    : "Daily Backup Schedule",
					"summary" : "UPDATED!",
					"when"    : "daily at 2:05pm"
				},
				{
					"uuid"    : "51e69607-eb48-4679-afd2-bc3b4c92e691",
					"name"    : "Weekly Backups",
					"summary" : "A schedule for weekly bosh-blobs, during normal maintenance windows",
					"when"    : "sundays at 3:15am"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("can delete unused schedules", func() {
		res := DELETE(API, "/v1/schedule/647bc775-b07b-4f87-bb67-d84cccac34a7")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(Equal(""))

		res = GET(API, "/v1/schedules")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "51e69607-eb48-4679-afd2-bc3b4c92e691",
					"name"    : "Weekly Backups",
					"summary" : "A schedule for weekly bosh-blobs, during normal maintenance windows",
					"when"    : "sundays at 3:15am"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("refuses to delete a schedule that is in use", func() {
		res := DELETE(API, "/v1/schedule/51e69607-eb48-4679-afd2-bc3b4c92e691")
		Ω(res.Code).Should(Equal(403))
		Ω(res.Body.String()).Should(Equal(""))
	})

	It("ignores other HTTP methods", func() {
		for _, method := range []string{"PUT", "DELETE", "PATCH", "OPTIONS", "TRACE"} {
			NotImplemented(API, method, "/v1/schedules", nil)
		}

		for _, method := range []string{"GET", "HEAD", "POST", "PATCH", "OPTIONS", "TRACE"} {
			NotImplemented(API, method, "/v1/schedules/sub/requests", nil)
			NotImplemented(API, method, "/v1/schedule/sub/requests", nil)
			NotImplemented(API, method, "/v1/schedule/5981f34c-ef58-4e3b-a91e-428480c68100", nil)
		}
	})

	It("ignores malformed UUIDs", func() {
		for _, id := range []string{"malformed-uuid-01234", "", "(abcdef-01234-56-789)"} {
			NotImplemented(API, "GET", fmt.Sprintf("/v1/schedule/%s", id), nil)
			NotImplemented(API, "PUT", fmt.Sprintf("/v2/schedule/%s", id), nil)
		}
	})
})
