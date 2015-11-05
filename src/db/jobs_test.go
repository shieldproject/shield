package db_test
/*
import (
	. "db"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"
)

var _ = Describe("Retrieving Jobs", func() {
	var db *DB
	BeforeEach(func() {
		var err error
		db, err = Database()
		Ω(err).ShouldNot(HaveOccurred())
	})
	Context("With an empty database", func() {
		It("should return an empty list of jobs", func() {
			jobs, err := db.GetAllJobs()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(len(jobs)).Should(Equal(0))
		})
	})
	Context("With a non-empty database", func() {
		BeforeEach(func() {
			var err error
			JOB_ONE_UUID := `6809c55a-8250-11e5-8bcf-feff819cdc9f`
			TARGET_ONE_UUID := `c957554e-6fe0-4ae9-816d-307e20f155cb`
			STORE_ONE_UUID := `32d810b6-073b-4296-8c68-6544a91760f9`
			SCHED_ONE_UUID := `b36eeea3-9f5c-46f2-a337-6de5344e8d0f`
			RETEN_ONE_UUID := `52c6f512-5c90-4364-998c-f849fa416243`
			JOB_TWO_UUID := `b20ca8b6-8250-11e5-8bcf-feff819cdc9f`
			TARGET_TWO_UUID := `cf65a73e-79c1-48e8-b706-23ec7644c721`
			STORE_TWO_UUID := `9eb022c4-227f-44f1-b11b-b6d8bcfc3c4f`
			SCHED_TWO_UUID := `4b8432b9-b5e9-46d5-b23e-ba70983a2acc`
			RETEN_TWO_UUID := `f7dee6c2-59c4-439d-9d92-04046b8beb68`
			db, err = Database(
				`INSERT INTO jobs (uuid, target_uuid, store_uuid, schedule_uuid, retention_uuid, paused, name, summary) VALUES
					("`+JOB_ONE_UUID+`",
					 "`+TARGET_ONE_UUID+`",
					 "`+STORE_ONE_UUID+`",
					 "`+SCHED_ONE_UUID+`",
					 "`+RETEN_ONE_UUID+`",
					 "t",
					 "job 1",
					 "First test job in queue")`,
				`INSERT INTO targets (uuid, name, summary, plugin, endpoint) VALUES
					 ("`+TARGET_ONE_UUID+`",
					 "redis-shared",
					 "Shared Redis services for CF",
					 "redis",
					 "<<redis-configuration>>")`,
				`INSERT INTO stores (uuid, name, summary, plugin, endpoint) VALUES
					("`+STORE_ONE_UUID+`",
					 "redis-shared",
					 "Shared Redis services for CF",
					 "redis",
					 "<<redis-configuration>>")`,
				`INSERT INTO schedules (uuid, name, summary, timespec) VALUES
					("`+SCHED_ONE_UUID+`",
					 "Weekly Backups",
					 "A schedule for weekly bosh-blobs, during normal maintenance windows",
					 "sundays at 3:15am")`,
				`INSERT INTO retention (uuid, name, summary, expiry) VALUES
					("`+RETEN_ONE_UUID+`",
					 "Hourly Retention",
					 "Keep backups for 1 hour",
					 3600)`,
				`INSERT INTO jobs (uuid, target_uuid, store_uuid, schedule_uuid, retention_uuid, paused, name, summary) VALUES
					("`+JOB_TWO_UUID+`",
					 "`+TARGET_TWO_UUID+`",
					 "`+STORE_TWO_UUID+`",
					 "`+SCHED_TWO_UUID+`",
					 "`+RETEN_TWO_UUID+`",
					 "f",
					 "job 2",
					 "Second test job in queue")`,
				`INSERT INTO targets (uuid, name, summary, plugin, endpoint) VALUES
					("`+TARGET_TWO_UUID+`",
					 "s3",
					 "Amazon S3 Blobstore",
					 "s3",
					 "<<s3-configuration>>")`,
				`INSERT INTO stores (uuid, name, summary, plugin, endpoint) VALUES
					("`+STORE_TWO_UUID+`",
					 "s3",
					 "Amazon S3 Blobstore",
					 "s3",
					 "<<s3-configuration>>")`,
				`INSERT INTO schedules (uuid, name, summary, timespec) VALUES
					("`+SCHED_TWO_UUID+`",
					 "Daily Backups",
					 "Use for daily (11-something-at-night) bosh-blobs",
					 "daily at 11:24pm")`,
				`INSERT INTO retention (uuid, name, summary, expiry) VALUES
					("`+RETEN_TWO_UUID+`",
					 "Yearly Retention",
					 "Keep backups for 1 year",
					 31536000)`,
			)
			Ω(err).ShouldNot(HaveOccurred())
		})
		It("should return a complete list of jobs", func() {
			jobs, err := db.GetAllJobs()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(len(jobs)).Should(Equal(2))
		})
	})
})
*/
