package db

type v11Schema struct{}

func (s v11Schema) Deploy(db *DB) error {
	var err error

	// set the tenant_uuid column to NOT NULL
	err = db.Exec(`CREATE TABLE jobs_new (
	               uuid               UUID PRIMARY KEY,
	               target_uuid        UUID NOT NULL,
	               store_uuid         UUID NOT NULL,
	               tenant_uuid        UUID NOT NULL,
	               name               TEXT NOT NULL,
	               summary            TEXT NOT NULL,
	               schedule           TEXT NOT NULL,
	               keep_n             INTEGER NOT NULL DEFAULT 0,
	               keep_days          INTEGER NOT NULL DEFAULT 0,
	               next_run           INTEGER DEFAULT 0,
	               priority           INTEGER DEFAULT 50,
	               paused             BOOLEAN NOT NULL DEFAULT 0,
	               fixed_key          INTEGER DEFAULT 0,
	               healthy            BOOLEAN NOT NULL DEFAULT 0
	             )`)
	if err != nil {
		return err
	}
	err = db.Exec(`INSERT INTO jobs_new (uuid, target_uuid, store_uuid, tenant_uuid,
	                                     name, summary, schedule, keep_n, keep_days,
	                                     next_run, priority, paused, fixed_key, healthy)
	                              SELECT j.uuid, j.target_uuid, j.store_uuid, j.tenant_uuid,
	                                     j.name, j.summary, j.schedule, j.keep_n, j.keep_days,
	                                     j.next_run, j.priority, IFNULL(j.paused, 0), j.fixed_key, IFNULL(j.healthy, 0)
	                                FROM jobs j`)
	if err != nil {
		return err
	}
	err = db.Exec(`DROP TABLE jobs`)
	if err != nil {
		return err
	}
	err = db.Exec(`ALTER TABLE jobs_new RENAME TO jobs`)
	if err != nil {
		return err
	}

	err = db.Exec(`UPDATE schema_info set version = 11`)
	if err != nil {
		return err
	}

	return nil
}
