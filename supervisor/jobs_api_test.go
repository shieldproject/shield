package supervisor_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	. "github.com/starkandwayne/shield/supervisor"

	"github.com/starkandwayne/shield/db"
)

var _ = Describe("/v1/jobs API", func() {
	var API http.Handler
	var resyncChan chan int
	var adhocChan chan *db.Task

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

	databaseEntries := []string{
		// TARGETS
		`INSERT INTO targets (uuid, name, summary, agent, plugin, endpoint) VALUES
				("` + TARGET_REDIS + `",
				 "redis-shared",
				 "Shared Redis services for CF",
				 "10.11.22.33:4455",
				 "redis",
				 "<<redis-configuration>>")`,
		`INSERT INTO targets (uuid, name, summary, agent, plugin, endpoint) VALUES
				("` + TARGET_PG + `",
				 "pg1",
				 "Test Postgres Service",
				 "10.11.22.33:4455",
				 "pg",
				 "<<pg-configuration>>")`,

		// STORES
		`INSERT INTO stores (uuid, name, summary, plugin, endpoint) VALUES
				("` + STORE_S3 + `",
				 "s3",
				 "Amazon S3 Blobstore",
				 "s3",
				 "<<s3-configuration>>")`,

		// SCHEDULES
		`INSERT INTO schedules (uuid, name, summary, timespec) VALUES
				("` + SCHED_DAILY + `",
				 "Daily",
				 "Backups that should run every day",
				 "daily 3am")`,
		`INSERT INTO schedules (uuid, name, summary, timespec) VALUES
				("` + SCHED_WEEKLY + `",
				 "Weekly",
				 "Backups that should be run every Sunday",
				 "sundays at 10pm")`,

		// RETENTION POLICIES
		`INSERT INTO retention (uuid, name, summary, expiry) VALUES
				("` + RETAIN_SHORT + `",
				 "Short-Term Retention",
				 "Prefered retention policy for daily backups",
				 345600)`, // 4 days
		`INSERT INTO retention (uuid, name, summary, expiry) VALUES
				("` + RETAIN_LONG + `",
				 "Long-Term Retention",
				 "For stuff we need to keep longer than a month",
				 7776000)`, // 90 days
		// daily backup for pg -> s3, short retention
		`INSERT INTO jobs (uuid, name, summary, paused,
					target_uuid, store_uuid, schedule_uuid, retention_uuid) VALUES
				("` + PG_S3_DAILY + `",
				 "PG operational hot backups",
				 "For short-term operational restores",
				 0,
				 "` + TARGET_PG + `",
				 "` + STORE_S3 + `",
				 "` + SCHED_DAILY + `",
				 "` + RETAIN_SHORT + `")`,

		// weekly backup for pg -> s3, long retention
		`INSERT INTO jobs (uuid, name, summary, paused,
					target_uuid, store_uuid, schedule_uuid, retention_uuid) VALUES
				("` + PG_S3_WEEKLY + `",
				 "Main DB weekly backup (long-term)",
				 "For long-term storage requirements",
				 0,
				 "` + TARGET_PG + `",
				 "` + STORE_S3 + `",
				 "` + SCHED_WEEKLY + `",
				 "` + RETAIN_LONG + `")`,

		// daily backup for redis -> s3, long retention
		`INSERT INTO jobs (uuid, name, summary, paused,
					target_uuid, store_uuid, schedule_uuid, retention_uuid) VALUES
				("` + REDIS_S3_DAILY + `",
				 "Redis Daily Backup",
				 "mandated by TKT-1234",
				 0,
				 "` + TARGET_REDIS + `",
				 "` + STORE_S3 + `",
				 "` + SCHED_DAILY + `",
				 "` + RETAIN_LONG + `")`,

		// (paused) weekly backup for redis -> s3, long retention
		`INSERT INTO jobs (uuid, name, summary, paused,
					target_uuid, store_uuid, schedule_uuid, retention_uuid) VALUES
				("` + REDIS_S3_WEEKLY + `",
				 "Redis Weekly Backup",
				 "...",
				 1,
				 "` + TARGET_REDIS + `",
				 "` + STORE_S3 + `",
				 "` + SCHED_WEEKLY + `",
				 "` + RETAIN_LONG + `")`,
	}
	var data *db.DB

	BeforeEach(func() {
		var err error
		data, err = Database(databaseEntries...)
		Ω(err).ShouldNot(HaveOccurred())
		resyncChan = make(chan int, 1)
		adhocChan = make(chan *db.Task, 1)
	})

	JustBeforeEach(func() {
		API = JobAPI{
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

	Context("when running adhoc jobs", func() {
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
			It("can rerun unpaused jobs", func() {
				res := POST(API, "/v1/job/"+PG_S3_WEEKLY+"/run", "")
				Ω(res.Code).Should(Equal(200))
				expected, err := json.Marshal(map[string]string{
					"ok":        "scheduled",
					"task_uuid": testUUID,
				})
				Ω(err).ShouldNot(HaveOccurred())
				Ω(res.Body.String()).Should(MatchJSON(expected))
				Ω(<-errChan).Should(BeNil())
			})

			It("can rerun paused jobs", func() {
				res := POST(API, "/v1/job/"+REDIS_S3_WEEKLY+"/run", "")
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
				res := POST(API, "/v1/job/"+PG_S3_WEEKLY+"/run", "")
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
