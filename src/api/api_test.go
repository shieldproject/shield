package api_test

import (
	. "api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	"db"

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
		Ω(w.Code).Should(Equal(415))
		Ω(w.Body.String()).Should(Equal(""))
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
				"schedule-uuid-0001-aa",
				"Weekly Backups",
				"A schedule for weekly backups, during normal maintenance windows",
				"sundays at 3:15am")

			database.Exec("new-schedule",
				"schedule-uuid-0002-aa",
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
					"uuid"    : "schedule-uuid-0002-aa",
					"name"    : "Daily Backups",
					"summary" : "Use for daily (11-something-at-night) backups",
					"when"    : "daily at 11:24pm"
				},
				{
					"uuid"    : "schedule-uuid-0001-aa",
					"name"    : "Weekly Backups",
					"summary" : "A schedule for weekly backups, during normal maintenance windows",
					"when"    : "sundays at 3:15am"
				}
			]`)) //expectedJSON))
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

		It("ignores other HTTP methods", func() {
			handler := ScheduleAPI{Data: orm}
			for _, method := range []string{
				"PUT", "DELETE", "PATCH", "OPTIONS", "TRACE",
			} {
				notImplemented(handler, method, "/v1/schedules", nil)
			}
		})

		/* FIXME: handle ?unused=[tf] query string... */

		/* FIXME: write tests for GET /v1/schedule/:uuid */
		/* FIXME: write tests for DELETE /v1/schedule/:uuid */
		/*        (incl. test for delete of an in-use schedule) */
	})
})
