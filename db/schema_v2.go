package db

import (
	"fmt"
)

type v2Schema struct{}

func (s v2Schema) Deploy(db *DB) error {
	var err error

	var textType string
	switch db.Driver {
	case "mysql":
		textType = "VARCHAR(255)"
	case "postgres", "sqlite3":
		textType = "TEXT"
	default:
		return fmt.Errorf("unsupported database driver '%s'", db.Driver)
	}

	err = db.Exec(fmt.Sprintf(`ALTER TABLE archives ADD COLUMN purge_reason %s DEFAULT ''`, textType))
	if err != nil {
		return err
	}

	err = db.Exec(fmt.Sprintf(`ALTER TABLE archives ADD COLUMN status %s DEFAULT 'valid'`, textType))
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
