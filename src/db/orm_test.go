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
			orm, err := NewORM(db)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(orm).ShouldNot(BeNil())
			Ω(orm.Setup()).ShouldNot(HaveOccurred())
		})
		Context("With an empty database", func() {
			It("should return an empty list of jobs", func() {
				jobs, err := orm.GetAllJobs()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(len(jobs)).Should(Equal(0))
			})
		})
	})
})
