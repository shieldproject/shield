package db

type v2Schema struct{}

func (s v2Schema) Deploy(db *DB) error {
	err := db.Exec(`ALTER TABLE archives ADD COLUMN purge_reason TEXT DEFAULT ''`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE archives ADD COLUMN status TEXT DEFAULT 'valid'`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE tasks ADD COLUMN store_uuid UUID`)
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
