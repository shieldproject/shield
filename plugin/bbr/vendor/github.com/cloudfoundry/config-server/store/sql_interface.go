package store

import "github.com/BurntSushi/migration"

type ISql interface {
	Open(driverName, dataSourceName string, migrations []migration.Migrator) (IDb, error)
}
