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
	"net/http/httptest"
	"strings"
)

var _ = Describe("HTTP API /v1/retention", func() {
	var orm *db.ORM

	BeforeEach(func() {
		var err error
		orm, err = setupORM(
			`INSERT INTO retention (uuid, name, summary, expiry) VALUES
				("43705750-33b7-4134-a532-ce069abdc08f",
				 "Short-Term Retention",
				 "retain bosh-blobs for two weeks",
				 1209600)`, // 14 days

			`INSERT INTO retention (uuid, name, summary, expiry) VALUES
				("3e783b71-d595-498d-a739-e01fb335098a",
				 "Important Materials",
				 "Keep for 90d",
				 7776000)`, // 90 days

			`INSERT INTO jobs (uuid, retention_uuid) VALUES
				("abc-def",
				 "43705750-33b7-4134-a532-ce069abdc08f")`,
		)
		Ω(err).ShouldNot(HaveOccurred())
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
					"summary" : "retain bosh-blobs for two weeks",
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

	It("can delete unused retention policies", func() {
		handler := RetentionAPI{Data: orm}
		req, _ := http.NewRequest("DELETE", "/v1/retention/3e783b71-d595-498d-a739-e01fb335098a", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		Ω(w.Code).Should(Equal(200))
		Ω(w.Body.String()).Should(Equal(""))

		req, _ = http.NewRequest("GET", "/v1/retention", nil)
		w = httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		Ω(w.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "43705750-33b7-4134-a532-ce069abdc08f",
					"name"    : "Short-Term Retention",
					"summary" : "retain bosh-blobs for two weeks",
					"expires" : 1209600
				}
			]`))
		Ω(w.Code).Should(Equal(200))
	})

	It("refuses to delete a retention policy that is in use", func() {
		handler := RetentionAPI{Data: orm}
		req, _ := http.NewRequest("DELETE", "/v1/retention/43705750-33b7-4134-a532-ce069abdc08f", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		Ω(w.Code).Should(Equal(403))
		Ω(w.Body.String()).Should(Equal(""))
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
})
