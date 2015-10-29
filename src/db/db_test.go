package db_test

import (
	. "db"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"database/sql"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"
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
				Ω(db.Disconnect()).ShouldNot(HaveOccurred())
			})
		})

		Context("With an in-memory SQLite database", func() {
			It("should succeed", func() {
				db := &DB{
					Driver: "sqlite3",
					DSN:    ":memory:",
				}

				Ω(db.Connect()).ShouldNot(HaveOccurred())
				Ω(db.Connected()).Should(BeTrue())
				Ω(db.Disconnect()).ShouldNot(HaveOccurred())
			})
		})
	})

	Describe("Registering (cached) SQL queries", func() {
		var db *DB

		BeforeEach(func() {
			db = &DB{
				Driver: "sqlite3",
				DSN:    ":memory:",
			}

			Ω(db.Connect()).ShouldNot(HaveOccurred())
			Ω(db.Connected()).Should(BeTrue())
		})

		Context("With an empty query cache", func() {
			It("has nothing registered", func() {
				Ω(db.Cached("my-query")).Should(BeFalse())
			})

			It("can register a SQL command", func() {
				Ω(db.Cache("my-query", "SELECT * FROM table")).ShouldNot(HaveOccurred())
				Ω(db.Cached("my-query")).Should(BeTrue())
			})

			It("can register the same name multiple times", func() {
				Ω(db.Cache("my-query", "SELECT * FROM table")).ShouldNot(HaveOccurred())
				Ω(db.Cache("my-query", "SELECT * FROM other_table")).ShouldNot(HaveOccurred())
				Ω(db.Cached("my-query")).Should(BeTrue())
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

			Ω(db.Connect()).ShouldNot(HaveOccurred())

			Ω(db.Cache("schema", `CREATE TABLE things (type TEXT, number INTEGER)`)).
				ShouldNot(HaveOccurred())
			Ω(db.Cache("new-thing", `INSERT INTO things (type, number) VALUES (?, 0)`)).
				ShouldNot(HaveOccurred())
			Ω(db.Cache("increase", `UPDATE things SET number = number + ? WHERE type = ?`)).
				ShouldNot(HaveOccurred())
			Ω(db.Cache("how-many?", `SELECT number FROM things WHERE type = ?`)).
				ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			db.Disconnect()
		})

		Context("With an empty database", func() {
			It("can create tables", func() {
				Ω(db.Exec("schema")).ShouldNot(HaveOccurred())
			})
		})

		numberOfThingsIn := func(r *sql.Rows) int {
			var n int

			Ω(r).ShouldNot(BeNil())
			Ω(r.Next()).Should(BeTrue())
			Ω(r.Scan(&n)).ShouldNot(HaveOccurred())
			return n
		}

		Context("With an empty table", func() {
			BeforeEach(func() {
				db.Disconnect()
				Ω(db.Connect()).ShouldNot(HaveOccurred())

				Ω(db.Exec("schema")).ShouldNot(HaveOccurred())
			})

			It("can insert records", func() {
				Ω(db.Exec("new-thing", "monkey")).ShouldNot(HaveOccurred())

				r, err := db.Query("how-many?", "monkey")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(numberOfThingsIn(r)).Should(Equal(0))
			})

			It("can update records", func() {
				Ω(db.Exec("new-thing", "monkey")).ShouldNot(HaveOccurred())
				Ω(db.Exec("increase", 42, "monkey")).ShouldNot(HaveOccurred())

				r, err := db.Query("how-many?", "monkey")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(numberOfThingsIn(r)).Should(Equal(42))
			})

			It("raises an error if an uncached query is requested for Exec()", func() {
				Ω(db.Exec("not-a-query")).Should(HaveOccurred())
			})

			It("raises an error if an uncached query is requested for Query()", func() {
				r, err := db.Query("not-a-query")
				Ω(r).Should(BeNil())
				Ω(err).Should(HaveOccurred())
			})

			It("can run arbitrary SQL", func() {
				Ω(db.ExecOnce("INSERT INTO things (type, number) VALUES (?, ?)", "lion", 3)).
					ShouldNot(HaveOccurred())

				r, err := db.Query("how-many?", "lion")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(numberOfThingsIn(r)).Should(Equal(3))
			})
		})

		Context("With malformed SQL queries", func() {
			BeforeEach(func() {
				Ω(db.Cache("error", `DO STUFF IN SQL`)).
					ShouldNot(HaveOccurred())
			})

			It("propagates errors from sql driver", func() {
				Ω(db.Exec("errors")).Should(HaveOccurred())
			})
		})
	})
})
