package db

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Dialect", func() {
	Describe("DetectDialect", func() {
		It("detects sqlite3", func() {
			Ω(DetectDialect("sqlite3")).Should(Equal(DialectSQLite3))
		})

		It("detects mysql", func() {
			Ω(DetectDialect("mysql")).Should(Equal(DialectMySQL))
		})

		It("detects postgres", func() {
			Ω(DetectDialect("postgres")).Should(Equal(DialectPostgreSQL))
		})

		It("detects pgx as postgres", func() {
			Ω(DetectDialect("pgx")).Should(Equal(DialectPostgreSQL))
		})

		It("defaults to sqlite3 for unknown drivers", func() {
			Ω(DetectDialect("unknown")).Should(Equal(DialectSQLite3))
		})
	})

	Describe("Rebind", func() {
		Context("for SQLite3", func() {
			It("passes through ? placeholders unchanged", func() {
				Ω(Rebind(DialectSQLite3, "SELECT * FROM t WHERE a = ? AND b = ?")).
					Should(Equal("SELECT * FROM t WHERE a = ? AND b = ?"))
			})
		})

		Context("for MySQL", func() {
			It("passes through ? placeholders unchanged", func() {
				Ω(Rebind(DialectMySQL, "SELECT * FROM t WHERE a = ? AND b = ?")).
					Should(Equal("SELECT * FROM t WHERE a = ? AND b = ?"))
			})
		})

		Context("for PostgreSQL", func() {
			It("converts ? to $1, $2, ...", func() {
				Ω(Rebind(DialectPostgreSQL, "SELECT * FROM t WHERE a = ? AND b = ?")).
					Should(Equal("SELECT * FROM t WHERE a = $1 AND b = $2"))
			})

			It("handles queries with no placeholders", func() {
				Ω(Rebind(DialectPostgreSQL, "SELECT * FROM t")).
					Should(Equal("SELECT * FROM t"))
			})

			It("handles INSERT with many placeholders", func() {
				Ω(Rebind(DialectPostgreSQL, "INSERT INTO t (a, b, c) VALUES (?, ?, ?)")).
					Should(Equal("INSERT INTO t (a, b, c) VALUES ($1, $2, $3)"))
			})

			It("handles double-digit placeholder numbers", func() {
				q := "INSERT INTO t VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
				expected := "INSERT INTO t VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)"
				Ω(Rebind(DialectPostgreSQL, q)).Should(Equal(expected))
			})
		})
	})

	Describe("IsNoSuchTable", func() {
		It("returns false for nil error", func() {
			Ω(IsNoSuchTable(nil, "schema_info")).Should(BeFalse())
		})

		It("matches SQLite3 missing table error", func() {
			err := fmt.Errorf("no such table: schema_info")
			Ω(IsNoSuchTable(err, "schema_info")).Should(BeTrue())
		})

		It("matches PostgreSQL missing table error", func() {
			err := fmt.Errorf(`pq: relation "schema_info" does not exist`)
			Ω(IsNoSuchTable(err, "schema_info")).Should(BeTrue())
		})

		It("matches MySQL missing table error", func() {
			err := fmt.Errorf("Error 1146: Table 'db.schema_info' doesn't exist")
			Ω(IsNoSuchTable(err, "schema_info")).Should(BeTrue())
		})

		It("returns false for unrelated errors", func() {
			err := fmt.Errorf("connection refused")
			Ω(IsNoSuchTable(err, "schema_info")).Should(BeFalse())
		})
	})
})
