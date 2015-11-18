package supervisor_test

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/starkandwayne/shield/supervisor"
	"net/http"
	"time"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"
)

var _ = Describe("/v1/archives API", func() {
	var API http.Handler
	var resyncChan chan int

	STORE_S3 := `05c3d005-f968-452f-bd59-bee8e79ab982`

	TARGET_REDIS := `66be7c43-6c57-4391-8ea9-e770d6ab5e9e`
	TARGET_PG := `fab00c82-aac3-4e5f-8a2f-c534f81cdee3`

	PG_ARCHIVE_1 := `a97f5532-3a9c-489b-a414-ba9d6740fa79`
	PG_ARCHIVE_2 := `b0eda11f-0414-4f6a-841f-c08609c542d0`

	REDIS_ARCHIVE_1 := `47dccf5e-e69d-4f94-9b29-ac8b185dda31`

	BeforeEach(func() {

		unixtime := func (t string) string {
			utc, err := time.LoadLocation("UTC")
			Ω(err).ShouldNot(HaveOccurred())
			tempt, err := time.ParseInLocation("2006-01-02 15:04:05", t, utc)
			Ω(err).ShouldNot(HaveOccurred())
			return fmt.Sprintf("%d",tempt.Unix())
		}

		data, err := Database(

			// TARGETS
			`INSERT INTO targets (uuid, name, summary, plugin, endpoint) VALUES
				("`+TARGET_REDIS+`",
				 "redis-shared",
				 "Shared Redis services for CF",
				 "redis",
				 "<<redis-configuration>>")`,
			`INSERT INTO targets (uuid, name, summary, plugin, endpoint) VALUES
				("`+TARGET_PG+`",
				 "pg1",
				 "Test Postgres Service",
				 "pg",
				 "<<pg-configuration>>")`,

			// STORES
			`INSERT INTO stores (uuid, name, summary, plugin, endpoint) VALUES
				("`+STORE_S3+`",
				 "s3",
				 "Amazon S3 Blobstore",
				 "s3",
				 "<<s3-configuration>>")`,

			// archive #1 for pg -> s3
			`INSERT INTO archives (uuid, target_uuid, store_uuid, store_key,
					taken_at, expires_at, notes) VALUES
				("`+PG_ARCHIVE_1+`",
				 "`+TARGET_PG+`",
				 "`+STORE_S3+`",
				 "pg-archive-1-key",
				 ` + unixtime("2015-04-21 03:00:01") + `,
				 ` + unixtime("2015-06-18 03:00:01") + `,
				 "test backup")`,

			// archive #2 for pg -> s3
			`INSERT INTO archives (uuid, target_uuid, store_uuid, store_key,
					taken_at, expires_at, notes) VALUES
				("`+PG_ARCHIVE_2+`",
				 "`+TARGET_PG+`",
				 "`+STORE_S3+`",
				 "pg-archive-2-key",
				 ` + unixtime("2015-04-28 03:00:01") + `,
				 ` + unixtime("2015-06-25 03:00:01") + `,
				 "")`,

			// archive #1 for redis -> s3
			`INSERT INTO archives (uuid, target_uuid, store_uuid, store_key,
					taken_at, expires_at, notes) VALUES
				("`+REDIS_ARCHIVE_1+`",
				 "`+TARGET_REDIS+`",
				 "`+STORE_S3+`",
				 "redis-archive-1-key",
				 ` + unixtime("2015-04-23 14:35:22") + `,
				 ` + unixtime("2015-04-25 14:35:22") + `,
				 "Good Redis Backup")`,
		)
		Ω(err).ShouldNot(HaveOccurred())
		resyncChan = make(chan int, 1)
		API = ArchiveAPI{
			Data:       data,
			ResyncChan: resyncChan,
		}
	})

	AfterEach(func() {
		close(resyncChan)
		resyncChan = nil
	})

	It("should retrieve all archives, sorted reverse chronological", func() {
		res := GET(API, "/v1/archives")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + PG_ARCHIVE_2 + `",
					"notes"           : "",
					"key"             : "pg-archive-2-key",
					"taken_at"        : "2015-04-28 03:00:01",
					"expires_at"      : "2015-06-25 03:00:01",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_uuid"     : "` + TARGET_PG + `",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>"
				},
				{
					"uuid"            : "` + REDIS_ARCHIVE_1 + `",
					"notes"           : "Good Redis Backup",
					"key"             : "redis-archive-1-key",
					"taken_at"        : "2015-04-23 14:35:22",
					"expires_at"      : "2015-04-25 14:35:22",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>"
				},
				{
					"uuid"            : "` + PG_ARCHIVE_1 + `",
					"notes"           : "test backup",
					"key"             : "pg-archive-1-key",
					"taken_at"        : "2015-04-21 03:00:01",
					"expires_at"      : "2015-06-18 03:00:01",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_uuid"     : "` + TARGET_PG + `",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve archives based on target UUID", func() {
		res := GET(API, "/v1/archives?target="+TARGET_PG)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + PG_ARCHIVE_2 + `",
					"notes"           : "",
					"key"             : "pg-archive-2-key",
					"taken_at"        : "2015-04-28 03:00:01",
					"expires_at"      : "2015-06-25 03:00:01",
					"target_uuid"     : "` + TARGET_PG + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"store_uuid"      : "` + STORE_S3 + `",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>"
				},
				{
					"uuid"            : "` + PG_ARCHIVE_1 + `",
					"notes"           : "test backup",
					"key"             : "pg-archive-1-key",
					"taken_at"        : "2015-04-21 03:00:01",
					"expires_at"      : "2015-06-18 03:00:01",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_uuid"     : "` + TARGET_PG + `",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve archives taken before a given timestamp", func() {
		res := GET(API, "/v1/archives?before=20150424")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + REDIS_ARCHIVE_1 + `",
					"notes"           : "Good Redis Backup",
					"key"             : "redis-archive-1-key",
					"taken_at"        : "2015-04-23 14:35:22",
					"expires_at"      : "2015-04-25 14:35:22",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>"
				},
				{
					"uuid"            : "` + PG_ARCHIVE_1 + `",
					"notes"           : "test backup",
					"key"             : "pg-archive-1-key",
					"taken_at"        : "2015-04-21 03:00:01",
					"expires_at"      : "2015-06-18 03:00:01",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_uuid"     : "` + TARGET_PG + `",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>"
				}
			]`))
	})

	It("should retrieve archives taken after a given timestamp", func() {
		res := GET(API, "/v1/archives?after=20150424")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + PG_ARCHIVE_2 + `",
					"notes"           : "",
					"key"             : "pg-archive-2-key",
					"taken_at"        : "2015-04-28 03:00:01",
					"expires_at"      : "2015-06-25 03:00:01",
					"target_uuid"     : "` + TARGET_PG + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"store_uuid"      : "` + STORE_S3 + `",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>"
				}
			]`))
	})

	It("should retrieve archives taken between two timestamps", func() {
		res := GET(API, "/v1/archives?after=20150422&before=20150424")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + REDIS_ARCHIVE_1 + `",
					"notes"           : "Good Redis Backup",
					"key"             : "redis-archive-1-key",
					"taken_at"        : "2015-04-23 14:35:22",
					"expires_at"      : "2015-04-25 14:35:22",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>"
				}
			]`))
	})

	It("can retrieve a single archive by UUID", func() {
		res := GET(API, "/v1/archive/"+REDIS_ARCHIVE_1)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{
				"uuid"            : "` + REDIS_ARCHIVE_1 + `",
				"notes"           : "Good Redis Backup",
				"key"             : "redis-archive-1-key",
				"taken_at"        : "2015-04-23 14:35:22",
				"expires_at"      : "2015-04-25 14:35:22",
				"store_uuid"      : "` + STORE_S3 + `",
				"store_plugin"    : "s3",
				"store_endpoint"  : "<<s3-configuration>>",
				"target_uuid"     : "` + TARGET_REDIS + `",
				"target_plugin"   : "redis",
				"target_endpoint" : "<<redis-configuration>>"
			}`))
	})

	It("returns a 404 for unknown UUIDs", func() {
		res := GET(API, "/v1/archive/"+TARGET_REDIS) // it's a target...
		Ω(res.Code).Should(Equal(404))
	})

	It("cannot create new archives", func() {
		res := POST(API, "/v1/archives", WithJSON(`{}`))
		Ω(res.Code).Should(Equal(415))
	})

	It("can annotate archives", func() {
		res := GET(API, "/v1/archives?target="+TARGET_REDIS)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + REDIS_ARCHIVE_1 + `",
					"notes"           : "Good Redis Backup",
					"key"             : "redis-archive-1-key",
					"taken_at"        : "2015-04-23 14:35:22",
					"expires_at"      : "2015-04-25 14:35:22",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>"
				}
			]`))

		res = PUT(API, "/v1/archive/"+REDIS_ARCHIVE_1, WithJSON(`{
			"notes" : "These are my updated notes on this archive"
		}`))
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{"ok":"updated"}`))
		Eventually(resyncChan).Should(Receive())

		res = GET(API, "/v1/archives?target="+TARGET_REDIS)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + REDIS_ARCHIVE_1 + `",
					"notes"           : "These are my updated notes on this archive",
					"key"             : "redis-archive-1-key",
					"taken_at"        : "2015-04-23 14:35:22",
					"expires_at"      : "2015-04-25 14:35:22",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>"
				}
			]`))
	})

	It("requires the `notes' key when updating archives", func() {
		res := PUT(API, "/v1/archive/"+REDIS_ARCHIVE_1, WithJSON("{}"))
		Ω(res.Code).Should(Equal(400))
	})

	It("can delete archives", func() {
		res := GET(API, "/v1/archives?target="+TARGET_REDIS)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + REDIS_ARCHIVE_1 + `",
					"notes"           : "Good Redis Backup",
					"key"             : "redis-archive-1-key",
					"taken_at"        : "2015-04-23 14:35:22",
					"expires_at"      : "2015-04-25 14:35:22",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>"
				}
			]`))

		res = DELETE(API, "/v1/archive/"+REDIS_ARCHIVE_1)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{"ok":"deleted"}`))
		Eventually(resyncChan).Should(Receive())

		res = GET(API, "/v1/archives?target="+TARGET_REDIS)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[]`))
	})

	It("ignores other HTTP methods", func() {
		for _, method := range []string{"PUT", "DELETE", "PATCH", "OPTIONS", "TRACE"} {
			NotImplemented(API, method, "/v1/archives", nil)
		}

		for _, method := range []string{"GET", "HEAD", "POST", "PATCH", "OPTIONS", "TRACE"} {
			NotImplemented(API, method, "/v1/archives/sub/requests", nil)
			NotImplemented(API, method, "/v1/archive/sub/requests", nil)
		}
	})

	It("ignores malformed UUIDs", func() {
		for _, id := range []string{"malformed-uuid-01234", "(abcdef-01234-56-789)"} {
			NotImplemented(API, "PUT", fmt.Sprintf("/v1/archive/%s", id), nil)
		}
	})
})
