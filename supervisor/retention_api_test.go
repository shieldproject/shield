package supervisor_test

import (
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/starkandwayne/shield/supervisor"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"
)

var _ = Describe("HTTP API /v1/retention", func() {
	var API http.Handler
	var resyncChan chan int

	SHORT := `43705750-33b7-4134-a532-ce069abdc08f`
	LONG := `3e783b71-d595-498d-a739-e01fb335098a`

	NIL := `00000000-0000-0000-0000-000000000000`

	BeforeEach(func() {
		data, err := Database(
			`INSERT INTO retention (uuid, name, summary, expiry) VALUES
				("`+SHORT+`",
				 "Short-Term Retention",
				 "retain bosh-blobs for two weeks",
				 1209600)`, // 14 days

			`INSERT INTO retention (uuid, name, summary, expiry) VALUES
				("`+LONG+`",
				 "Important Materials",
				 "Keep for 90d",
				 7776000)`, // 90 days

			`INSERT INTO jobs (uuid, retention_uuid, schedule_uuid, target_uuid, store_uuid) VALUES
				("abc-def",
				 "`+SHORT+`", "`+NIL+`", "`+NIL+`", "`+NIL+`")`,
		)
		Ω(err).ShouldNot(HaveOccurred())
		resyncChan = make(chan int, 1)
		API = RetentionAPI{
			Data:       data,
			ResyncChan: resyncChan,
		}
	})

	AfterEach(func() {
		close(resyncChan)
		resyncChan = nil
	})

	It("should retrieve all retention policies", func() {
		res := GET(API, "/v1/retention")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "` + LONG + `",
					"name"    : "Important Materials",
					"summary" : "Keep for 90d",
					"expires" : 7776000
				},
				{
					"uuid"    : "` + SHORT + `",
					"name"    : "Short-Term Retention",
					"summary" : "retain bosh-blobs for two weeks",
					"expires" : 1209600
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve all retention policies matching the name 'short'", func() {
		res := GET(API, "/v1/retention?name=short")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "` + SHORT + `",
					"name"    : "Short-Term Retention",
					"summary" : "retain bosh-blobs for two weeks",
					"expires" : 1209600
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve only unused retention policies for ?unused=t", func() {
		res := GET(API, "/v1/retention?unused=t")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "` + LONG + `",
					"name"    : "Important Materials",
					"summary" : "Keep for 90d",
					"expires" : 7776000
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve only unused retention policies named 'short' for ?unused=t and ?name=short", func() {
		res := GET(API, "/v1/retention?unused=t&name=short")
		Ω(res.Body.String()).Should(MatchJSON(`[]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve only used retention policies for ?unused=f", func() {
		res := GET(API, "/v1/retention?unused=f")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "` + SHORT + `",
					"name"    : "Short-Term Retention",
					"summary" : "retain bosh-blobs for two weeks",
					"expires" : 1209600
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("can retrieve a single retention policy by UUID", func() {
		res := GET(API, "/v1/retention/"+SHORT)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{
					"uuid"    : "` + SHORT + `",
					"name"    : "Short-Term Retention",
					"summary" : "retain bosh-blobs for two weeks",
					"expires" : 1209600
			}`))
	})

	It("returns a 404 for unknown UUIDs", func() {
		res := GET(API, "/v1/retention/85612f54-74fa-4897-aafc-01c1f0d3ae2e")
		Ω(res.Code).Should(Equal(404))
	})

	It("can create new retention policies", func() {
		res := POST(API, "/v1/retention", WithJSON(`{
			"name"    : "New Policy",
			"summary" : "A new one",
			"expires" : 86401
		}`))
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchRegexp(`{"ok":"created","uuid":"[a-z0-9-]+"}`))
		Eventually(resyncChan).Should(Receive())
	})

	It("requires the `name' and `expires' keys to create a new policy", func() {
		res := POST(API, "/v1/retention", "{}")
		Ω(res.Code).Should(Equal(400))
		Ω(res.Body.String()).Should(Equal(`{"missing":["name","expires"]}`))
	})

	It("requires a valid `expires' that is > 3600 to create a new retention policy", func() {
		res := POST(API, "/v1/retention", WithJSON(`{
			"name"    : "New Policy",
			"summary" : "Expires way too fast",
			"expires" : 60
		}`))
		Ω(res.Code).Should(Equal(400))
		Ω(res.Body.String()).Should(Equal(`{"Errors":{"expires":"60 is less than 3600"}}`))
	})

	It("can update existing retention policy", func() {
		res := PUT(API, "/v1/retention/"+SHORT, WithJSON(`{
			"name"    : "Renamed",
			"summary" : "UPDATED!",
			"expires" : 1209000
		}`))
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{"ok":"updated"}`))
		Eventually(resyncChan).Should(Receive())

		res = GET(API, "/v1/retention")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "` + LONG + `",
					"name"    : "Important Materials",
					"summary" : "Keep for 90d",
					"expires" : 7776000
				},
				{
					"uuid"    : "` + SHORT + `",
					"name"    : "Renamed",
					"summary" : "UPDATED!",
					"expires" : 1209000
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("requires the `name' and `expires' field to update an existing retention policy", func() {
		res := PUT(API, "/v1/retention/"+SHORT, "{}")
		Ω(res.Code).Should(Equal(400))
		Ω(res.Body.String()).Should(Equal(`{"missing":["name","expires"]}`))
	})

	It("requires a valid `expires' that is > 3600 to update an existing retention policy", func() {
		res := PUT(API, "/v1/retention/"+SHORT, WithJSON(`{
				"name"    : "Policy",
				"summary" : "Expires way too fast",
				"expires" : 60
			}`))
		Ω(res.Code).Should(Equal(400))
		Ω(res.Body.String()).Should(Equal(`{"Errors":{"expires":"60 is less than 3600"}}`))
	})

	It("can delete unused retention policies", func() {
		res := DELETE(API, "/v1/retention/"+LONG)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{"ok":"deleted"}`))
		Eventually(resyncChan).Should(Receive())

		res = GET(API, "/v1/retention")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "` + SHORT + `",
					"name"    : "Short-Term Retention",
					"summary" : "retain bosh-blobs for two weeks",
					"expires" : 1209600
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("refuses to delete a retention policy that is in use", func() {
		res := DELETE(API, "/v1/retention/"+SHORT)
		Ω(res.Code).Should(Equal(403))
		Ω(res.Body.String()).Should(Equal(""))
	})

	It("validates JSON payloads", func() {
		JSONValidated(API, "POST", "/v1/retention")
		JSONValidated(API, "PUT", "/v1/retention/"+SHORT)
	})

	It("ignores other HTTP methods", func() {
		for _, method := range []string{"PUT", "DELETE", "PATCH", "OPTIONS", "TRACE"} {
			NotImplemented(API, method, "/v1/retention", nil)
		}

		for _, method := range []string{"GET", "HEAD", "POST", "PATCH", "OPTIONS", "TRACE"} {
			NotImplemented(API, method, "/v1/retention/sub/requests", nil)
		}
	})

	It("ignores malformed UUIDs", func() {
		for _, id := range []string{"malformed-uuid-01234", "", "(abcdef-01234-56-789)"} {
			NotImplemented(API, "GET", fmt.Sprintf("/v1/retention/%s", id), nil)
			NotImplemented(API, "PUT", fmt.Sprintf("/v1/retention/%s", id), nil)
		}
	})
})
