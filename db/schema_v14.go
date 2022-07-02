package db

type v14Schema struct{}

func (s v14Schema) Deploy(db *DB) error {
	var err error

	err = db.Exec(`ALTER TABLE jobs ADD COLUMN restore BOOLEAN NOT NULL DEFAULT 0`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE jobs ADD COLUMN restoreto_uuid UUID`)
	if err != nil {
		return err
	}

	err = db.Exec(`UPDATE schema_info set version = 14`)
	if err != nil {
		return err
	}

	return nil
}
