package db

import (
	"fmt"
)

type v3Schema struct{}

func (s v3Schema) Deploy(db *DB) error {
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

	err = db.Exec(fmt.Sprintf(`ALTER TABLE tasks ADD COLUMN target_plugin %s DEFAULT ''`, textType))
	if err != nil {
		return err
	}

	err = db.Exec(fmt.Sprintf(`ALTER TABLE tasks ADD COLUMN target_endpoint %s DEFAULT ''`, textType))
	if err != nil {
		return err
	}

	err = db.Exec(fmt.Sprintf(`ALTER TABLE tasks ADD COLUMN store_plugin %s DEFAULT ''`, textType))
	if err != nil {
		return err
	}

	err = db.Exec(fmt.Sprintf(`ALTER TABLE tasks ADD COLUMN store_endpoint %s DEFAULT ''`, textType))
	if err != nil {
		return err
	}

	err = db.Exec(fmt.Sprintf(`ALTER TABLE tasks ADD COLUMN restore_key %s DEFAULT ''`, textType))
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE tasks ADD COLUMN timeout_at INTEGER DEFAULT NULL`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE tasks ADD COLUMN attempts INTEGER DEFAULT 0`)
	if err != nil {
		return err
	}

	err = db.Exec(fmt.Sprintf(`ALTER TABLE tasks ADD COLUMN agent %s DEFAULT ''`, textType))
	if err != nil {
		return err
	}

	err = db.Exec(`UPDATE schema_info set version = 3`)
	if err != nil {
		return err
	}

	return nil
}
