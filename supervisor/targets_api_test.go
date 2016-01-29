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

var _ = Describe("/v1/targets API", func() {
	var API http.Handler
	var channel chan int

	TARGET_REDIS := `66be7c43-6c57-4391-8ea9-e770d6ab5e9e`
	TARGET_S3 := `05c3d005-f968-452f-bd59-bee8e79ab982`

	NIL := `00000000-0000-0000-0000-000000000000`

	BeforeEach(func() {
		data, err := Database(
			`INSERT INTO targets (uuid, name, summary, agent, plugin, endpoint) VALUES
				("`+TARGET_REDIS+`",
				 "redis-shared",
				 "Shared Redis services for CF",
				 "127.0.0.1:5544",
				 "redis",
				 "<<redis-configuration>>")`,

			`INSERT INTO targets (uuid, name, summary, agent, plugin, endpoint) VALUES
				("`+TARGET_S3+`",
				 "s3",
				 "Amazon S3 Blobstore",
				 "127.0.0.1:5544",
				 "s3",
				 "<<s3-configuration>>")`,

			`INSERT INTO jobs (uuid, store_uuid, target_uuid, schedule_uuid, retention_uuid)
				VALUES ("abc-def", "`+NIL+`", "`+TARGET_S3+`", "`+NIL+`", "`+NIL+`")`,
		)
		Ω(err).ShouldNot(HaveOccurred())
		channel = make(chan int, 1)
		API = &TargetAPI{
			Data:       data,
			ResyncChan: channel,
		}
	})

	AfterEach(func() {
		close(channel)
		channel = nil
	})

	It("should retrieve all targets", func() {
		res := GET(API, "/v1/targets")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "` + TARGET_REDIS + `",
					"name"     : "redis-shared",
					"summary"  : "Shared Redis services for CF",
					"agent"    : "127.0.0.1:5544",
					"plugin"   : "redis",
					"endpoint" : "<<redis-configuration>>"
				},
				{
					"uuid"     : "` + TARGET_S3 + `",
					"name"     : "s3",
					"summary"  : "Amazon S3 Blobstore",
					"agent"    : "127.0.0.1:5544",
					"plugin"   : "s3",
					"endpoint" : "<<s3-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve all targets named 'redis'", func() {
		res := GET(API, "/v1/targets?name=redis")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "` + TARGET_REDIS + `",
					"name"     : "redis-shared",
					"summary"  : "Shared Redis services for CF",
					"agent"    : "127.0.0.1:5544",
					"plugin"   : "redis",
					"endpoint" : "<<redis-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve only unused targets ?unused=t", func() {
		res := GET(API, "/v1/targets?unused=t")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "` + TARGET_REDIS + `",
					"name"     : "redis-shared",
					"summary"  : "Shared Redis services for CF",
					"agent"    : "127.0.0.1:5544",
					"plugin"   : "redis",
					"endpoint" : "<<redis-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve only unused targets named 's' for ?unused=t and ?name=s", func() {
		res := GET(API, "/v1/targets?unused=t&name=s")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "` + TARGET_REDIS + `",
					"name"     : "redis-shared",
					"summary"  : "Shared Redis services for CF",
					"agent"    : "127.0.0.1:5544",
					"plugin"   : "redis",
					"endpoint" : "<<redis-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve only used targets for ?unused=f", func() {
		res := GET(API, "/v1/targets?unused=f")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "` + TARGET_S3 + `",
					"name"     : "s3",
					"summary"  : "Amazon S3 Blobstore",
					"agent"    : "127.0.0.1:5544",
					"plugin"   : "s3",
					"endpoint" : "<<s3-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should filter targets by plugin name", func() {
		res := GET(API, "/v1/targets?plugin=redis")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "` + TARGET_REDIS + `",
					"name"     : "redis-shared",
					"summary"  : "Shared Redis services for CF",
					"agent"    : "127.0.0.1:5544",
					"plugin"   : "redis",
					"endpoint" : "<<redis-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))

		res = GET(API, "/v1/targets?plugin=s3")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "` + TARGET_S3 + `",
					"name"     : "s3",
					"summary"  : "Amazon S3 Blobstore",
					"agent"    : "127.0.0.1:5544",
					"plugin"   : "s3",
					"endpoint" : "<<s3-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))

		res = GET(API, "/v1/targets?plugin=enoent")
		Ω(res.Body.String()).Should(MatchJSON(`[]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should filter by combinations of `plugin' and `unused' parameters", func() {
		res := GET(API, "/v1/targets?plugin=s3&unused=f")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "` + TARGET_S3 + `",
					"name"     : "s3",
					"summary"  : "Amazon S3 Blobstore",
					"agent"    : "127.0.0.1:5544",
					"plugin"   : "s3",
					"endpoint" : "<<s3-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))

		res = GET(API, "/v1/targets?plugin=s3&unused=t")
		Ω(res.Body.String()).Should(MatchJSON(`[]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("can retrieve a single target by UUID", func() {
		res := GET(API, "/v1/target/"+TARGET_S3)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{
					"uuid"     : "` + TARGET_S3 + `",
					"name"     : "s3",
					"summary"  : "Amazon S3 Blobstore",
					"agent"    : "127.0.0.1:5544",
					"plugin"   : "s3",
					"endpoint" : "<<s3-configuration>>"
			}`))
	})

	It("returns a 404 for unknown UUIDs", func() {
		res := GET(API, "/v1/target/14a0865d-81e7-4cfe-b733-6170a368eecd")
		Ω(res.Code).Should(Equal(404))
	})

	It("can create new targets", func() {
		res := POST(API, "/v1/targets", WithJSON(`{
			"name"     : "New Target",
			"summary"  : "A new one",
			"plugin"   : "s3",
			"endpoint" : "[ENDPOINT]",
			"agent"    : "127.0.0.1:5544"
		}`))
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchRegexp(`{"ok":"created","uuid":"[a-z0-9-]+"}`))
		Eventually(channel).Should(Receive())
	})

	It("requires the `name', `plugin', `endpoint', `agent' keys to create a new target", func() {
		res := POST(API, "/v1/targets", "{}")
		Ω(res.Code).Should(Equal(400))
		Ω(res.Body.String()).Should(Equal(`{"missing":["name","plugin","endpoint","agent"]}`))
	})

	It("can update existing target", func() {
		res := PUT(API, "/v1/target/"+TARGET_REDIS, WithJSON(`{
			"name"     : "Renamed",
			"summary"  : "UPDATED!",
			"plugin"   : "redis",
			"endpoint" : "{NEW-ENDPOINT}",
			"agent"    : "127.0.0.1:1660"
		}`))
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{"ok":"updated"}`))
		Eventually(channel).Should(Receive())

		res = GET(API, "/v1/targets")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "` + TARGET_REDIS + `",
					"name"     : "Renamed",
					"summary"  : "UPDATED!",
					"agent"    : "127.0.0.1:1660",
					"plugin"   : "redis",
					"endpoint" : "{NEW-ENDPOINT}"
				},
				{
					"uuid"     : "` + TARGET_S3 + `",
					"name"     : "s3",
					"summary"  : "Amazon S3 Blobstore",
					"agent"    : "127.0.0.1:5544",
					"plugin"   : "s3",
					"endpoint" : "<<s3-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("requires the `name', `plugin', `endpoint', `agent' key to update an existing target", func() {
		res := PUT(API, "/v1/target/"+TARGET_REDIS, "{}")
		Ω(res.Code).Should(Equal(400))
		Ω(res.Body.String()).Should(Equal(`{"missing":["name","plugin","endpoint","agent"]}`))
	})

	It("can delete unused targets", func() {
		res := DELETE(API, "/v1/target/"+TARGET_REDIS)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{"ok":"deleted"}`))
		Eventually(channel).Should(Receive())

		res = GET(API, "/v1/targets")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "` + TARGET_S3 + `",
					"name"     : "s3",
					"summary"  : "Amazon S3 Blobstore",
					"agent"    : "127.0.0.1:5544",
					"plugin"   : "s3",
					"endpoint" : "<<s3-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("refuses to delete a target that is in use", func() {
		res := DELETE(API, "/v1/target/"+TARGET_S3)
		Ω(res.Code).Should(Equal(403))
		Ω(res.Body.String()).Should(Equal(""))
	})

	It("validates JSON payloads", func() {
		JSONValidated(API, "POST", "/v1/targets")
		JSONValidated(API, "PUT", "/v1/target/"+TARGET_S3)
	})

	It("ignores other HTTP methods", func() {
		for _, method := range []string{"PUT", "DELETE", "PATCH", "OPTIONS", "TRACE"} {
			NotImplemented(API, method, "/v1/targets", nil)
		}

		for _, method := range []string{"GET", "HEAD", "POST", "PATCH", "OPTIONS", "TRACE"} {
			NotImplemented(API, method, "/v1/targets/sub/requests", nil)
			NotImplemented(API, method, "/v1/target/sub/requests", nil)
		}
	})

	It("ignores malformed UUIDs", func() {
		for _, id := range []string{"malformed-uuid-01234", "", "(abcdef-01234-56-789)"} {
			NotImplemented(API, "GET", fmt.Sprintf("/v1/target/%s", id), nil)
			NotImplemented(API, "PUT", fmt.Sprintf("/v1/target/%s", id), nil)
		}
	})
})
