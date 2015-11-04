package db_test

import (
	"fmt"

	. "db"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"
)

func Database(sqls ...string) (*DB, error) {
	var db *DB
	db = &DB{
		Driver: "sqlite3",
		DSN:    ":memory:",
	}

	if err := db.Connect(); err != nil {
		return nil, err
	}

	if err := db.Setup(); err != nil {
		db.Disconnect()
		return nil, err
	}

	for _, s := range sqls {
		err := db.Exec(s)
		if err != nil {
			db.Disconnect()
			return nil, err
		}
	}

	return db, nil
}

var _ = Describe("Database Schema", func() {
	Describe("Initializing the schema", func() {
		Context("With a new database", func() {
			var db *DB

			BeforeEach(func() {
				db = &DB{
					Driver: "sqlite3",
					DSN:    ":memory:",
				}

				Ω(db.Connect()).Should(Succeed())
				Ω(db.Connected()).Should(BeTrue())
			})

			It("should not create tables until Setup() is called", func() {
				Ω(db.Exec("SELECT * FROM schema_info")).
					Should(HaveOccurred())
			})

			It("should create tables during Setup()", func() {
				Ω(db.Setup()).Should(Succeed())
				Ω(db.Exec("SELECT * FROM schema_info")).
					Should(Succeed())
			})

			It("should set the version number in schema_info", func() {
				Ω(db.Setup()).Should(Succeed())

				r, err := db.Query(`SELECT version FROM schema_info`)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(r).ShouldNot(BeNil())
				Ω(r.Next()).Should(BeTrue())

				var v int
				Ω(r.Scan(&v)).Should(Succeed())
				Ω(v).Should(Equal(1))
			})

			It("creates the correct tables", func() {
				Ω(db.Setup()).Should(Succeed())

				tableExists := func(table string) {
					sql := fmt.Sprintf("SELECT * FROM %s", table)
					Ω(db.Exec(sql)).Should(Succeed())
				}

				tableExists("targets")
				tableExists("stores")
				tableExists("schedules")
				tableExists("retention")
				tableExists("jobs")
				tableExists("archives")
				tableExists("tasks")
			})
		})
	})
	Describe("Retrieving Jobs", func() {
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
				db, err = Database(
					`INSERT INTO jobs (uuid, target_uuid, store_uuid, schedule_uuid, retention_uuid, paused, name, summary) VALUES
						("6809c55a-8250-11e5-8bcf-feff819cdc9f",
						 "c957554e-6fe0-4ae9-816d-307e20f155cb",
						 "32d810b6-073b-4296-8c68-6544a91760f9",
						 "b36eeea3-9f5c-46f2-a337-6de5344e8d0f",
						 "52c6f512-5c90-4364-998c-f849fa416243",
						 "t",
						 "job 1",
						 "First test job in queue")`,

					`INSERT INTO targets (uuid, name, summary, plugin, endpoint) VALUES
						 ("c957554e-6fe0-4ae9-816d-307e20f155cb",
						 "redis-shared",
						 "Shared Redis services for CF",
						 "redis",
						 "<<redis-configuration>>")`,

					`INSERT INTO stores (uuid, name, summary, plugin, endpoint) VALUES
						("32d810b6-073b-4296-8c68-6544a91760f9",
						 "redis-shared",
						 "Shared Redis services for CF",
						 "redis",
						 "<<redis-configuration>>")`,

					`INSERT INTO schedules (uuid, name, summary, timespec) VALUES
						("b36eeea3-9f5c-46f2-a337-6de5344e8d0f",
						 "Weekly Backups",
						 "A schedule for weekly bosh-blobs, during normal maintenance windows",
						 "sundays at 3:15am")`,

					`INSERT INTO retention (uuid, name, summary, expiry) VALUES
						("52c6f512-5c90-4364-998c-f849fa416243",
						 "Hourly Retention",
						 "Keep backups for 1 hour",
						 3600)`,

					`INSERT INTO jobs (uuid, target_uuid, store_uuid, schedule_uuid, retention_uuid, paused, name, summary) VALUES
						("b20ca8b6-8250-11e5-8bcf-feff819cdc9f",
						 "cf65a73e-79c1-48e8-b706-23ec7644c721",
						 "9eb022c4-227f-44f1-b11b-b6d8bcfc3c4f",
						 "4b8432b9-b5e9-46d5-b23e-ba70983a2acc",
						 "f7dee6c2-59c4-439d-9d92-04046b8beb68",
						 "f",
						 "job 2",
						 "Second test job in queue")`,

					`INSERT INTO targets (uuid, name, summary, plugin, endpoint) VALUES
						("cf65a73e-79c1-48e8-b706-23ec7644c721",
						 "s3",
						 "Amazon S3 Blobstore",
						 "s3",
						 "<<s3-configuration>>")`,

					`INSERT INTO stores (uuid, name, summary, plugin, endpoint) VALUES
						("9eb022c4-227f-44f1-b11b-b6d8bcfc3c4f",
						 "s3",
						 "Amazon S3 Blobstore",
						 "s3",
						 "<<s3-configuration>>")`,

					`INSERT INTO schedules (uuid, name, summary, timespec) VALUES
						("4b8432b9-b5e9-46d5-b23e-ba70983a2acc",
						 "Daily Backups",
						 "Use for daily (11-something-at-night) bosh-blobs",
						 "daily at 11:24pm")`,

					`INSERT INTO retention (uuid, name, summary, expiry) VALUES
						("f7dee6c2-59c4-439d-9d92-04046b8beb68",
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
})
