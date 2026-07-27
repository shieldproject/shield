package db

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
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

// Vendor codes for "that relation does not exist".
const (
	pgUndefinedTable    = "42P01" // SQLSTATE undefined_table
	mysqlErrNoSuchTable = 1146    // ER_NO_SUCH_TABLE
)

// IsNoSuchTable returns true if err indicates a missing table,
// across all supported database backends.
//
// Every driver reports this differently, and none of them render it in a form
// that survives string comparison: pgx formats errors as
// "ERROR: ... (SQLSTATE 42P01)" -- nothing like lib/pq's "pq: relation ..." --
// and go-sql-driver emits "Error 1146 (42S02): ..." whenever the server
// supplies a SQLSTATE, which every supported MySQL does. So match on the
// structured error values the drivers return instead.
func IsNoSuchTable(err error, table string) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgUndefinedTable && namesTable(pgErr.Message, table)
	}

	var myErr *mysql.MySQLError
	if errors.As(err, &myErr) {
		return myErr.Number == mysqlErrNoSuchTable && namesTable(myErr.Message, table)
	}

	// go-sqlite3 returns a sqlite3.Error carrying only the generic
	// SQLITE_ERROR code, so its message is the sole signal. Importing
	// go-sqlite3 to type-assert would force cgo on every consumer of this
	// package for no gain, so match the message.
	msg := err.Error()
	return strings.Contains(msg, "no such table") && namesTable(msg, table)
}

// namesTable reports whether a driver's error message refers to table.
// SQLite says "no such table: jobs", PostgreSQL says `relation "jobs" does not
// exist", and MySQL says "Table 'shield.jobs' doesn't exist" -- the name is
// always present, never with a consistent delimiter. Compare identifier tokens
// so that a message about "jobs" is not read as one about "job".
func namesTable(msg, table string) bool {
	if table == "" {
		return false
	}
	for _, token := range strings.FieldsFunc(msg, func(r rune) bool {
		return !(r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r))
	}) {
		if strings.EqualFold(token, table) {
			return true
		}
	}
	return false
}
