package supervisor_test

import (
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	. "github.com/starkandwayne/shield/supervisor"
)

var _ = Describe("/v1/stores API", func() {
	var API http.Handler
	var resyncChan chan int

	STORE_REDIS := `66be7c43-6c57-4391-8ea9-e770d6ab5e9e`
	STORE_S3 := `05c3d005-f968-452f-bd59-bee8e79ab982`

	NIL := `00000000-0000-0000-0000-000000000000`

	BeforeEach(func() {
		data, err := Database(
			`INSERT INTO stores (uuid, name, summary, plugin, endpoint) VALUES
				("`+STORE_REDIS+`",
				 "redis-shared",
				 "Shared Redis services for CF",
				 "redis",
				 "<<redis-configuration>>")`,

			`INSERT INTO stores (uuid, name, summary, plugin, endpoint) VALUES
				("`+STORE_S3+`",
				 "s3",
				 "Amazon S3 Blobstore",
				 "s3",
				 "<<s3-configuration>>")`,

			`INSERT INTO jobs (uuid, store_uuid, target_uuid, schedule_uuid, retention_uuid) VALUES
				("abc-def",
				 "`+STORE_S3+`", "`+NIL+`", "`+NIL+`", "`+NIL+`")`,
		)
		Ω(err).ShouldNot(HaveOccurred())
		resyncChan = make(chan int, 1)
		API = StoreAPI{
			Data:       data,
			ResyncChan: resyncChan,
		}
	})

	AfterEach(func() {
		close(resyncChan)
		resyncChan = nil
	})

	It("should retrieve all stores", func() {
		res := GET(API, "/v1/stores")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "` + STORE_REDIS + `",
					"name"     : "redis-shared",
					"summary"  : "Shared Redis services for CF",
					"plugin"   : "redis",
					"endpoint" : "<<redis-configuration>>"
				},
				{
					"uuid"     : "` + STORE_S3 + `",
					"name"     : "s3",
					"summary"  : "Amazon S3 Blobstore",
					"plugin"   : "s3",
					"endpoint" : "<<s3-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve all stores named 'redis'", func() {
		res := GET(API, "/v1/stores?name=redis")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "` + STORE_REDIS + `",
					"name"     : "redis-shared",
					"summary"  : "Shared Redis services for CF",
					"plugin"   : "redis",
					"endpoint" : "<<redis-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve only unused stores ?unused=t", func() {
		res := GET(API, "/v1/stores?unused=t")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "` + STORE_REDIS + `",
					"name"     : "redis-shared",
					"summary"  : "Shared Redis services for CF",
					"plugin"   : "redis",
					"endpoint" : "<<redis-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve only unused stores named s3 for ?unused=t and ?name=s3", func() {
		res := GET(API, "/v1/stores?unused=t&name=s3")
		Ω(res.Body.String()).Should(MatchJSON(`[]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve only used stores for ?unused=f", func() {
		res := GET(API, "/v1/stores?unused=f")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "` + STORE_S3 + `",
					"name"     : "s3",
					"summary"  : "Amazon S3 Blobstore",
					"plugin"   : "s3",
					"endpoint" : "<<s3-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should filter stores by plugin name", func() {
		res := GET(API, "/v1/stores?plugin=redis")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "` + STORE_REDIS + `",
					"name"     : "redis-shared",
					"summary"  : "Shared Redis services for CF",
					"plugin"   : "redis",
					"endpoint" : "<<redis-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))

		res = GET(API, "/v1/stores?plugin=s3")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "` + STORE_S3 + `",
					"name"     : "s3",
					"summary"  : "Amazon S3 Blobstore",
					"plugin"   : "s3",
					"endpoint" : "<<s3-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))

		res = GET(API, "/v1/stores?plugin=enoent")
		Ω(res.Body.String()).Should(MatchJSON(`[]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should filter by combinations of `plugin' and `unused' parameters", func() {
		res := GET(API, "/v1/stores?plugin=s3&unused=f")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "` + STORE_S3 + `",
					"name"     : "s3",
					"summary"  : "Amazon S3 Blobstore",
					"plugin"   : "s3",
					"endpoint" : "<<s3-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))

		res = GET(API, "/v1/stores?plugin=s3&unused=t")
		Ω(res.Body.String()).Should(MatchJSON(`[]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("can retrieve a single store by UUID", func() {
		res := GET(API, "/v1/store/"+STORE_S3)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{
				"uuid"     : "` + STORE_S3 + `",
				"name"     : "s3",
				"summary"  : "Amazon S3 Blobstore",
				"plugin"   : "s3",
				"endpoint" : "<<s3-configuration>>"
			}`))
	})

	It("returns a 404 for unknown UUIDs", func() {
		res := GET(API, "/v1/store/de33cdc2-2502-457b-97d8-1bed423b85ac")
		Ω(res.Code).Should(Equal(404))
	})

	It("can create new stores", func() {
		res := POST(API, "/v1/stores", WithJSON(`{
			"name"     : "New Store",
			"summary"  : "A new one",
			"plugin"   : "s3",
			"endpoint" : "[ENDPOINT]"
		}`))
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchRegexp(`{"ok":"created","uuid":"[a-z0-9-]+"}`))
		Eventually(resyncChan).Should(Receive())
	})

	It("requires the `name', `plugin', and `endpoint' keys to create a new store", func() {
		res := POST(API, "/v1/stores", "{}")
		Ω(res.Code).Should(Equal(400))
		Ω(res.Body.String()).Should(Equal(`{"missing":["name","plugin","endpoint"]}`))
	})

	It("can update existing store", func() {
		res := PUT(API, "/v1/store/"+STORE_REDIS, WithJSON(`{
			"name"     : "Renamed",
			"summary"  : "UPDATED!",
			"plugin"   : "redis",
			"endpoint" : "{NEW-ENDPOINT}"
		}`))
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{"ok":"updated"}`))
		Eventually(resyncChan).Should(Receive())

		res = GET(API, "/v1/stores")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "` + STORE_REDIS + `",
					"name"     : "Renamed",
					"summary"  : "UPDATED!",
					"plugin"   : "redis",
					"endpoint" : "{NEW-ENDPOINT}"
				},
				{
					"uuid"     : "` + STORE_S3 + `",
					"name"     : "s3",
					"summary"  : "Amazon S3 Blobstore",
					"plugin"   : "s3",
					"endpoint" : "<<s3-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("requires the `name', `plugin', and `endpoint' keys to update an existing store", func() {
		res := PUT(API, "/v1/store/"+STORE_REDIS, "{}")
		Ω(res.Code).Should(Equal(400))
		Ω(res.Body.String()).Should(Equal(`{"missing":["name","plugin","endpoint"]}`))
	})

	It("can delete unused stores", func() {
		res := DELETE(API, "/v1/store/"+STORE_REDIS)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{"ok":"deleted"}`))
		Eventually(resyncChan).Should(Receive())

		res = GET(API, "/v1/stores")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "` + STORE_S3 + `",
					"name"     : "s3",
					"summary"  : "Amazon S3 Blobstore",
					"plugin"   : "s3",
					"endpoint" : "<<s3-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("refuses to delete a store that is in use", func() {
		res := DELETE(API, "/v1/store/"+STORE_S3)
		Ω(res.Code).Should(Equal(403))
		Ω(res.Body.String()).Should(Equal(""))
	})

	It("validates JSON payloads", func() {
		JSONValidated(API, "POST", "/v1/stores")
		JSONValidated(API, "PUT", "/v1/store/"+STORE_S3)
	})

	It("ignores other HTTP methods", func() {
		for _, method := range []string{"PUT", "DELETE", "PATCH", "OPTIONS", "TRACE"} {
			NotImplemented(API, method, "/v1/stores", nil)
		}

		for _, method := range []string{"GET", "HEAD", "POST", "PATCH", "OPTIONS", "TRACE"} {
			NotImplemented(API, method, "/v1/stores/sub/requests", nil)
			NotImplemented(API, method, "/v1/store/sub/requests", nil)
		}
	})

	It("ignores malformed UUIDs", func() {
		for _, id := range []string{"malformed-uuid-01234", "", "(abcdef-01234-56-789)"} {
			NotImplemented(API, "GET", fmt.Sprintf("/v1/store/%s", id), nil)
			NotImplemented(API, "PUT", fmt.Sprintf("/v1/store/%s", id), nil)
		}
	})
})
