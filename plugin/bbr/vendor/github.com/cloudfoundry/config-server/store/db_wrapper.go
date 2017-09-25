package store

import (
	"database/sql"
)

type DBWrapper struct {
	db *sql.DB
}

func NewDbWrapper(db *sql.DB) DBWrapper {
	return DBWrapper{db}
}

func (w DBWrapper) Exec(query string, args ...interface{}) (sql.Result, error) {
	return w.db.Exec(query, args...)
}

func (w DBWrapper) Query(query string, args ...interface{}) (IRows, error) {
	rows, err := w.db.Query(query, args...)
	return NewRowsWrapper(rows), err
}

func (w DBWrapper) QueryRow(query string, args ...interface{}) IRow {
	return NewRowWrapper(w.db.QueryRow(query, args...))
}

func (w DBWrapper) Close() {
	w.db.Close()
}

func (w DBWrapper) SetMaxOpenConns(n int) {
	w.db.SetMaxOpenConns(n)
}

func (w DBWrapper) SetMaxIdleConns(n int) {
	w.db.SetMaxIdleConns(n)
}
