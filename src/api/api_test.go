package api_test

import (
	. "api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	"db"

	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

var _ = Describe("HTTP Rest API", func() {
	setupORM := func() (*db.ORM, *db.DB, error) {
		database := &db.DB{
			Driver: "sqlite3",
			DSN:    ":memory:",
		}

		if err := database.Connect(); err != nil {
			return nil, nil, err
		}

		orm, err := db.NewORM(database)
		if err != nil {
			database.Disconnect()
			return nil, nil, err
		}

		if err := orm.Setup(); err != nil {
			database.Disconnect()
			return nil, nil, err
		}

		return orm, database, nil
	}

	notImplemented := func(h http.Handler, method string, uri string, body io.Reader) {
		req, _ := http.NewRequest(method, uri, body)
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)
		Ω(w.Code).Should(Equal(415),
			fmt.Sprintf("%s %s should elicit HTTP 415 (Not Implemented) response...", method, uri))
		Ω(w.Body.String()).Should(Equal(""),
			fmt.Sprintf("%s %s should have no HTTP Response Body...", method, uri))
	}

	notFound := func(h http.Handler, method string, uri string, body io.Reader) {
		req, _ := http.NewRequest(method, uri, body)
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)
		Ω(w.Code).Should(Equal(404))
		Ω(w.Body.String()).Should(Equal(""))
	}

	Describe("/v1/ping API", func() {
		It("handles GET requests", func() {
			handler := PingAPI{}
			req, _ := http.NewRequest("GET", "/v1/ping", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)
			Ω(w.Code).Should(Equal(200))
			Ω(w.Body.String()).Should(Equal(`{"ok":"pong"}`))
		})

		It("ignores other HTTP methods", func() {
			for _, method := range []string{
				"PUT", "POST", "DELETE", "PATCH", "OPTIONS", "TRACE",
			} {
				notImplemented(PingAPI{}, method, "/v1/ping", nil)
			}
		})

		It("ignores requests not to /v1/ping (sub-URIs)", func() {
			notFound(PingAPI{}, "GET", "/v1/ping/stuff", nil)
			notFound(PingAPI{}, "OPTIONS", "/v1/ping/OPTIONAL/STUFF", nil)
		})
	})

	Describe("/v1/schedule API", func() {
		var database *db.DB
		var orm *db.ORM

		BeforeEach(func() {
			var err error
			orm, database, err = setupORM()
			Ω(err).ShouldNot(HaveOccurred())

			database.Cache("new-schedule", `
				INSERT INTO schedules (uuid, name, summary, timespec) VALUES (?, ?, ?, ?)
			`)

			database.Exec("new-schedule",
				"51e69607-eb48-4679-afd2-bc3b4c92e691",
				"Weekly Backups",
				"A schedule for weekly backups, during normal maintenance windows",
				"sundays at 3:15am")

			database.Exec("new-schedule",
				"647bc775-b07b-4f87-bb67-d84cccac34a7",
				"Daily Backups",
				"Use for daily (11-something-at-night) backups",
				"daily at 11:24pm")
		})

		It("should retrieve all schedules", func() {
			handler := ScheduleAPI{Data: orm}
			req, _ := http.NewRequest("GET", "/v1/schedules", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)
			Ω(w.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "647bc775-b07b-4f87-bb67-d84cccac34a7",
					"name"    : "Daily Backups",
					"summary" : "Use for daily (11-something-at-night) backups",
					"when"    : "daily at 11:24pm"
				},
				{
					"uuid"    : "51e69607-eb48-4679-afd2-bc3b4c92e691",
					"name"    : "Weekly Backups",
					"summary" : "A schedule for weekly backups, during normal maintenance windows",
					"when"    : "sundays at 3:15am"
				}
			]`))
			Ω(w.Code).Should(Equal(200))
		})

		It("can create new schedules", func() {
			handler := ScheduleAPI{Data: orm}
			req, _ := http.NewRequest("POST", "/v1/schedules",
				strings.NewReader(
					`{"name" :"My New Schedule","summary":"A new schedule","when":"daily 2pm"}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)
			Ω(w.Code).Should(Equal(200))
			Ω(w.Body.String()).Should(MatchRegexp(`{"ok":"created","uuid":"[a-z0-9-]+"}`))
		})

		It("requires the `name' and `when' keys in POST'ed data", func() {
			handler := ScheduleAPI{Data: orm}
			req, _ := http.NewRequest("POST", "/v1/schedules", strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)
			Ω(w.Code).Should(Equal(400))
		})

		It("can update existing schedules", func() {
			handler := ScheduleAPI{Data: orm}
			req, _ := http.NewRequest("PUT", "/v1/schedule/647bc775-b07b-4f87-bb67-d84cccac34a7",
				strings.NewReader(
					`{"name" :"Daily Backup Schedule","summary":"UPDATED!","when":"daily at 2:05pm"}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)
			Ω(w.Code).Should(Equal(200))
			Ω(w.Body.String()).Should(MatchJSON(`{"ok":"updated","uuid":"647bc775-b07b-4f87-bb67-d84cccac34a7"}`))

			req, _ = http.NewRequest("GET", "/v1/schedules", nil)
			w = httptest.NewRecorder()

			handler.ServeHTTP(w, req)
			Ω(w.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "647bc775-b07b-4f87-bb67-d84cccac34a7",
					"name"    : "Daily Backup Schedule",
					"summary" : "UPDATED!",
					"when"    : "daily at 2:05pm"
				},
				{
					"uuid"    : "51e69607-eb48-4679-afd2-bc3b4c92e691",
					"name"    : "Weekly Backups",
					"summary" : "A schedule for weekly backups, during normal maintenance windows",
					"when"    : "sundays at 3:15am"
				}
			]`))
			Ω(w.Code).Should(Equal(200))
		})

		It("ignores other HTTP methods", func() {
			handler := ScheduleAPI{Data: orm}
			for _, method := range []string{"PUT", "DELETE", "PATCH", "OPTIONS", "TRACE"} {
				notImplemented(handler, method, "/v1/schedules", nil)
			}

			for _, method := range []string{"GET", "HEAD", "POST", "PATCH", "OPTIONS", "TRACE"} {
				notImplemented(handler, method, "/v1/schedules/sub/requests", nil)
				notImplemented(handler, method, "/v1/schedule/sub/requests", nil)
				notImplemented(handler, method, "/v1/schedule/5981f34c-ef58-4e3b-a91e-428480c68100", nil)
			}
		})

		It("ignores malformed UUIDs", func() {
			handler := ScheduleAPI{Data: orm}
			for _, id := range []string{"malformed-uuid-01234", "", "(abcdef-01234-56-789)"} {
				notImplemented(handler, "GET", fmt.Sprintf("/v1/schedule/%s", id), nil)
				notImplemented(handler, "PUT", fmt.Sprintf("/v1/schedule/%s", id), nil)
			}
		})

		/* FIXME: handle ?unused=[tf] query string... */

		/* FIXME: write tests for DELETE /v1/schedule/:uuid */
		/*        (incl. test for delete of an in-use schedule) */
	})

	Describe("/v1/retention API", func() {
		var database *db.DB
		var orm *db.ORM

		BeforeEach(func() {
			var err error
			orm, database, err = setupORM()
			Ω(err).ShouldNot(HaveOccurred())

			database.Cache("new-policy", `
				INSERT INTO retention (uuid, name, summary, expiry) VALUES (?, ?, ?, ?)
			`)

			database.Exec("new-policy",
				"43705750-33b7-4134-a532-ce069abdc08f",
				"Short-Term Retention",
				"retain backups for two weeks",
				86400*14)

			database.Exec("new-policy",
				"3e783b71-d595-498d-a739-e01fb335098a",
				"Important Materials",
				"Keep for 90d",
				86400*90)
		})

		It("should retrieve all retention policies", func() {
			handler := RetentionAPI{Data: orm}
			req, _ := http.NewRequest("GET", "/v1/retention", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)
			Ω(w.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "3e783b71-d595-498d-a739-e01fb335098a",
					"name"    : "Important Materials",
					"summary" : "Keep for 90d",
					"expires" : 7776000
				},
				{
					"uuid"    : "43705750-33b7-4134-a532-ce069abdc08f",
					"name"    : "Short-Term Retention",
					"summary" : "retain backups for two weeks",
					"expires" : 1209600
				}
			]`))
			Ω(w.Code).Should(Equal(200))
		})

		It("can create new retention policies", func() {
			handler := RetentionAPI{Data: orm}
			req, _ := http.NewRequest("POST", "/v1/retention",
				strings.NewReader(
					`{"name" :"New Policy","summary":"A new one","expires":86401}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)
			Ω(w.Code).Should(Equal(200))
			Ω(w.Body.String()).Should(MatchRegexp(`{"ok":"created","uuid":"[a-z0-9-]+"}`))
		})

		It("requires the `name' and `when' keys in POST'ed data", func() {
			handler := RetentionAPI{Data: orm}
			req, _ := http.NewRequest("POST", "/v1/retention", strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)
			Ω(w.Code).Should(Equal(400))
		})

		It("can update existing retention policy", func() {
			handler := RetentionAPI{Data: orm}
			req, _ := http.NewRequest("PUT", "/v1/retention/43705750-33b7-4134-a532-ce069abdc08f",
				strings.NewReader(
					`{"name" :"Renamed","summary":"UPDATED!","expires":1209000}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)
			Ω(w.Code).Should(Equal(200))
			Ω(w.Body.String()).Should(MatchJSON(`{"ok":"updated","uuid":"43705750-33b7-4134-a532-ce069abdc08f"}`))

			req, _ = http.NewRequest("GET", "/v1/retention", nil)
			w = httptest.NewRecorder()

			handler.ServeHTTP(w, req)
			Ω(w.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "3e783b71-d595-498d-a739-e01fb335098a",
					"name"    : "Important Materials",
					"summary" : "Keep for 90d",
					"expires" : 7776000
				},
				{
					"uuid"    : "43705750-33b7-4134-a532-ce069abdc08f",
					"name"    : "Renamed",
					"summary" : "UPDATED!",
					"expires" : 1209000
				}
			]`))
			Ω(w.Code).Should(Equal(200))
		})

		It("ignores other HTTP methods", func() {
			handler := RetentionAPI{Data: orm}
			for _, method := range []string{"PUT", "DELETE", "PATCH", "OPTIONS", "TRACE"} {
				notImplemented(handler, method, "/v1/retention", nil)
			}

			for _, method := range []string{"GET", "HEAD", "POST", "PATCH", "OPTIONS", "TRACE"} {
				notImplemented(handler, method, "/v1/retention/sub/requests", nil)
				notImplemented(handler, method, "/v1/retention/sub/requests", nil)
				notImplemented(handler, method, "/v1/retention/5981f34c-ef58-4e3b-a91e-428480c68100", nil)
			}
		})

		It("ignores malformed UUIDs", func() {
			handler := RetentionAPI{Data: orm}
			for _, id := range []string{"malformed-uuid-01234", "", "(abcdef-01234-56-789)"} {
				notImplemented(handler, "GET", fmt.Sprintf("/v1/retention/%s", id), nil)
				notImplemented(handler, "PUT", fmt.Sprintf("/v1/retention/%s", id), nil)
			}
		})

		/* FIXME: handle ?unused=[tf] query string... */

		/* FIXME: write tests for DELETE /v1/retention/:uuid */
		/*        (incl. test for delete of an in-use retention policy) */
	})
})
