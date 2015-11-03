package db

import (
	"database/sql"
	"fmt"
)

type DB struct {
	connection *sql.DB
	Driver     string
	DSN        string

	qCache map[string]*sql.Stmt
	qAlias map[string]string
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

	r, err := s.Query(args...)
	if err != nil {
		return nil, err
	}

	return r, nil
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
