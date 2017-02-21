package supervisor_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/starkandwayne/shield/db"
	. "github.com/starkandwayne/shield/supervisor"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"
)

var _ = Describe("/v1/archives API", func() {
	var API http.Handler
	var resyncChan chan int
	var adhocChan chan *db.Task

	STORE_S3 := `05c3d005-f968-452f-bd59-bee8e79ab982`

	TARGET_REDIS := `66be7c43-6c57-4391-8ea9-e770d6ab5e9e`
	TARGET_PG := `fab00c82-aac3-4e5f-8a2f-c534f81cdee3`
	TARGET_INVALID := `825abfc4-73ff-40d0-b878-58e0dcda9084`

	PG_ARCHIVE_1 := `a97f5532-3a9c-489b-a414-ba9d6740fa79`
	PG_ARCHIVE_2 := `b0eda11f-0414-4f6a-841f-c08609c542d0`

	REDIS_ARCHIVE_1 := `47dccf5e-e69d-4f94-9b29-ac8b185dda31`

	INVALID_ARCHIVE_1 := `2eaa8cad-57d0-4bdd-bb53-25f9acc2ef29`

	BeforeEach(func() {

		unixtime := func(t string) string {
			utc, err := time.LoadLocation("UTC")
			Ω(err).ShouldNot(HaveOccurred())
			tempt, err := time.ParseInLocation("2006-01-02 15:04:05", t, utc)
			Ω(err).ShouldNot(HaveOccurred())
			return fmt.Sprintf("%d", tempt.Unix())
		}

		data, err := Database(

			// TARGETS
			`INSERT INTO targets (uuid, name, summary, plugin, endpoint, agent) VALUES
				("`+TARGET_REDIS+`",
				 "redis-shared",
				 "Shared Redis services for CF",
				 "redis",
				 "<<redis-configuration>>",
				"127.0.0.1:5444")`,
			`INSERT INTO targets (uuid, name, summary, plugin, endpoint, agent) VALUES
				("`+TARGET_PG+`",
				 "pg1",
				 "Test Postgres Service",
				 "pg",
				 "<<pg-configuration>>",
				"127.0.0.1:5444")`,
			`INSERT INTO targets (uuid, name, summary, plugin, endpoint, agent) VALUES
				("`+TARGET_INVALID+`",
				 "pg1",
				 "Test Invalid Service",
				 "invalid",
				 "<<invalid-configuration>>",
				"127.0.0.1:5444")`,

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
				 `+unixtime("2015-04-21 03:00:01")+`,
				 `+unixtime("2015-06-18 03:00:01")+`,
				 "test backup")`,

			// archive #2 for pg -> s3
			`INSERT INTO archives (uuid, target_uuid, store_uuid, store_key,
					taken_at, expires_at, notes) VALUES
				("`+PG_ARCHIVE_2+`",
				 "`+TARGET_PG+`",
				 "`+STORE_S3+`",
				 "pg-archive-2-key",
				 `+unixtime("2015-04-28 03:00:01")+`,
				 `+unixtime("2015-06-25 03:00:01")+`,
				 "")`,

			// archive #1 for redis -> s3
			`INSERT INTO archives (uuid, target_uuid, store_uuid, store_key,
					taken_at, expires_at, notes) VALUES
				("`+REDIS_ARCHIVE_1+`",
				 "`+TARGET_REDIS+`",
				 "`+STORE_S3+`",
				 "redis-archive-1-key",
				 `+unixtime("2015-04-23 14:35:22")+`,
				 `+unixtime("2015-04-25 14:35:22")+`,
				 "Good Redis Backup")`,

			// archive #1 for invalid -> s3
			`INSERT INTO archives (uuid, target_uuid, store_uuid, store_key,
					taken_at, expires_at, notes, status) VALUES
				("`+INVALID_ARCHIVE_1+`",
				 "`+TARGET_INVALID+`",
				 "`+STORE_S3+`",
				 "invalid-archive-1-key",
				 `+unixtime("2015-04-23 14:35:22")+`,
				 `+unixtime("2015-04-25 14:35:22")+`,
				 "Invalid Backup",
				 "invalid")`,
		)
		Ω(err).ShouldNot(HaveOccurred())
		resyncChan = make(chan int, 1)
		adhocChan = make(chan *db.Task, 1)
		API = ArchiveAPI{
			Data:       data,
			ResyncChan: resyncChan,
			Tasks:      adhocChan,
		}
	})

	AfterEach(func() {
		close(resyncChan)
		resyncChan = nil
		close(adhocChan)
		adhocChan = nil
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
			  "status"           : "valid",
			  "purge_reason"    : "",
			  "store_uuid"      : "` + STORE_S3 + `",
			  "store_name"      : "s3",
			  "store_plugin"    : "s3",
			  "store_endpoint"  : "<<s3-configuration>>",
			  "target_uuid"     : "` + TARGET_PG + `",
			  "target_name"     : "pg1",
			  "target_plugin"   : "pg",
			  "target_endpoint" : "<<pg-configuration>>"
			},
			{
			  "uuid"            : "2eaa8cad-57d0-4bdd-bb53-25f9acc2ef29",
			  "key"             : "invalid-archive-1-key",
			  "taken_at"        : "2015-04-23 14:35:22",
			  "expires_at"      : "2015-04-25 14:35:22",
			  "notes"           : "Invalid Backup",
			  "status"          : "invalid",
			  "purge_reason"    : "",
			  "store_uuid"      : "05c3d005-f968-452f-bd59-bee8e79ab982",
			  "store_name"      : "s3",
			  "store_plugin"    : "s3",
			  "store_endpoint"  : "\u003c\u003cs3-configuration\u003e\u003e",
			  "target_uuid"     : "825abfc4-73ff-40d0-b878-58e0dcda9084",
			  "target_name"     : "pg1",
			  "target_plugin"   : "invalid",
			  "target_endpoint" : "\u003c\u003cinvalid-configuration\u003e\u003e"
			},
			{
			  "uuid"            : "` + REDIS_ARCHIVE_1 + `",
			  "notes"           : "Good Redis Backup",
			  "key"             : "redis-archive-1-key",
			  "taken_at"        : "2015-04-23 14:35:22",
			  "expires_at"      : "2015-04-25 14:35:22",
			  "status"           : "valid",
			  "purge_reason"    : "",
			  "store_uuid"      : "` + STORE_S3 + `",
			  "store_name"      : "s3",
			  "store_plugin"    : "s3",
			  "store_endpoint"  : "<<s3-configuration>>",
			  "target_uuid"     : "` + TARGET_REDIS + `",
			  "target_name"     : "redis-shared",
			  "target_plugin"   : "redis",
			  "target_endpoint" : "<<redis-configuration>>"
			},
			{
			  "uuid"            : "` + PG_ARCHIVE_1 + `",
			  "notes"           : "test backup",
			  "key"             : "pg-archive-1-key",
			  "taken_at"        : "2015-04-21 03:00:01",
			  "expires_at"      : "2015-06-18 03:00:01",
			  "status"           : "valid",
			  "purge_reason"    : "",
			  "store_uuid"      : "` + STORE_S3 + `",
			  "store_name"      : "s3",
			  "store_plugin"    : "s3",
			  "store_endpoint"  : "<<s3-configuration>>",
			  "target_uuid"     : "` + TARGET_PG + `",
			  "target_name"     : "pg1",
			  "target_plugin"   : "pg",
			  "target_endpoint" : "<<pg-configuration>>"
			}
			]`))
		Ω(res.Code).Should(Equal(200))
	})
	It("should retrieve qty of archives based on valid limit", func() {
		res := GET(API, "/v1/archives?limit=1")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`
			[{
				"uuid"            : "b0eda11f-0414-4f6a-841f-c08609c542d0",
				"key"             : "pg-archive-2-key",
				"taken_at"        : "2015-04-28 03:00:01",
				"expires_at"      : "2015-06-25 03:00:01",
				"notes"           : "",
				"status"          : "valid",
				"purge_reason"    : "",
				"target_uuid"     : "fab00c82-aac3-4e5f-8a2f-c534f81cdee3",
				"target_name"     : "pg1",
				"target_plugin"   : "pg",
				"target_endpoint" : "\u003c\u003cpg-configuration\u003e\u003e",
				"store_uuid"      : "05c3d005-f968-452f-bd59-bee8e79ab982",
				"store_name"      : "s3",
				"store_plugin"    : "s3",
				"store_endpoint"  : "\u003c\u003cs3-configuration\u003e\u003e"
		}]`))
	})
	It("should fail when provided an invalid limit", func() {
		res := GET(API, "/v1/archives?limit=n")
		Ω(res.Code).Should(Equal(400))
		Ω(res.Body.String()).Should(MatchJSON(`{"error":"invalid limit supplied"}`))

		res = GET(API, "/v1/archives?limit=-1")
		Ω(res.Code).Should(Equal(400))
		Ω(res.Body.String()).Should(MatchJSON(`{"error":"invalid limit supplied"}`))
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
					"status"           : "valid",
					"purge_reason"    : "",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_name"      : "s3",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_uuid"     : "` + TARGET_PG + `",
					"target_name"     : "pg1",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>"
				},
				{
					"uuid"            : "` + PG_ARCHIVE_1 + `",
					"notes"           : "test backup",
					"key"             : "pg-archive-1-key",
					"taken_at"        : "2015-04-21 03:00:01",
					"expires_at"      : "2015-06-18 03:00:01",
					"status"           : "valid",
					"purge_reason"    : "",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_name"      : "s3",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_uuid"     : "` + TARGET_PG + `",
					"target_name"     : "pg1",
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
					"uuid"            : "2eaa8cad-57d0-4bdd-bb53-25f9acc2ef29",
					"key"             : "invalid-archive-1-key",
					"taken_at"        : "2015-04-23 14:35:22",
					"expires_at"      : "2015-04-25 14:35:22",
					"notes"           : "Invalid Backup",
					"status"          : "invalid",
					"purge_reason"    : "",
					"target_uuid"     : "825abfc4-73ff-40d0-b878-58e0dcda9084",
					"target_name"     : "pg1",
					"target_plugin"   : "invalid",
					"target_endpoint" : "\u003c\u003cinvalid-configuration\u003e\u003e",
					"store_uuid"      : "05c3d005-f968-452f-bd59-bee8e79ab982",
					"store_name"      : "s3",
					"store_plugin"    : "s3",
					"store_endpoint"  : "\u003c\u003cs3-configuration\u003e\u003e"
				},
				{
					"uuid"            : "` + REDIS_ARCHIVE_1 + `",
					"notes"           : "Good Redis Backup",
					"key"             : "redis-archive-1-key",
					"taken_at"        : "2015-04-23 14:35:22",
					"expires_at"      : "2015-04-25 14:35:22",
					"status"           : "valid",
					"purge_reason"    : "",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_name"      : "s3",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_name"     : "redis-shared",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>"
				},
				{
					"uuid"            : "` + PG_ARCHIVE_1 + `",
					"notes"           : "test backup",
					"key"             : "pg-archive-1-key",
					"taken_at"        : "2015-04-21 03:00:01",
					"expires_at"      : "2015-06-18 03:00:01",
					"status"           : "valid",
					"purge_reason"    : "",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_name"      : "s3",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_uuid"     : "` + TARGET_PG + `",
					"target_name"     : "pg1",
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
					"status"          : "valid",
					"purge_reason"    : "",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_name"      : "s3",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_uuid"     : "` + TARGET_PG + `",
					"target_name"     : "pg1",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>"

				}
			]`))
	})

	It("should retrieve archives taken between two timestamps with a specific status", func() {
		res := GET(API, "/v1/archives?after=20150422&before=20150424&status=valid")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + REDIS_ARCHIVE_1 + `",
					"notes"           : "Good Redis Backup",
					"key"             : "redis-archive-1-key",
					"taken_at"        : "2015-04-23 14:35:22",
					"expires_at"      : "2015-04-25 14:35:22",
					"status"           : "valid",
					"purge_reason"    : "",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_name"      : "s3",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_name"     : "redis-shared",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>"
				}
			]`))
	})
	It("Should retrieve archives with a specific status", func() {
		res := GET(API, "/v1/archives?status=invalid")
		Expect(res.Code).Should(Equal(200))
		Expect(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + INVALID_ARCHIVE_1 + `",
					"notes"           : "Invalid Backup",
					"key"             : "invalid-archive-1-key",
					"taken_at"        : "2015-04-23 14:35:22",
					"expires_at"      : "2015-04-25 14:35:22",
					"status"           : "invalid",
					"purge_reason"    : "",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_name"      : "s3",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_uuid"     : "` + TARGET_INVALID + `",
					"target_name"     : "pg1",
					"target_plugin"   : "invalid",
					"target_endpoint" : "<<invalid-configuration>>",
					"status"           : "invalid"
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
				"status"           : "valid",
				"purge_reason"    : "",
				"expires_at"      : "2015-04-25 14:35:22",
				"store_uuid"      : "` + STORE_S3 + `",
				"store_name"      : "s3",
				"store_plugin"    : "s3",
				"store_endpoint"  : "<<s3-configuration>>",
				"target_uuid"     : "` + TARGET_REDIS + `",
				"target_name"     : "redis-shared",
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
		Ω(res.Code).Should(Equal(501))
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
					"status"          : "valid",
					"purge_reason"    : "",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_name"      : "s3",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_name"     : "redis-shared",
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
					"status"           : "valid",
					"purge_reason"    : "",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_name"      : "s3",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_name"     : "redis-shared",
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
					"status"          : "valid",
					"purge_reason"    : "",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_name"      : "s3",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_name"     : "redis-shared",
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

	Context("When queuing up restore jobs", func() {
		var errChan chan error
		var taskChannelFodder db.TaskInfo
		JustBeforeEach(func() {
			errChan = make(chan error, 1)
			go func() {
				var task *db.Task
				select {
				case task = <-adhocChan:
					errChan <- nil
					task.TaskUUIDChan <- &taskChannelFodder
				case <-time.After(2 * time.Second):
					errChan <- errors.New("I timed out!")
				}
			}()
		})
		Context("when the task is created with no error", func() {
			testUUID := "magical-mystery-uuid"
			BeforeEach(func() {
				taskChannelFodder = db.TaskInfo{
					Err:  false,
					Info: testUUID,
				}
			})
			It("it returns the task uuid", func() {
				res := POST(API, "/v1/archive/"+REDIS_ARCHIVE_1+"/restore", "")
				Ω(res.Code).Should(Equal(200))
				expected, err := json.Marshal(map[string]string{
					"ok":        "scheduled",
					"task_uuid": testUUID,
				})
				Ω(err).ShouldNot(HaveOccurred())
				Ω(res.Body.String()).Should(MatchJSON(expected))
				Ω(<-errChan).Should(BeNil())
			})
		})
		Context("when there is an error creating the task", func() {
			errorMessage := "All your task are belong to us"
			BeforeEach(func() {
				taskChannelFodder = db.TaskInfo{
					Err:  true,
					Info: errorMessage,
				}
			})

			It("returns the error through the API", func() {
				res := POST(API, "/v1/archive/"+REDIS_ARCHIVE_1+"/restore", "")
				Ω(res.Code).Should(Equal(500))
				expected, err := json.Marshal(map[string]string{
					"error": errorMessage,
				})
				Ω(err).ShouldNot(HaveOccurred())
				Ω(res.Body.String()).Should(MatchJSON(expected))
				Ω(<-errChan).Should(BeNil())
			})
		})

	})
})
