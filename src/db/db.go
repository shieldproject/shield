package db

import (
	"database/sql"
	"fmt"
)

type Query struct {
	Raw    string
	Cooked *sql.Stmt
}

type DB struct {
	connection *sql.DB
	Driver     string
	DSN        string

	qCache map[string]*Query
}

// Are we connected?
func (db *DB) Connected() bool {
	if db.connection == nil {
		return false
	}
	return true
}

// Connect to the backend database
func (db *DB) Connect() error {
	connection, err := sql.Open(db.Driver, db.DSN)
	if err != nil {
		return err
	}

	db.connection = connection
	if db.qCache == nil {
		db.qCache = make(map[string]*Query)
	}
	return nil
}

// Disconnect from the backend database
func (db *DB) Disconnect() error {
	if db.connection != nil {
		if err := db.connection.Close(); err != nil {
			return err
		}
		db.connection = nil

		for _, q := range db.qCache {
			q.Cooked = nil
		}
	}
	return nil
}

// Register a SQL query
func (db *DB) Cache(name string, sql string) error {
	db.qCache[name] = &Query{Raw: sql}
	return nil
}

// Is a SQL query cached?
func (db *DB) Cached(name string) bool {
	_, ok := db.qCache[name]
	return ok
}

// Execute a one-off, non-data query (CREATE TABLE, DROP TABLE, etc.)
func (db *DB) ExecOnce(sql string, args ...interface{}) error {
	s, err := db.connection.Prepare(sql)
	if err != nil {
		return err
	}

	_, err = s.Exec(args...)
	return err
}

// Execute a named, non-data query (INSERT, UPDATE, DELETE, etc.)
func (db *DB) Exec(name string, args ...interface{}) error {
	s, err := db.statement(name)
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
func (db *DB) Query(name string, args ...interface{}) (*sql.Rows, error) {
	s, err := db.statement(name)
	if err != nil {
		return nil, err
	}

	r, err := s.Query(args...)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Return the prepared Statement for a named query
func (db *DB) statement(name string) (*sql.Stmt, error) {
	if db.connection == nil {
		return nil, fmt.Errorf("Not connected to database")
	}

	q, ok := db.qCache[name]
	if !ok {
		return nil, fmt.Errorf("Unknown query '%s'", name)
	}

	if q.Cooked == nil {
		cooked, err := db.connection.Prepare(q.Raw)
		if err != nil {
			return nil, err
		}

		q.Cooked = cooked
	}

	if q.Cooked == nil { // still?
		return nil, fmt.Errorf("Weird bug: query '%s' is still not properly prepared", name)
	}

	return q.Cooked, nil
}
