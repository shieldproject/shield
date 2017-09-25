package store

import "database/sql"

type RowWrapper struct {
	row *sql.Row
}

func NewRowWrapper(row *sql.Row) RowWrapper {
	return RowWrapper{row}
}

func (w RowWrapper) Scan(dest ...interface{}) error {
	return w.row.Scan(dest...)
}
