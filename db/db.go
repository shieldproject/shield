package db

import (
	"database/sql"
	"fmt"
	"os"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/pborman/uuid"

	"github.com/starkandwayne/shield/core/bus"
)

var GlobalTenantUUID = uuid.NIL.String()

type DB struct {
	connection *sqlx.DB
	Driver     string
	DSN        string

	exclusive sync.Mutex
	cache     map[string]*sql.Stmt
	bus       *bus.Bus
}

// Connect to the backend database
func Connect(file string) (*DB, error) {
	db := &DB{
		Driver: "sqlite3",
		DSN:    file,
	}

	connection, err := sqlx.Open(db.Driver, db.DSN)
	if err != nil {
		return nil, err
	}
	db.connection = connection

	if db.cache == nil {
		db.cache = make(map[string]*sql.Stmt)
	}

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
		db.cache = make(map[string]*sql.Stmt)
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

	s, err := db.statement(sql)
	if err != nil {
		return err
	}
	_, err = s.Exec(args...)
	if err != nil {
		return err
	}

	return nil
}

// Execute a named, data query (SELECT)
func (db *DB) Query(sql string, args ...interface{}) (*sql.Rows, error) {
	db.exclusive.Lock()
	defer db.exclusive.Unlock()

	s, err := db.statement(sql)
	if err != nil {
		return nil, err
	}

	if os.Getenv("SHIELD_DB_TRACE") != "" {
		fmt.Fprintf(os.Stdout, "\n DB is querying '%s' args:['%s'] \n", sql, args)
	}

	r, err := s.Query(args...)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Execute a data query (SELECT) and return how many rows were returned
func (db *DB) Count(sql string, args ...interface{}) (uint, error) {
	r, err := db.Query(sql, args...)
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

// Return the prepared statement for a given SQL query
func (db *DB) statement(sql string) (*sql.Stmt, error) {
	if db.connection == nil {
		return nil, fmt.Errorf("Not connected to database")
	}

	sql = db.connection.Rebind(sql)
	if _, ok := db.cache[sql]; !ok {
		stmt, err := db.connection.Prepare(sql)
		if err != nil {
			return nil, err
		}
		db.cache[sql] = stmt
	}

	if q, ok := db.cache[sql]; ok {
		return q, nil
	}

	return nil, fmt.Errorf("Weird bug: query '%s' is still not properly prepared", sql)
}

// Generate a randomized UUID
func RandomID() string {
	return uuid.NewRandom().String()
}
