package api_test

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/starkandwayne/shield/api"
	"net/http"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"
)

var _ = Describe("/v1/targets API", func() {
	var API http.Handler
	var channel chan int

	BeforeEach(func() {
		data, err := Database(
			`INSERT INTO targets (uuid, name, summary, plugin, endpoint) VALUES
				("66be7c43-6c57-4391-8ea9-e770d6ab5e9e",
				 "redis-shared",
				 "Shared Redis services for CF",
				 "redis",
				 "<<redis-configuration>>")`,

			`INSERT INTO targets (uuid, name, summary, plugin, endpoint) VALUES
				("05c3d005-f968-452f-bd59-bee8e79ab982",
				 "s3",
				 "Amazon S3 Blobstore",
				 "s3",
				 "<<s3-configuration>>")`,

			`INSERT INTO jobs (uuid, target_uuid) VALUES
				("abc-def",
				 "05c3d005-f968-452f-bd59-bee8e79ab982")`,
		)
		Ω(err).ShouldNot(HaveOccurred())
		channel = make(chan int, 1)
		API = &TargetAPI{Data: data, SuperChan: channel}
	})

	AfterEach(func() {
		close(channel)
		channel = nil
	})

	It("should retrieve all targets", func() {
		res := GET(API, "/v1/targets")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "66be7c43-6c57-4391-8ea9-e770d6ab5e9e",
					"name"     : "redis-shared",
					"summary"  : "Shared Redis services for CF",
					"plugin"   : "redis",
					"endpoint" : "<<redis-configuration>>"
				},
				{
					"uuid"     : "05c3d005-f968-452f-bd59-bee8e79ab982",
					"name"     : "s3",
					"summary"  : "Amazon S3 Blobstore",
					"plugin"   : "s3",
					"endpoint" : "<<s3-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve only unused targets ?unused=t", func() {
		res := GET(API, "/v1/targets?unused=t")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "66be7c43-6c57-4391-8ea9-e770d6ab5e9e",
					"name"     : "redis-shared",
					"summary"  : "Shared Redis services for CF",
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
					"uuid"     : "05c3d005-f968-452f-bd59-bee8e79ab982",
					"name"     : "s3",
					"summary"  : "Amazon S3 Blobstore",
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
					"uuid"     : "66be7c43-6c57-4391-8ea9-e770d6ab5e9e",
					"name"     : "redis-shared",
					"summary"  : "Shared Redis services for CF",
					"plugin"   : "redis",
					"endpoint" : "<<redis-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))

		res = GET(API, "/v1/targets?plugin=s3")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "05c3d005-f968-452f-bd59-bee8e79ab982",
					"name"     : "s3",
					"summary"  : "Amazon S3 Blobstore",
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
					"uuid"     : "05c3d005-f968-452f-bd59-bee8e79ab982",
					"name"     : "s3",
					"summary"  : "Amazon S3 Blobstore",
					"plugin"   : "s3",
					"endpoint" : "<<s3-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))

		res = GET(API, "/v1/targets?plugin=s3&unused=t")
		Ω(res.Body.String()).Should(MatchJSON(`[]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("can create new targets", func() {
		res := POST(API, "/v1/targets", WithJSON(`{
			"name"     : "New Target",
			"summary"  : "A new one",
			"plugin"   : "s3",
			"endpoint" : "[ENDPOINT]"
		}`))
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchRegexp(`{"ok":"created","uuid":"[a-z0-9-]+"}`))
		Eventually(channel).Should(Receive())
	})

	It("requires the `name' and `when' keys in POST'ed data", func() {
		res := POST(API, "/v1/targets", "{}")
		Ω(res.Code).Should(Equal(400))
	})

	It("can update existing target", func() {
		res := PUT(API, "/v1/target/66be7c43-6c57-4391-8ea9-e770d6ab5e9e", WithJSON(`{
			"name"     : "Renamed",
			"summary"  : "UPDATED!",
			"plugin"   : "redis",
			"endpoint" : "{NEW-ENDPOINT}"
		}`))
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{"ok":"updated"}`))
		Eventually(channel).Should(Receive())

		res = GET(API, "/v1/targets")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "66be7c43-6c57-4391-8ea9-e770d6ab5e9e",
					"name"     : "Renamed",
					"summary"  : "UPDATED!",
					"plugin"   : "redis",
					"endpoint" : "{NEW-ENDPOINT}"
				},
				{
					"uuid"     : "05c3d005-f968-452f-bd59-bee8e79ab982",
					"name"     : "s3",
					"summary"  : "Amazon S3 Blobstore",
					"plugin"   : "s3",
					"endpoint" : "<<s3-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("can delete unused targets", func() {
		res := DELETE(API, "/v1/target/66be7c43-6c57-4391-8ea9-e770d6ab5e9e")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{"ok":"deleted"}`))
		Eventually(channel).Should(Receive())

		res = GET(API, "/v1/targets")
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"     : "05c3d005-f968-452f-bd59-bee8e79ab982",
					"name"     : "s3",
					"summary"  : "Amazon S3 Blobstore",
					"plugin"   : "s3",
					"endpoint" : "<<s3-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("refuses to delete a target that is in use", func() {
		res := DELETE(API, "/v1/target/05c3d005-f968-452f-bd59-bee8e79ab982")
		Ω(res.Code).Should(Equal(403))
		Ω(res.Body.String()).Should(Equal(""))
	})

	It("ignores other HTTP methods", func() {
		for _, method := range []string{"PUT", "DELETE", "PATCH", "OPTIONS", "TRACE"} {
			NotImplemented(API, method, "/v1/targets", nil)
		}

		for _, method := range []string{"GET", "HEAD", "POST", "PATCH", "OPTIONS", "TRACE"} {
			NotImplemented(API, method, "/v1/targets/sub/requests", nil)
			NotImplemented(API, method, "/v1/target/sub/requests", nil)
			NotImplemented(API, method, "/v1/target/5981f34c-ef58-4e3b-a91e-428480c68100", nil)
		}
	})

	It("ignores malformed UUIDs", func() {
		for _, id := range []string{"malformed-uuid-01234", "", "(abcdef-01234-56-789)"} {
			NotImplemented(API, "GET", fmt.Sprintf("/v1/target/%s", id), nil)
			NotImplemented(API, "PUT", fmt.Sprintf("/v1/target/%s", id), nil)
		}
	})
})
