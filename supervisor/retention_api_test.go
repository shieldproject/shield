package supervisor_test

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/starkandwayne/shield/supervisor"
	"net/http"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"
)

var _ = Describe("HTTP API /v1/retention", func() {
	var API http.Handler
	var resyncChan chan int

	BeforeEach(func() {
		data, err := Database(
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
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve only unused retention policies for ?unused=t", func() {
		res := GET(API, "/v1/retention?unused=t")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "3e783b71-d595-498d-a739-e01fb335098a",
					"name"    : "Important Materials",
					"summary" : "Keep for 90d",
					"expires" : 7776000
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve only used schedules for ?unused=f", func() {
		res := GET(API, "/v1/retention?unused=f")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "43705750-33b7-4134-a532-ce069abdc08f",
					"name"    : "Short-Term Retention",
					"summary" : "retain bosh-blobs for two weeks",
					"expires" : 1209600
				}
			]`))
		Ω(res.Code).Should(Equal(200))
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

	It("requires the `name' and `when' keys in POST'ed data", func() {
		res := POST(API, "/v1/retention", "{}")
		Ω(res.Code).Should(Equal(400))
	})

	It("can update existing retention policy", func() {
		res := PUT(API, "/v1/retention/43705750-33b7-4134-a532-ce069abdc08f", WithJSON(`{
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
		Ω(res.Code).Should(Equal(200))
	})

	It("requires the `name' field to update an existing retention policy", func() {
		res := PUT(API, "/v1/retention/43705750-33b7-4134-a532-ce069abdc08f", WithJSON(`{
			"summary" : "UPDATED!",
			"expires" : 1209000
		}`))
		Ω(res.Code).Should(Equal(400))
	})

	It("requires the `summary' field to update an existing retention policy", func() {
		res := PUT(API, "/v1/retention/43705750-33b7-4134-a532-ce069abdc08f", WithJSON(`{
			"name"    : "Renamed",
			"expires" : 1209000
		}`))
		Ω(res.Code).Should(Equal(400))
	})

	It("requires a valid `expiry' field of > 3600 to update an existing retention policy", func() {
		res := PUT(API, "/v1/retention/43705750-33b7-4134-a532-ce069abdc08f", WithJSON(`{
			"name"    : "Renamed",
			"summary" : "UPDATED!",
			"expires" : 3599
		}`))
		Ω(res.Code).Should(Equal(400))
	})

	It("can delete unused retention policies", func() {
		res := DELETE(API, "/v1/retention/3e783b71-d595-498d-a739-e01fb335098a")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{"ok":"deleted"}`))
		Eventually(resyncChan).Should(Receive())

		res = GET(API, "/v1/retention")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"    : "43705750-33b7-4134-a532-ce069abdc08f",
					"name"    : "Short-Term Retention",
					"summary" : "retain bosh-blobs for two weeks",
					"expires" : 1209600
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("refuses to delete a retention policy that is in use", func() {
		res := DELETE(API, "/v1/retention/43705750-33b7-4134-a532-ce069abdc08f")
		Ω(res.Code).Should(Equal(403))
		Ω(res.Body.String()).Should(Equal(""))
	})

	It("ignores other HTTP methods", func() {
		for _, method := range []string{"PUT", "DELETE", "PATCH", "OPTIONS", "TRACE"} {
			NotImplemented(API, method, "/v1/retention", nil)
		}

		for _, method := range []string{"GET", "HEAD", "POST", "PATCH", "OPTIONS", "TRACE"} {
			NotImplemented(API, method, "/v1/retention/sub/requests", nil)
			NotImplemented(API, method, "/v1/retention/5981f34c-ef58-4e3b-a91e-428480c68100", nil)
		}
	})

	It("ignores malformed UUIDs", func() {
		for _, id := range []string{"malformed-uuid-01234", "", "(abcdef-01234-56-789)"} {
			NotImplemented(API, "GET", fmt.Sprintf("/v1/retention/%s", id), nil)
			NotImplemented(API, "PUT", fmt.Sprintf("/v1/retention/%s", id), nil)
		}
	})
})
