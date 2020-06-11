package db

import (
	"database/sql"
	"fmt"
	"os"
	"sync"

	"github.com/jhunt/go-log"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pborman/uuid"

	"github.com/shieldproject/shield/core/bus"
)

var GlobalTenantUUID = uuid.NIL.String()

type DB struct {
	connection *sqlx.DB
	Driver     string
	DSN        string

	exclusive sync.Mutex
	bus       *bus.Bus
}

// Connect to the backend database
func Connect(file string) (*DB, error) {
	db := &DB{
		Driver: "postgres",
		DSN:    file,
	}

	connection, err := sqlx.Open(db.Driver, db.DSN)
	if err != nil {
		return nil, err
	}
	db.connection = connection

	return db, nil
}

// Are we connected?
func (db *DB) Connected() bool {
	return db.connection != nil
}

// Disconnect from the backend database
func (db *DB) Disconnect() error {
	if db.connection != nil {
		if err := db.connection.Close(); err != nil {
			return err
		}
		db.connection = nil
	}
	return nil
}

// Have the database start sending SHIELD Bus Events to a message bus
func (db *DB) Inform(mbus *bus.Bus) {
	db.bus = mbus
}

// Execute a named, non-data query (INSERT, UPDATE, DELETE, etc.)
func (db *DB) Exec(sql string, args ...interface{}) error {
	db.exclusive.Lock()
	defer db.exclusive.Unlock()
	return db.exec(sql, args...)
}

// Execute a named, non-data query (INSERT, UPDATE, DELETE, etc.)
// The caller is responsible for locking the Mutex.
func (db *DB) exec(sql string, args ...interface{}) error {
	s, err := db.statement(sql)
	if err != nil {
		return err
	}
	_, err = s.Exec(args...)
	s.Close()
	return err
}

type queryResult struct {
	stmt *sql.Stmt
	rows *sql.Rows
}

func (r *queryResult) Next() bool {
	return r.rows.Next()
}

func (r *queryResult) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

func (r *queryResult) Close() error {
	err := r.rows.Close()
	if err != nil {
		r.stmt.Close()
		return err
	}
	return r.stmt.Close()
}

// Execute a named, data query (SELECT)
// The caller is responsible for holding the database lock, as the rows object
// returned holds a SQLite read lock until the rows object is closed.
func (db *DB) query(sql string, args ...interface{}) (*queryResult, error) {
	s, err := db.statement(sql)
	if err != nil {
		return nil, err
	}

	if os.Getenv("SHIELD_DB_TRACE") != "" {
		fmt.Fprintf(os.Stdout, "\n DB is querying '%s' args:['%s'] \n", sql, args)
	}

	r, err := s.Query(args...)
	if err != nil {
		s.Close()
		return nil, err
	}

	return &queryResult{
		stmt: s,
		rows: r,
	}, nil
}

// Execute a data query (SELECT) and return how many rows were returned
func (db *DB) Count(sql string, args ...interface{}) (uint, error) {
	db.exclusive.Lock()
	defer db.exclusive.Unlock()
	return db.count(sql, args...)
}

// Execute a data query (SELECT) and return how many rows were returned
// The caller is responsible for locking the Mutex.
func (db *DB) count(sql string, args ...interface{}) (uint, error) {
	r, err := db.query(sql, args...)
	if err != nil {
		return 0, err
	}

	var n uint = 0
	for r.Next() {
		n++
	}
	r.Close()
	return n, nil
}

// Execute a data query (SELECT) and return true if any rows exist
func (db *DB) Exists(sql string, args ...interface{}) (bool, error) {
	db.exclusive.Lock()
	defer db.exclusive.Unlock()
	return db.exists(sql, args...)
}

// Execute a data query (SELECT) and return true if any rows exist
// The caller is responsible for locking the Mutex.
func (db *DB) exists(sql string, args ...interface{}) (bool, error) {
	n, err := db.count(sql, args...)
	return n > 0, err
}

// Run some arbitrary code with the database lock held,
// properly releasing it whenever the passed function returns.
func (db *DB) exclusively(fn func() error) error {
	db.exclusive.Lock()
	defer db.exclusive.Unlock()
	return fn()
}

func (db *DB) transactionally(fn func() error) error {
	return db.exclusively(func() (err error) {
		log.Infof("beginning transaction...")
		if err = db.exec("BEGIN TRANSACTION"); err != nil {
			return
		}
		defer func() {
			if r := recover(); r != nil {
				log.Infof("rolling back (panic: %s)...", r)
				db.exec("ROLLBACK")
				panic(r)
			}

			if err != nil {
				log.Infof("rolling back (error: %s)...", err)
				db.exec("ROLLBACK")
				return
			}

			log.Infof("commiting transaction...")
			err = db.exec("COMMIT")
		}()
		return fn()
	})
}

// Return the prepared statement for a given SQL query
func (db *DB) statement(sql string) (*sql.Stmt, error) {
	if db.connection == nil {
		return nil, fmt.Errorf("Not connected to database")
	}

	return db.connection.Prepare(db.connection.Rebind(sql))
}

// Generate a randomized UUID
func RandomID() string {
	return uuid.NewRandom().String()
}
