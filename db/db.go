package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/starkandwayne/goutils/log"

	. "github.com/starkandwayne/shield/timestamp"
)

func parseEpochTime(et int64) Timestamp {
	return NewTimestamp(time.Unix(et, 0).UTC())
}

type DB struct {
	connection *sql.DB
	Driver     string
	DSN        string

	qCache map[string]*sql.Stmt
	qAlias map[string]string
}

func (db *DB) Copy() *DB {
	return &DB{
		Driver: db.Driver,
		DSN:    db.DSN,
	}
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
		db.qCache = make(map[string]*sql.Stmt)
	}
	if db.qAlias == nil {
		db.qAlias = make(map[string]string)
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
		db.qCache = make(map[string]*sql.Stmt)
	}
	return nil
}

// Register a SQL query alias
func (db *DB) Alias(name string, sql string) error {
	db.qAlias[name] = sql
	return nil
}

// Execute a named, non-data query (INSERT, UPDATE, DELETE, etc.)
func (db *DB) Exec(sql_or_name string, args ...interface{}) error {
	s, err := db.statement(sql_or_name)
	if err != nil {
		return err
	}

	log.Debugf("Parameters: %v", args)
	_, err = s.Exec(args...)
	if err != nil {
		return err
	}

	return nil
}

// Execute a named, data query (SELECT)
func (db *DB) Query(sql_or_name string, args ...interface{}) (*sql.Rows, error) {
	s, err := db.statement(sql_or_name)
	if err != nil {
		return nil, err
	}

	log.Debugf("Parameters: %v", args)
	r, err := s.Query(args...)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Execute a data query (SELECT) and return how many rows were returned
func (db *DB) Count(sql_or_name string, args ...interface{}) (uint, error) {
	r, err := db.Query(sql_or_name, args...)
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

// Transparently resolve SQL aliases to real SQL query text
func (db *DB) resolve(sql_or_name string) string {
	if sql, ok := db.qAlias[sql_or_name]; ok {
		return sql
	}
	return sql_or_name
}

// Return the prepared Statement for a given SQL query
func (db *DB) statement(sql_or_name string) (*sql.Stmt, error) {
	sql := db.resolve(sql_or_name)
	if db.connection == nil {
		return nil, fmt.Errorf("Not connected to database")
	}

	log.Debugf("Executing SQL: %s", sql)

	q, ok := db.qCache[sql]
	if !ok {
		stmt, err := db.connection.Prepare(sql)
		if err != nil {
			return nil, err
		}
		db.qCache[sql] = stmt
	}

	q, ok = db.qCache[sql]
	if !ok {
		return nil, fmt.Errorf("Weird bug: query '%s' is still not properly prepared", sql)
	}
	return q, nil
}
