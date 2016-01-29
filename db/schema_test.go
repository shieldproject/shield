package db_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// sql drivers
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	. "github.com/starkandwayne/shield/db"
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
				Ω(v).Should(Equal(2))
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

	Describe("Schema Version Interrogation", func() {
		It("should return an error for a bad database connection", func() {
			db := &DB{
				Driver: "postgres",
				DSN:    "host=127.86.86.86, port=8686",
			}

			db.Connect()
			_, err := db.SchemaVersion()
			Ω(err).Should(HaveOccurred())
		})
	})
})
