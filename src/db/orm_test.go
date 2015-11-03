package db_test

import (
	"fmt"

	. "db"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"
)

var _ = Describe("ORM", func() {
	Describe("Initializing the schema", func() {
		Context("With a new database", func() {
			var db *DB

			BeforeEach(func() {
				db = &DB{
					Driver: "sqlite3",
					DSN:    ":memory:",
				}

				Ω(db.Connect()).ShouldNot(HaveOccurred())
				Ω(db.Connected()).Should(BeTrue())
			})

			It("should succeed", func() {
				orm, err := NewORM(db)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(orm).ShouldNot(BeNil())
			})

			It("should not create tables until Setup() is called", func() {
				orm, err := NewORM(db)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(orm).ShouldNot(BeNil())

				Ω(db.ExecOnce("SELECT * FROM schema_info")).
					Should(HaveOccurred())
			})

			It("should create tables during Setup()", func() {
				orm, err := NewORM(db)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(orm).ShouldNot(BeNil())

				Ω(orm.Setup()).ShouldNot(HaveOccurred())

				Ω(db.ExecOnce("SELECT * FROM schema_info")).
					ShouldNot(HaveOccurred())
			})

			It("should set the version number in schema_info", func() {
				orm, err := NewORM(db)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(orm).ShouldNot(BeNil())

				Ω(orm.Setup()).ShouldNot(HaveOccurred())

				Ω(db.Cache("schema-version", `SELECT version FROM schema_info`)).
					ShouldNot(HaveOccurred())

				r, err := db.Query("schema-version")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(r).ShouldNot(BeNil())
				Ω(r.Next()).Should(BeTrue())

				var v int
				Ω(r.Scan(&v)).ShouldNot(HaveOccurred())
				Ω(v).Should(Equal(1))
			})

			It("creates the correct tables", func() {
				orm, err := NewORM(db)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(orm).ShouldNot(BeNil())

				Ω(orm.Setup()).ShouldNot(HaveOccurred())

				tableExists := func(table string) {
					sql := fmt.Sprintf("SELECT * FROM %s", table)
					Ω(db.ExecOnce(sql)).ShouldNot(HaveOccurred())
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
		var orm *ORM
		BeforeEach(func() {
			db = &DB{
				Driver: "sqlite3",
				DSN:    ":memory:",
			}

			//Read as "ShouldNot(HaveErrored())"
			Ω(db.Connect()).ShouldNot(HaveOccurred())
			Ω(db.Connected()).Should(BeTrue())

			// New ORM for all contexts
			var err error
			orm, err = NewORM(db)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(orm).ShouldNot(BeNil())
			Ω(orm.Setup()).ShouldNot(HaveOccurred())
		})
		Context("With an empty database", func() {
			It("should return an empty list of jobs", func() {
				Ω(orm).ShouldNot(BeNil())
				jobs, err := orm.GetAllJobs()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(len(jobs)).Should(Equal(0))
			})
		})
		BeforeEach(func() {
			db.Cache("new-job", `
				INSERT INTO jobs (uuid, target_uuid, store_uuid, schedule_uuid, retention_uuid, paused, name, summary) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`)

			db.Exec("new-job",
				"6809c55a-8250-11e5-8bcf-feff819cdc9f",
				"c957554e-6fe0-4ae9-816d-307e20f155cb",
				"32d810b6-073b-4296-8c68-6544a91760f9",
				"b36eeea3-9f5c-46f2-a337-6de5344e8d0f",
				"52c6f512-5c90-4364-998c-f849fa416243",
				true,
				"job 1",
				"First test job in queue")

			db.Exec("new-job",
				"b20ca8b6-8250-11e5-8bcf-feff819cdc9f",
				"cf65a73e-79c1-48e8-b706-23ec7644c721",
				"9eb022c4-227f-44f1-b11b-b6d8bcfc3c4f",
				"4b8432b9-b5e9-46d5-b23e-ba70983a2acc",
				"f7dee6c2-59c4-439d-9d92-04046b8beb68",
				false,
				"job 2",
				"Second test job in queue")
		})
		Context("With a non-empty database", func() {
			It("should return a complete list of jobs", func() {
				jobs, err := orm.GetAllJobs()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(len(jobs)).Should(Equal(2))
			})
		})
	})
})
