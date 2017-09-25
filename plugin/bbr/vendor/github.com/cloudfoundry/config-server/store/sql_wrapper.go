package store

import (
	"github.com/BurntSushi/migration"
)

type SQLWrapper struct {
}

func NewSQLWrapper() SQLWrapper {
	return SQLWrapper{}
}

func (w SQLWrapper) Open(driverName, dataSourceName string, migrations []migration.Migrator) (IDb, error) {
	db, err := migration.Open(driverName, dataSourceName, migrations)
	return NewDbWrapper(db), err
}
