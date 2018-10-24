package db

type v6Schema struct{}

func (s v6Schema) Deploy(db *DB) error {
	var err error

	// set the tenant_uuid column to NOT NULL
	err = db.exec(`CREATE TABLE jobs_new (
	               uuid               UUID PRIMARY KEY,
	               target_uuid        UUID NOT NULL,
	               store_uuid         UUID NOT NULL,
	               tenant_uuid        UUID NOT NULL,
	               name               TEXT,
	               summary            TEXT,
	               schedule           TEXT NOT NULL,
	               next_run           INTEGER DEFAULT 0,
	               retention_uuid     UUID NOT NULL,
	               priority           INTEGER DEFAULT 50,
	               paused             BOOLEAN,
	               fixed_key          INTEGER DEFAULT 0
	             )`)
	if err != nil {
		return err
	}
	err = db.exec(`INSERT INTO jobs_new (uuid, target_uuid, store_uuid, tenant_uuid,
	                                     schedule, next_run, retention_uuid,
	                                     priority, paused, name, summary)
	                              SELECT uuid, target_uuid, store_uuid, tenant_uuid,
	                                     schedule, next_run, retention_uuid,
	                                     priority, paused, name, summary
	                                FROM jobs`)
	if err != nil {
		return err
	}
	err = db.exec(`DROP TABLE jobs`)
	if err != nil {
		return err
	}
	err = db.exec(`ALTER TABLE jobs_new RENAME TO jobs`)
	if err != nil {
		return err
	}

	err = db.exec(`UPDATE schema_info set version = 6`)
	if err != nil {
		return err
	}

	return nil
}
