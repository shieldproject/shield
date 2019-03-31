package db

import (
	"database/sql"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"
)

var _ = Describe("Database", func() {
	Describe("Connecting to the database", func() {
		Context("With an in-memory SQLite database", func() {
			It("should succeed", func() {
				db, err := Connect(":memory:")

				Ω(err).ShouldNot(HaveOccurred())
				Ω(db.Connected()).Should(BeTrue())
				Ω(db.Disconnect()).Should(Succeed())
			})
		})
	})

	Describe("Running SQL queries", func() {
		var db *DB

		BeforeEach(func() {
			db, _ = Connect(":memory:")
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
				var err error
				db, err = Connect(":memory:")
				Ω(err).ShouldNot(HaveOccurred())

				Ω(db.Exec(`CREATE TABLE things (type TEXT, number INTEGER)`)).Should(Succeed())
			})

			It("can insert records", func() {
				Ω(db.Exec(`INSERT INTO things (type, number) VALUES (?, 0)`, "monkey")).Should(Succeed())

				r, err := db.Query(`SELECT number FROM things WHERE type = ?`, "monkey")
				Ω(err).Should(Succeed())
				Ω(numberOfThingsIn(r)).Should(Equal(0))
			})

			It("can update records", func() {
				Ω(db.Exec(`INSERT INTO things (type, number) VALUES (?, 0)`, "monkey")).Should(Succeed())
				Ω(db.Exec(`UPDATE things SET number = number + ? WHERE type = ?`, 42, "monkey")).Should(Succeed())

				r, err := db.Query(`SELECT number FROM things WHERE type = ?`, "monkey")
				Ω(err).Should(Succeed())
				Ω(numberOfThingsIn(r)).Should(Equal(42))
			})

			It("can handle queries without arguments", func() {
				Ω(db.Exec(`INSERT INTO things (type, number) VALUES (?, 0)`, "monkey")).Should(Succeed())
				Ω(db.Exec(`UPDATE things SET number = number + ? WHERE type = ?`, 13, "monkey")).Should(Succeed())

				r, err := db.Query(`SELECT number FROM things WHERE type = "monkey"`)
				Ω(err).Should(Succeed())
				Ω(numberOfThingsIn(r)).Should(Equal(13))
			})

			It("can run arbitrary SQL", func() {
				Ω(db.Exec("INSERT INTO things (type, number) VALUES (?, ?)", "lion", 3)).
					Should(Succeed())

				r, err := db.Query(`SELECT number FROM things WHERE type = ?`, "lion")
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

	Describe("Stressing the database", func() {
		Context("With varying levels of concurrency", func() {
			var db *DB

			BeforeEach(func() {
				var err error
				db, err = Connect("file::memory:?cache=shared")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(db.Exec(`CREATE TABLE stuff (numb INTEGER, iter INTEGER)`)).Should(Succeed())
			})

			AfterEach(func() {
				db.Disconnect()
			})

			stressor := func(reply chan error, db *DB, n, times int) {
				Ω(db.Connected()).Should(BeTrue())
				for i := 0; i < times; i++ {
					err := db.Exec("INSERT INTO stuff (numb, iter) VALUES (?, ?)", n, i)
					if err != nil {
						reply <- err
						return
					}
				}
				reply <- nil
			}

			stress := func(n int) func() {
				return func() {
					reply := make(chan error, n)
					for i := 0; i < n; i++ {
						go stressor(reply, db, i, 100)
					}

					for i := 0; i < n; i++ {
						err := <-reply
						Ω(err).ShouldNot(HaveOccurred())
					}
				}
			}

			It("can handle a single writer", stress(1))
			It("can handle two concurrent writers", stress(2))
			It("can handle twenty concurrent writers", stress(20))
			It("can handle a hundred concurrent writers", stress(100))
			It("can handle a thousand concurrent writers", stress(1000))
		})
	})
})
