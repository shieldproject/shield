package db_test

import (
	"database/sql"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	. "github.com/starkandwayne/shield/db"
)

var _ = Describe("Database", func() {
	Describe("Connecting to the database", func() {
		Context("With an invalid DSN", func() {
			It("should fail", func() {
				db := &DB{
					Driver: "invalid",
					DSN:    "does-not-matter",
				}

				Ω(db.Connect()).Should(HaveOccurred())
				Ω(db.Connected()).Should(BeFalse())
				Ω(db.Disconnect()).Should(Succeed())
			})
		})

		Context("With an in-memory SQLite database", func() {
			It("should succeed", func() {
				db := &DB{
					Driver: "sqlite3",
					DSN:    ":memory:",
				}

				Ω(db.Connect()).Should(Succeed())
				Ω(db.Connected()).Should(BeTrue())
				Ω(db.Disconnect()).Should(Succeed())
			})
		})
	})

	Describe("Running SQL queries", func() {
		var db *DB

		BeforeEach(func() {
			db = &DB{
				Driver: "sqlite3",
				DSN:    ":memory:",
			}
			Ω(db.Connect()).Should(Succeed())
		})

		AfterEach(func() {
			db.Disconnect()
		})

		Context("With an empty database", func() {
			It("can create tables", func() {
				Ω(db.Exec(`CREATE TABLE things (type TEXT, number INTEGER)`)).Should(Succeed())
			})
		})

		numberOfThingsIn := func(r *sql.Rows) int {
			var n int

			Ω(r).ShouldNot(BeNil())
			Ω(r.Next()).Should(BeTrue())
			Ω(r.Scan(&n)).Should(Succeed())
			return n
		}

		Context("With an empty table", func() {
			BeforeEach(func() {
				db.Disconnect()
				Ω(db.Connect()).Should(Succeed())

				Ω(db.Exec(`CREATE TABLE things (type TEXT, number INTEGER)`)).Should(Succeed())
			})

			It("can insert records", func() {
				Ω(db.Exec(`INSERT INTO things (type, number) VALUES ($1, 0)`, "monkey")).Should(Succeed())

				r, err := db.Query(`SELECT number FROM things WHERE type = $1`, "monkey")
				Ω(err).Should(Succeed())
				Ω(numberOfThingsIn(r)).Should(Equal(0))
			})

			It("can update records", func() {
				Ω(db.Exec(`INSERT INTO things (type, number) VALUES ($1, 0)`, "monkey")).Should(Succeed())
				Ω(db.Exec(`UPDATE things SET number = number + $1 WHERE type = $2`, 42, "monkey")).Should(Succeed())

				r, err := db.Query(`SELECT number FROM things WHERE type = $1`, "monkey")
				Ω(err).Should(Succeed())
				Ω(numberOfThingsIn(r)).Should(Equal(42))
			})

			It("can handle queries without arguments", func() {
				Ω(db.Exec(`INSERT INTO things (type, number) VALUES ($1, 0)`, "monkey")).Should(Succeed())
				Ω(db.Exec(`UPDATE things SET number = number + $1 WHERE type = $2`, 13, "monkey")).Should(Succeed())

				r, err := db.Query(`SELECT number FROM things WHERE type = "monkey"`)
				Ω(err).Should(Succeed())
				Ω(numberOfThingsIn(r)).Should(Equal(13))
			})

			It("can alias queries", func() {
				Ω(db.Alias("new-thing", `INSERT INTO things (type, number) VALUES ($1, 0)`)).Should(Succeed())
				Ω(db.Alias("increment", `UPDATE things SET number = number + $1 WHERE type = $2`)).Should(Succeed())
				Ω(db.Alias("how-many", `SELECT number FROM things WHERE type = "monkey"`)).Should(Succeed())

				Ω(db.Exec("new-thing", "monkey")).Should(Succeed())
				Ω(db.Exec("increment", 13, "monkey")).Should(Succeed())

				r, err := db.Query("how-many")
				Ω(err).Should(Succeed())
				Ω(numberOfThingsIn(r)).Should(Equal(13))
			})

			It("can run arbitrary SQL", func() {
				Ω(db.Exec("INSERT INTO things (type, number) VALUES ($1, $2)", "lion", 3)).
					Should(Succeed())

				r, err := db.Query(`SELECT number FROM things WHERE type = $1`, "lion")
				Ω(err).Should(Succeed())
				Ω(numberOfThingsIn(r)).Should(Equal(3))
			})
		})

		Context("With malformed SQL queries", func() {
			It("propagates errors from sql driver", func() {
				Ω(db.Exec(`DO STUFF IN SQL`)).Should(HaveOccurred())
			})
		})
	})
})
