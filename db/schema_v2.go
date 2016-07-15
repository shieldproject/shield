package db

import (
	"fmt"
)

type v2Schema struct{}

func (s v2Schema) Deploy(db *DB) error {
	var err error

	switch db.Driver {
	case "mysql":
		err = db.Exec(`ALTER TABLE archives ADD COLUMN purge_reason TEXT NOT NULL DEFAULT ''`)
	case "postgres", "sqlite3":
		err = db.Exec(`ALTER TABLE archives ADD COLUMN purge_reason TEXT DEFAULT ''`)
	default:
		return fmt.Errorf("unsupported database driver '%s'", db.Driver)
	}
	if err != nil {
		return err
	}

	var defaultValue string
	switch db.Driver {
	case "mysql":
		defaultValue = ""
	case "postgres", "sqlite3":
		defaultValue = "valid"
	}

	err = db.Exec(fmt.Sprintf(`ALTER TABLE archives ADD COLUMN status TEXT DEFAULT '%s'`, defaultValue))
	if err != nil {
		return err
	}

	var uuidType string
	switch db.Driver {
	case "mysql":
		uuidType = "VARCHAR(36)"
	case "postgres", "sqlite3":
		uuidType = "UUID"
	}
	err = db.Exec(fmt.Sprintf(`ALTER TABLE tasks ADD COLUMN store_uuid %s`, uuidType))
	if err != nil {
		return err
	}

	err = db.Exec(`UPDATE archives SET status = 'valid'`)
	if err != nil {
		return err
	}

	err = db.Exec(`UPDATE schema_info set version = 2`)
	if err != nil {
		return err
	}

	return nil
}
