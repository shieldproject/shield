package db

import (
	"fmt"
	"strings"
)

// Dialect identifies the SQL dialect for placeholder and syntax differences.
type Dialect int

const (
	DialectSQLite3 Dialect = iota
	DialectMySQL
	DialectPostgreSQL
)

// DetectDialect returns the appropriate Dialect for a database/sql driver name.
func DetectDialect(driver string) Dialect {
	switch driver {
	case "postgres", "pgx":
		return DialectPostgreSQL
	case "mysql":
		return DialectMySQL
	default:
		return DialectSQLite3
	}
}

// Rebind converts ? placeholders to the dialect's native format.
// PostgreSQL uses $1, $2, ... while MySQL and SQLite3 use ? as-is.
func Rebind(dialect Dialect, query string) string {
	if dialect != DialectPostgreSQL {
		return query
	}

	var buf strings.Builder
	buf.Grow(len(query) + 16)
	n := 1
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			fmt.Fprintf(&buf, "$%d", n)
			n++
		} else {
			buf.WriteByte(query[i])
		}
	}
	return buf.String()
}

// IsNoSuchTable returns true if err indicates a missing table,
// across all supported database backends.
func IsNoSuchTable(err error, table string) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	switch {
	case msg == "no such table: "+table:
		return true
	case msg == fmt.Sprintf(`pq: relation "%s" does not exist`, table):
		return true
	case strings.HasPrefix(msg, "Error 1146: Table"):
		return true
	default:
		return false
	}
}
