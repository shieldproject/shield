package db

type v13Schema struct{}

func (s v13Schema) Deploy(db *DB) error {
	var err error

	err = db.Exec(`ALTER TABLE jobs ADD COLUMN retries INT NOT NULL DEFAULT 0`)
	if err != nil {
		return err
	}

	err = db.Exec(`UPDATE schema_info set version = 13`)
	if err != nil {
		return err
	}

	return nil
}
