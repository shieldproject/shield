package db

type v3Schema struct{}

func (s v3Schema) Deploy(db *DB) error {
	err := db.Exec(`ALTER TABLE tasks ADD COLUMN target_plugin TEXT DEFAULT ''`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE tasks ADD COLUMN target_endpoint TEXT DEFAULT ''`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE tasks ADD COLUMN store_plugin TEXT DEFAULT ''`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE tasks ADD COLUMN store_endpoint TEXT DEFAULT ''`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE tasks ADD COLUMN restore_key TEXT DEFAULT ''`)
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

	err = db.Exec(`ALTER TABLE tasks ADD COLUMN agent TEXT DEFAULT ''`)
	if err != nil {
		return err
	}

	err = db.Exec(`UPDATE schema_info set version = 3`)
	if err != nil {
		return err
	}

	return nil
}
