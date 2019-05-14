package db

type v8Schema struct{}

func (s v8Schema) Deploy(db *DB) error {
	var err error

	// track what fixups were done, and when
	err = db.Exec(`CREATE TABLE fixups (
	                 id         VARCHAR(100) PRIMARY KEY,
	                 name       TEXT,
	                 summary    TEXT,
	                 created_at INTEGER NOT NULL,
	                 applied_at INTEGER
	               )`)
	if err != nil {
		return err
	}

	err = db.Exec(`UPDATE schema_info set version = 8`)
	if err != nil {
		return err
	}

	return nil
}
