package db

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"
)

func Database(sqls ...string) (*DB, error) {
	db, err := Connect(":memory:")
	if err != nil {
		return nil, err
	}

	if _, err := db.Setup(0); err != nil {
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
				var err error
				db, err = Connect(":memory:")

				Ω(err).ShouldNot(HaveOccurred())
				Ω(db.Connected()).Should(BeTrue())
			})

			It("should not create tables until Setup() is called", func() {
				Ω(db.Exec("SELECT * FROM schema_info")).
					Should(HaveOccurred())
			})

			It("should create tables during Setup()", func() {
				_, err := db.Setup(0)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(db.Exec("SELECT * FROM schema_info")).
					Should(Succeed())
			})

			It("should set the version number in schema_info", func() {
				_, err := db.Setup(0)
				Ω(err).ShouldNot(HaveOccurred())

				r, err := db.Query(`SELECT version FROM schema_info`)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(r).ShouldNot(BeNil())
				Ω(r.Next()).Should(BeTrue())

				var v int
				Ω(r.Scan(&v)).Should(Succeed())
				Ω(v).Should(Equal(8))
			})

			It("creates the correct tables", func() {
				_, err := db.Setup(0)
				Ω(err).ShouldNot(HaveOccurred())

				tableExists := func(table string) {
					sql := fmt.Sprintf("SELECT * FROM %s", table)
					Ω(db.Exec(sql)).Should(Succeed())
				}

				tableExists("targets")
				tableExists("stores")
				tableExists("jobs")
				tableExists("archives")
				tableExists("tasks")
			})
		})
	})

	Describe("Schema Version Interrogation", func() {
		It("should return an error for a bad database connection", func() {
			db, _ := Connect("/path/to/no/such/file")
			_, err := db.SchemaVersion()
			Ω(err).Should(HaveOccurred())
		})
	})
})
