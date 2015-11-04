package api_test

import (
	. "api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	"net/http"
)

var _ = Describe("/v1/jobs API", func() {
	var API http.Handler

	STORE_S3 := `05c3d005-f968-452f-bd59-bee8e79ab982:= `
	TARGET_REDIS := `66be7c43-6c57-4391-8ea9-e770d6ab5e9e:= `
	TARGET_PG := `fab00c82-aac3-4e5f-8a2f-c534f81cdee3:= `
	SCHED_DAILY := `590eddbd-426f-408c-981b-9cf1faf2669e:= `
	SCHED_WEEKLY := `fce33a96-d352-480f-b04a-db7f2c14e98f:= `
	RETAIN_SHORT := `848ff67e-f857-47bd-9692-ae5f2be85674:= `
	RETAIN_LONG := `c5fca8e0-7d40-4cff-8dec-5f0df36ecee9:= `

	BeforeEach(func() {
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
				("abc-def-0001",
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
				("abc-def-0002",
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
				("abc-def-0003",
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
				("abc-def-0004",
				 "Redis Weekly Backup",
				 "...",
				 1,
				 "`+TARGET_REDIS+`",
				 "`+STORE_S3+`",
				 "`+SCHED_WEEKLY+`",
				 "`+RETAIN_LONG+`")`,
		)
		Ω(err).ShouldNot(HaveOccurred())
		API = JobAPI{Data: data}
	})

	It("should retrieve all jobs", func() {
		res := GET(API, "/v1/jobs")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "abc-def-0002",
					"name"            : "Main DB weekly backup (long-term)",
					"summary"         : "For long-term storage requirements",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"paused"          : false,
					"schedule_name"   : "Weekly",
					"schedule_uuid"   : "` + SCHED_WEEKLY + `",
					"schedule"        : "sundays at 10pm",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>"
				},
				{
					"uuid"            : "abc-def-0001",
					"name"            : "PG operational hot backups",
					"summary"         : "For short-term operational restores",
					"retention_name"  : "Short-Term Retention",
					"retention_uuid"  : "` + RETAIN_SHORT + `",
					"expiry"          : 345600,
					"schedule_name"   : "Daily",
					"schedule_uuid"   : "` + SCHED_DAILY + `",
					"schedule"        : "daily 3am",
					"paused"          : false,
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>"
				},
				{
					"uuid"            : "abc-def-0003",
					"name"            : "Redis Daily Backup",
					"summary"         : "mandated by TKT-1234",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"schedule_name"   : "Daily",
					"schedule_uuid"   : "` + SCHED_DAILY + `",
					"schedule"        : "daily 3am",
					"paused"          : false,
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>"
				},
				{
					"uuid"            : "abc-def-0004",
					"name"            : "Redis Weekly Backup",
					"summary"         : "...",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"schedule_name"   : "Weekly",
					"schedule_uuid"   : "` + SCHED_WEEKLY + `",
					"schedule"        : "sundays at 10pm",
					"paused"          : true,
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve jobs based on schedule UUID", func() {
		res := GET(API, "/v1/jobs?schedule=" + SCHED_WEEKLY)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "abc-def-0002",
					"name"            : "Main DB weekly backup (long-term)",
					"summary"         : "For long-term storage requirements",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"paused"          : false,
					"schedule_name"   : "Weekly",
					"schedule_uuid"   : "` + SCHED_WEEKLY + `",
					"schedule"        : "sundays at 10pm",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>"
				},
				{
					"uuid"            : "abc-def-0004",
					"name"            : "Redis Weekly Backup",
					"summary"         : "...",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"schedule_name"   : "Weekly",
					"schedule_uuid"   : "` + SCHED_WEEKLY + `",
					"schedule"        : "sundays at 10pm",
					"paused"          : true,
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve jobs based on retention policy UUID", func() {
		res := GET(API, "/v1/jobs?retention=" + RETAIN_SHORT)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "abc-def-0001",
					"name"            : "PG operational hot backups",
					"summary"         : "For short-term operational restores",
					"retention_name"  : "Short-Term Retention",
					"retention_uuid"  : "` + RETAIN_SHORT + `",
					"expiry"          : 345600,
					"schedule_name"   : "Daily",
					"schedule_uuid"   : "` + SCHED_DAILY + `",
					"schedule"        : "daily 3am",
					"paused"          : false,
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve jobs based on target UUID", func() {
		res := GET(API, "/v1/jobs?target=" + TARGET_PG)
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "abc-def-0002",
					"name"            : "Main DB weekly backup (long-term)",
					"summary"         : "For long-term storage requirements",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"paused"          : false,
					"schedule_name"   : "Weekly",
					"schedule_uuid"   : "` + SCHED_WEEKLY + `",
					"schedule"        : "sundays at 10pm",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>"
				},
				{
					"uuid"            : "abc-def-0001",
					"name"            : "PG operational hot backups",
					"summary"         : "For short-term operational restores",
					"retention_name"  : "Short-Term Retention",
					"retention_uuid"  : "` + RETAIN_SHORT + `",
					"expiry"          : 345600,
					"schedule_name"   : "Daily",
					"schedule_uuid"   : "` + SCHED_DAILY + `",
					"schedule"        : "daily 3am",
					"paused"          : false,
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve only unpaused jobs", func() {
		res := GET(API, "/v1/jobs?paused=f")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "abc-def-0002",
					"name"            : "Main DB weekly backup (long-term)",
					"summary"         : "For long-term storage requirements",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"paused"          : false,
					"schedule_name"   : "Weekly",
					"schedule_uuid"   : "` + SCHED_WEEKLY + `",
					"schedule"        : "sundays at 10pm",
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>"
				},
				{
					"uuid"            : "abc-def-0001",
					"name"            : "PG operational hot backups",
					"summary"         : "For short-term operational restores",
					"retention_name"  : "Short-Term Retention",
					"retention_uuid"  : "` + RETAIN_SHORT + `",
					"expiry"          : 345600,
					"schedule_name"   : "Daily",
					"schedule_uuid"   : "` + SCHED_DAILY + `",
					"schedule"        : "daily 3am",
					"paused"          : false,
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_plugin"   : "pg",
					"target_endpoint" : "<<pg-configuration>>"
				},
				{
					"uuid"            : "abc-def-0003",
					"name"            : "Redis Daily Backup",
					"summary"         : "mandated by TKT-1234",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"schedule_name"   : "Daily",
					"schedule_uuid"   : "` + SCHED_DAILY + `",
					"schedule"        : "daily 3am",
					"paused"          : false,
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})

	It("should retrieve only paused jobs", func() {
		res := GET(API, "/v1/jobs?paused=t")
		Ω(res.Code).Should(Equal(200))
		Ω(res.Body.String()).Should(MatchJSON(`[
				{
					"uuid"            : "abc-def-0004",
					"name"            : "Redis Weekly Backup",
					"summary"         : "...",
					"retention_name"  : "Long-Term Retention",
					"retention_uuid"  : "` + RETAIN_LONG + `",
					"expiry"          : 7776000,
					"schedule_name"   : "Weekly",
					"schedule_uuid"   : "` + SCHED_WEEKLY + `",
					"schedule"        : "sundays at 10pm",
					"paused"          : true,
					"store_plugin"    : "s3",
					"store_endpoint"  : "<<s3-configuration>>",
					"target_plugin"   : "redis",
					"target_endpoint" : "<<redis-configuration>>"
				}
			]`))
		Ω(res.Code).Should(Equal(200))
	})
})
