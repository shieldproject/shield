package db

type v5Schema struct{}

func (s v5Schema) Deploy(db *DB) error {
	var err error

	err = db.Exec(`ALTER TABLE targets ADD compression TEXT NOT NULL DEFAULT 'none'`)
	if err != nil {
		return err
	}

	err = db.Exec(`UPDATE targets SET compression = 'bzip2'`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE archives ADD compression TEXT NOT NULL DEFAULT 'none'`)
	if err != nil {
		return err
	}

	err = db.Exec(`UPDATE archives SET compression = 'bzip2'`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE tasks ADD compression TEXT NOT NULL DEFAULT ''`)
	if err != nil {
		return err
	}

	err = db.Exec(`UPDATE tasks SET compression = 'bzip2' WHERE op = 'backup'`)
	if err != nil {
		return err
	}

	err = db.Exec(`UPDATE schema_info set version = 5`)
	if err != nil {
		return err
	}

	return nil
}
