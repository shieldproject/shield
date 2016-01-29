package supervisor_test

import (
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	. "github.com/starkandwayne/shield/supervisor"
)

var _ = Describe("/v1/jobs API", func() {
	var API http.Handler
	var resyncChan chan int
	var adhocChan chan AdhocTask

	STORE_S3 := `05c3d005-f968-452f-bd59-bee8e79ab982`

	TARGET_REDIS := `66be7c43-6c57-4391-8ea9-e770d6ab5e9e`
	TARGET_PG := `fab00c82-aac3-4e5f-8a2f-c534f81cdee3`

	SCHED_DAILY := `590eddbd-426f-408c-981b-9cf1faf2669e`
	SCHED_WEEKLY := `fce33a96-d352-480f-b04a-db7f2c14e98f`

	RETAIN_SHORT := `848ff67e-f857-47bd-9692-ae5f2be85674`
	RETAIN_LONG := `c5fca8e0-7d40-4cff-8dec-5f0df36ecee9`

	PG_S3_DAILY := `a7bb2fc2-5576-4887-b7f0-92a9cdb9ea8f`
	PG_S3_WEEKLY := `ea5eec7a-20b8-40ef-9868-cb4c54133018`
	REDIS_S3_DAILY := `9f42b026-8698-4150-b508-561a73031a47`
	REDIS_S3_WEEKLY := `4259dd16-0fad-478b-815c-9a9a7d209f56`

	BeforeEach(func() {
		data, err := Database(

			// TARGETS
			`INSERT INTO targets (uuid, name, summary, agent, plugin, endpoint) VALUES
				("`+TARGET_REDIS+`",
				 "redis-shared",
				 "Shared Redis services for CF",
				 "10.11.22.33:4455",
				 "redis",
				 "<<redis-configuration>>")`,
			`INSERT INTO targets (uuid, name, summary, agent, plugin, endpoint) VALUES
				("`+TARGET_PG+`",
				 "pg1",
				 "Test Postgres Service",
				 "10.11.22.33:4455",
				 "pg",
				 "<<pg-configuration>>")`,

			// STORES
			`INSERT INTO stores (uuid, name, summary, plugin, endpoint) VALUES
				("`+STORE_S3+`",
				 "s3",
				 "Amazon S3 Blobstore",
				 "s3",
				 "<<s3-configuration>>")`,

			// SCHEDULES
			`INSERT INTO schedules (uuid, name, summary, timespec) VALUES
				("`+SCHED_DAILY+`",
				 "Daily",
				 "Backups that should run every day",
				 "daily 3am")`,
			`INSERT INTO schedules (uuid, name, summary, timespec) VALUES
				("`+SCHED_WEEKLY+`",
				 "Weekly",
				 "Backups that should be run every Sunday",
				 "sundays at 10pm")`,

			// RETENTION POLICIES
			`INSERT INTO retention (uuid, name, summary, expiry) VALUES
				("`+RETAIN_SHORT+`",
				 "Short-Term Retention",
				 "Prefered retention policy for daily backups",
				 345600)`, // 4 days
			`INSERT INTO retention (uuid, name, summary, expiry) VALUES
				("`+RETAIN_LONG+`",
				 "Long-Term Retention",
				 "For stuff we need to keep longer than a month",
				 7776000)`, // 90 days

			// daily backup for pg -> s3, short retention
			`INSERT INTO jobs (uuid, name, summary, paused,
					target_uuid, store_uuid, schedule_uuid, retention_uuid) VALUES
				("`+PG_S3_DAILY+`",
				 "PG operational hot backups",
				 "For short-term operational restores",
				 0,
				 "`+TARGET_PG+`",
				 "`+STORE_S3+`",
				 "`+SCHED_DAILY+`",
				 "`+RETAIN_SHORT+`")`,

			// weekly backup for pg -> s3, long retention
			`INSERT INTO jobs (uuid, name, summary, paused,
					target_uuid, store_uuid, schedule_uuid, retention_uuid) VALUES
				("`+PG_S3_WEEKLY+`",
				 "Main DB weekly backup (long-term)",
				 "For long-term storage requirements",
				 0,
				 "`+TARGET_PG+`",
				 "`+STORE_S3+`",
				 "`+SCHED_WEEKLY+`",
				 "`+RETAIN_LONG+`")`,

			// daily backup for redis -> s3, long retention
			`INSERT INTO jobs (uuid, name, summary, paused,
					target_uuid, store_uuid, schedule_uuid, retention_uuid) VALUES
				("`+REDIS_S3_DAILY+`",
				 "Redis Daily Backup",
				 "mandated by TKT-1234",
				 0,
				 "`+TARGET_REDIS+`",
				 "`+STORE_S3+`",
				 "`+SCHED_DAILY+`",
				 "`+RETAIN_LONG+`")`,

			// (paused) weekly backup for redis -> s3, long retention
			`INSERT INTO jobs (uuid, name, summary, paused,
					target_uuid, store_uuid, schedule_uuid, retention_uuid) VALUES
				("`+REDIS_S3_WEEKLY+`",
				 "Redis Weekly Backup",
				 "...",
				 1,
				 "`+TARGET_REDIS+`",
				 "`+STORE_S3+`",
				 "`+SCHED_WEEKLY+`",
				 "`+RETAIN_LONG+`")`,
		)
		Ω(err).ShouldNot(HaveOccurred())
		resyncChan = make(chan int, 1)
		adhocChan = make(chan AdhocTask, 1)
		API = JobAPI{
			Data:       data,
			ResyncChan: resyncChan,
			AdhocChan:  adhocChan,
		}
	})

	AfterEach(func() {
		close(resyncChan)
		resyncChan = nil

		close(adhocChan)
		adhocChan = nil
	})

	It("should retrieve all jobs", func() {
		res := GET(API, "/v1/jobs")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + PG_S3_WEEKLY + `",
					"name"            : "Main DB weekly backup (long-term)",
					"summary"         : "For long-term storage requirements",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"paused"          : false,
					"schedule_name"   : "Weekly",
					"schedule_uuid"   : "` + SCHED_WEEKLY + `",
					"schedule_when"   : "sundays at 10pm",
					"store_name"      : "s3",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_name"     : "pg1",
					"target_uuid"     : "` + TARGET_PG + `",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>",
					"agent"           : "10.11.22.33:4455"
				},
				{
					"uuid"            : "` + PG_S3_DAILY + `",
					"name"            : "PG operational hot backups",
					"summary"         : "For short-term operational restores",
					"retention_name"  : "Short-Term Retention",
					"retention_uuid"  : "` + RETAIN_SHORT + `",
					"expiry"          : 345600,
					"schedule_name"   : "Daily",
					"schedule_uuid"   : "` + SCHED_DAILY + `",
					"schedule_when"   : "daily 3am",
					"paused"          : false,
					"store_name"      : "s3",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_name"     : "pg1",
					"target_uuid"     : "` + TARGET_PG + `",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>",
					"agent"           : "10.11.22.33:4455"
				},
				{
					"uuid"            : "` + REDIS_S3_DAILY + `",
					"name"            : "Redis Daily Backup",
					"summary"         : "mandated by TKT-1234",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"schedule_name"   : "Daily",
					"schedule_uuid"   : "` + SCHED_DAILY + `",
					"schedule_when"   : "daily 3am",
					"paused"          : false,
					"store_name"      : "s3",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_name"     : "redis-shared",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>",
					"agent"           : "10.11.22.33:4455"
				},
				{
					"uuid"            : "` + REDIS_S3_WEEKLY + `",
					"name"            : "Redis Weekly Backup",
					"summary"         : "...",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"schedule_name"   : "Weekly",
					"schedule_uuid"   : "` + SCHED_WEEKLY + `",
					"schedule_when"   : "sundays at 10pm",
					"paused"          : true,
					"store_name"      : "s3",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_name"     : "redis-shared",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>",
					"agent"           : "10.11.22.33:4455"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve jobs based on schedule UUID", func() {
		res := GET(API, "/v1/jobs?schedule="+SCHED_WEEKLY)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + PG_S3_WEEKLY + `",
					"name"            : "Main DB weekly backup (long-term)",
					"summary"         : "For long-term storage requirements",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"paused"          : false,
					"schedule_name"   : "Weekly",
					"schedule_uuid"   : "` + SCHED_WEEKLY + `",
					"schedule_when"   : "sundays at 10pm",
					"store_name"      : "s3",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_name"     : "pg1",
					"target_uuid"     : "` + TARGET_PG + `",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>",
					"agent"           : "10.11.22.33:4455"
				},
				{
					"uuid"            : "` + REDIS_S3_WEEKLY + `",
					"name"            : "Redis Weekly Backup",
					"summary"         : "...",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"schedule_name"   : "Weekly",
					"schedule_uuid"   : "` + SCHED_WEEKLY + `",
					"schedule_when"   : "sundays at 10pm",
					"paused"          : true,
					"store_name"      : "s3",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_name"     : "redis-shared",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>",
					"agent"           : "10.11.22.33:4455"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve jobs based on retention policy UUID", func() {
		res := GET(API, "/v1/jobs?retention="+RETAIN_SHORT)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + PG_S3_DAILY + `",
					"name"            : "PG operational hot backups",
					"summary"         : "For short-term operational restores",
					"retention_name"  : "Short-Term Retention",
					"retention_uuid"  : "` + RETAIN_SHORT + `",
					"expiry"          : 345600,
					"schedule_name"   : "Daily",
					"schedule_uuid"   : "` + SCHED_DAILY + `",
					"schedule_when"   : "daily 3am",
					"paused"          : false,
					"store_name"      : "s3",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_name"     : "pg1",
					"target_uuid"     : "` + TARGET_PG + `",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>",
					"agent"           : "10.11.22.33:4455"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve jobs based on target UUID", func() {
		res := GET(API, "/v1/jobs?target="+TARGET_PG)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + PG_S3_WEEKLY + `",
					"name"            : "Main DB weekly backup (long-term)",
					"summary"         : "For long-term storage requirements",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"paused"          : false,
					"schedule_name"   : "Weekly",
					"schedule_uuid"   : "` + SCHED_WEEKLY + `",
					"schedule_when"   : "sundays at 10pm",
					"store_name"      : "s3",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_name"     : "pg1",
					"target_uuid"     : "` + TARGET_PG + `",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>",
					"agent"           : "10.11.22.33:4455"
				},
				{
					"uuid"            : "` + PG_S3_DAILY + `",
					"name"            : "PG operational hot backups",
					"summary"         : "For short-term operational restores",
					"retention_name"  : "Short-Term Retention",
					"retention_uuid"  : "` + RETAIN_SHORT + `",
					"expiry"          : 345600,
					"schedule_name"   : "Daily",
					"schedule_uuid"   : "` + SCHED_DAILY + `",
					"schedule_when"   : "daily 3am",
					"paused"          : false,
					"store_name"      : "s3",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_name"     : "pg1",
					"target_uuid"     : "` + TARGET_PG + `",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>",
					"agent"           : "10.11.22.33:4455"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve only unpaused jobs", func() {
		res := GET(API, "/v1/jobs?paused=f")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + PG_S3_WEEKLY + `",
					"name"            : "Main DB weekly backup (long-term)",
					"summary"         : "For long-term storage requirements",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"paused"          : false,
					"schedule_name"   : "Weekly",
					"schedule_uuid"   : "` + SCHED_WEEKLY + `",
					"schedule_when"   : "sundays at 10pm",
					"store_name"      : "s3",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_name"     : "pg1",
					"target_uuid"     : "` + TARGET_PG + `",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>",
					"agent"           : "10.11.22.33:4455"
				},
				{
					"uuid"            : "` + PG_S3_DAILY + `",
					"name"            : "PG operational hot backups",
					"summary"         : "For short-term operational restores",
					"retention_name"  : "Short-Term Retention",
					"retention_uuid"  : "` + RETAIN_SHORT + `",
					"expiry"          : 345600,
					"schedule_name"   : "Daily",
					"schedule_uuid"   : "` + SCHED_DAILY + `",
					"schedule_when"   : "daily 3am",
					"paused"          : false,
					"store_name"      : "s3",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_name"     : "pg1",
					"target_uuid"     : "` + TARGET_PG + `",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>",
					"agent"           : "10.11.22.33:4455"
				},
				{
					"uuid"            : "` + REDIS_S3_DAILY + `",
					"name"            : "Redis Daily Backup",
					"summary"         : "mandated by TKT-1234",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"schedule_name"   : "Daily",
					"schedule_uuid"   : "` + SCHED_DAILY + `",
					"schedule_when"   : "daily 3am",
					"paused"          : false,
					"store_name"      : "s3",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_name"     : "redis-shared",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>",
					"agent"           : "10.11.22.33:4455"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve only paused jobs", func() {
		res := GET(API, "/v1/jobs?paused=t")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + REDIS_S3_WEEKLY + `",
					"name"            : "Redis Weekly Backup",
					"summary"         : "...",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"schedule_name"   : "Weekly",
					"schedule_uuid"   : "` + SCHED_WEEKLY + `",
					"schedule_when"   : "sundays at 10pm",
					"paused"          : true,
					"store_name"      : "s3",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_name"     : "redis-shared",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>",
					"agent"           : "10.11.22.33:4455"
				}
			]`))
	})

	It("should retrieve jobs that match a search pattern", func() {
		res := GET(API, "/v1/jobs?name=Redis")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + REDIS_S3_DAILY + `",
					"name"            : "Redis Daily Backup",
					"summary"         : "mandated by TKT-1234",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"schedule_name"   : "Daily",
					"schedule_uuid"   : "` + SCHED_DAILY + `",
					"schedule_when"   : "daily 3am",
					"paused"          : false,
					"store_name"      : "s3",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_name"     : "redis-shared",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>",
					"agent"           : "10.11.22.33:4455"
				},
				{
					"uuid"            : "` + REDIS_S3_WEEKLY + `",
					"name"            : "Redis Weekly Backup",
					"summary"         : "...",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"schedule_name"   : "Weekly",
					"schedule_uuid"   : "` + SCHED_WEEKLY + `",
					"schedule_when"   : "sundays at 10pm",
					"paused"          : true,
					"store_name"      : "s3",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_name"     : "redis-shared",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>",
					"agent"           : "10.11.22.33:4455"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve jobs that match a search pattern, which is case-insensitive", func() {
		res := GET(API, "/v1/jobs?name=redis")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + REDIS_S3_DAILY + `",
					"name"            : "Redis Daily Backup",
					"summary"         : "mandated by TKT-1234",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"schedule_name"   : "Daily",
					"schedule_uuid"   : "` + SCHED_DAILY + `",
					"schedule_when"   : "daily 3am",
					"paused"          : false,
					"store_name"      : "s3",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_name"     : "redis-shared",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>",
					"agent"           : "10.11.22.33:4455"
				},
				{
					"uuid"            : "` + REDIS_S3_WEEKLY + `",
					"name"            : "Redis Weekly Backup",
					"summary"         : "...",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"schedule_name"   : "Weekly",
					"schedule_uuid"   : "` + SCHED_WEEKLY + `",
					"schedule_when"   : "sundays at 10pm",
					"paused"          : true,
					"store_name"      : "s3",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_name"     : "redis-shared",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>",
					"agent"           : "10.11.22.33:4455"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("can filter jobs based on composite matches", func() {
		res := GET(API, "/v1/jobs?paused=f&schedule="+SCHED_WEEKLY)
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + PG_S3_WEEKLY + `",
					"name"            : "Main DB weekly backup (long-term)",
					"summary"         : "For long-term storage requirements",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"paused"          : false,
					"schedule_name"   : "Weekly",
					"schedule_uuid"   : "` + SCHED_WEEKLY + `",
					"schedule_when"   : "sundays at 10pm",
					"store_name"      : "s3",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_name"     : "pg1",
					"target_uuid"     : "` + TARGET_PG + `",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>",
					"agent"           : "10.11.22.33:4455"
				}
			]`))
		Ω(res.Code).Should(Equal(200))

		res = GET(API, "/v1/jobs?paused=f&schedule="+SCHED_WEEKLY+"&retention="+RETAIN_SHORT)
		Ω(res.Body.String()).Should(MatchJSON(`[]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("can retrieve a single job by UUID", func() {
		res := GET(API, "/v1/job/"+PG_S3_WEEKLY)
		Ω(res.Body.String()).Should(MatchJSON(`{
				"uuid"            : "` + PG_S3_WEEKLY + `",
				"name"            : "Main DB weekly backup (long-term)",
				"summary"         : "For long-term storage requirements",
				"retention_name"  : "Long-Term Retention",
				"retention_uuid"  : "` + RETAIN_LONG + `",
				"expiry"          : 7776000,
				"paused"          : false,
				"schedule_name"   : "Weekly",
				"schedule_uuid"   : "` + SCHED_WEEKLY + `",
				"schedule_when"   : "sundays at 10pm",
				"store_name"      : "s3",
				"store_uuid"      : "` + STORE_S3 + `",
				"store_plugin"    : "s3",
				"store_endpoint"  : "<<s3-configuration>>",
				"target_name"     : "pg1",
				"target_uuid"     : "` + TARGET_PG + `",
				"target_plugin"   : "pg",
				"target_endpoint" : "<<pg-configuration>>",
				"agent"           : "10.11.22.33:4455"
			}`))
		Ω(res.Code).Should(Equal(200))
	})

	It("returns a 404 for unknown UUIDs", func() {
		res := GET(API, "/v1/job/"+SCHED_WEEKLY) // it's a schedule...
		Ω(res.Code).Should(Equal(404))
	})

	It("can create new jobs", func() {
		res := POST(API, "/v1/jobs", WithJSON(`{
			"name"      : "My New Job",
			"summary"   : "A new job",
			"target"    : "`+TARGET_PG+`",
			"store"     : "`+STORE_S3+`",
			"schedule"  : "`+SCHED_WEEKLY+`",
			"retention" : "`+RETAIN_SHORT+`"
		}`))
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchRegexp(`{"ok":"created","uuid":"[a-z0-9-]+"}`))
		Eventually(resyncChan).Should(Receive())
	})

	It("requires the `name', `store', `target', `schedule', and `retention' keys to create a new job", func() {
		res := POST(API, "/v1/jobs", "{}")
		Ω(res.Code).Should(Equal(400))
		Ω(res.Body.String()).Should(Equal(`{"missing":["name","store","target","schedule","retention"]}`))
	})

	It("can update existing jobs", func() {
		res := GET(API, "/v1/jobs?paused=t")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + REDIS_S3_WEEKLY + `",
					"name"            : "Redis Weekly Backup",
					"summary"         : "...",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"schedule_name"   : "Weekly",
					"schedule_uuid"   : "` + SCHED_WEEKLY + `",
					"schedule_when"   : "sundays at 10pm",
					"paused"          : true,
					"store_name"      : "s3",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_name"     : "redis-shared",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>",
					"agent"           : "10.11.22.33:4455"
				}
			]`))

		res = PUT(API, "/v1/job/"+REDIS_S3_WEEKLY, WithJSON(`{
			"name"      : "Redis WEEKLY backups",
			"summary"   : "...",
			"target"    : "`+TARGET_REDIS+`",
			"store"     : "`+STORE_S3+`",
			"retention" : "`+RETAIN_SHORT+`",
			"schedule"  : "`+SCHED_WEEKLY+`",
			"paused"    : true
		}`))
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{"ok":"updated"}`))
		Eventually(resyncChan).Should(Receive())

		res = GET(API, "/v1/jobs?paused=t")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + REDIS_S3_WEEKLY + `",
					"name"            : "Redis WEEKLY backups",
					"summary"         : "...",
					"retention_name"  : "Short-Term Retention",
					"retention_uuid"  : "` + RETAIN_SHORT + `",
					"expiry"          : 345600,
					"schedule_name"   : "Weekly",
					"schedule_uuid"   : "` + SCHED_WEEKLY + `",
					"schedule_when"   : "sundays at 10pm",
					"paused"          : true,
					"store_name"      : "s3",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_name"     : "redis-shared",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>",
					"agent"           : "10.11.22.33:4455"
				}
			]`))
	})

	It("requires the `name', `store', `target', `schedule', and `retention' keys to update an existing job", func() {
		res := PUT(API, "/v1/job/"+REDIS_S3_WEEKLY, "{}")
		Ω(res.Code).Should(Equal(400))
		Ω(res.Body.String()).Should(Equal(`{"missing":["name","store","target","schedule","retention"]}`))
	})

	It("can delete jobs", func() {
		res := GET(API, "/v1/jobs?paused=t")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + REDIS_S3_WEEKLY + `",
					"name"            : "Redis Weekly Backup",
					"summary"         : "...",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"schedule_name"   : "Weekly",
					"schedule_uuid"   : "` + SCHED_WEEKLY + `",
					"schedule_when"   : "sundays at 10pm",
					"paused"          : true,
					"store_name"      : "s3",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_name"     : "redis-shared",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>",
					"agent"           : "10.11.22.33:4455"
				}
			]`))

		res = DELETE(API, "/v1/job/"+REDIS_S3_WEEKLY)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{"ok":"deleted"}`))
		Eventually(resyncChan).Should(Receive())

		res = GET(API, "/v1/jobs?paused=t")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[]`))
	})

	It("can pause jobs", func() {
		res := GET(API, "/v1/jobs?paused=f&schedule="+SCHED_WEEKLY)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + PG_S3_WEEKLY + `",
					"name"            : "Main DB weekly backup (long-term)",
					"summary"         : "For long-term storage requirements",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"paused"          : false,
					"schedule_name"   : "Weekly",
					"schedule_uuid"   : "` + SCHED_WEEKLY + `",
					"schedule_when"   : "sundays at 10pm",
					"store_name"      : "s3",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_name"     : "pg1",
					"target_uuid"     : "` + TARGET_PG + `",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>",
					"agent"           : "10.11.22.33:4455"
				}
			]`))

		res = POST(API, "/v1/job/"+PG_S3_WEEKLY+"/pause", "")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{"ok":"paused"}`))
		Eventually(resyncChan).Should(Receive())

		res = GET(API, "/v1/jobs?paused=f&schedule="+SCHED_WEEKLY)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[]`))
	})

	It("fails to pause non-existent jobs", func() {
		res := POST(API, "/v1/job/"+uuid.NewRandom().String()+"/pause", "")
		Ω(res.Code).Should(Equal(404))
	})

	It("can unpause jobs", func() {
		res := GET(API, "/v1/jobs?paused=t")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "` + REDIS_S3_WEEKLY + `",
					"name"            : "Redis Weekly Backup",
					"summary"         : "...",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"schedule_name"   : "Weekly",
					"schedule_uuid"   : "` + SCHED_WEEKLY + `",
					"schedule_when"   : "sundays at 10pm",
					"paused"          : true,
					"store_name"      : "s3",
					"store_uuid"      : "` + STORE_S3 + `",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_name"     : "redis-shared",
					"target_uuid"     : "` + TARGET_REDIS + `",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>",
					"agent"           : "10.11.22.33:4455"
				}
			]`))

		res = POST(API, "/v1/job/"+REDIS_S3_WEEKLY+"/unpause", "")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{"ok":"unpaused"}`))
		Eventually(resyncChan).Should(Receive())

		res = GET(API, "/v1/jobs?paused=t")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[]`))
	})

	It("fails to unpause non-existent jobs", func() {
		res := POST(API, "/v1/job/"+uuid.NewRandom().String()+"/unpause", "")
		Ω(res.Code).Should(Equal(404))
	})

	It("can rerun unpaused jobs", func() {
		res := POST(API, "/v1/job/"+PG_S3_WEEKLY+"/run", "")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{"ok":"scheduled"}`))
		Eventually(adhocChan).Should(Receive())
	})

	It("can rerun paused jobs", func() {
		res := POST(API, "/v1/job/"+REDIS_S3_WEEKLY+"/run", "")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`{"ok":"scheduled"}`))
		Eventually(adhocChan).Should(Receive())
	})

	It("validates JSON payloads", func() {
		JSONValidated(API, "POST", "/v1/jobs")
		JSONValidated(API, "PUT", "/v1/job/"+REDIS_S3_WEEKLY)
		JSONValidated(API, "POST", "/v1/job/"+REDIS_S3_WEEKLY+"/run")
	})

	It("ignores other HTTP methods", func() {
		for _, method := range []string{"PUT", "DELETE", "PATCH", "OPTIONS", "TRACE"} {
			NotImplemented(API, method, "/v1/jobs", nil)
		}

		for _, method := range []string{"GET", "HEAD", "POST", "PATCH", "OPTIONS", "TRACE"} {
			NotImplemented(API, method, "/v1/jobs/sub/requests", nil)
			NotImplemented(API, method, "/v1/job/sub/requests", nil)
		}
	})

	It("ignores malformed UUIDs", func() {
		for _, id := range []string{"malformed-uuid-01234", "(abcdef-01234-56-789)"} {
			NotImplemented(API, "PUT", fmt.Sprintf("/v1/job/%s", id), nil)
			NotImplemented(API, "POST", fmt.Sprintf("/v1/job/%s/pause", id), nil)
			NotImplemented(API, "POST", fmt.Sprintf("/v1/job/%s/unpause", id), nil)
		}
	})
})
