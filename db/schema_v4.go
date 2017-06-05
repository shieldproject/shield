package db

type v4Schema struct{}

func (s v4Schema) Deploy(db *DB) error {
	err := db.Exec(`ALTER TABLE jobs ADD COLUMN schedule TEXT`)
	if err != nil {
		return err
	}

	err = db.Exec(`UPDATE jobs SET schedule =
	                  (SELECT timespec FROM schedules
	                   WHERE schedules.uuid = jobs.schedule_uuid)`)
	if err != nil {
		return err
	}

	if db.Driver == "sqlite3" {
		err = db.Exec(`CREATE TABLE jobs_new (
               uuid            UUID PRIMARY KEY,
               target_uuid     UUID NOT NULL,
               store_uuid      UUID NOT NULL,
               schedule        TEXT NOT NULL,
               retention_uuid  UUID NOT NULL,
               priority        INTEGER DEFAULT 50,
               paused          BOOLEAN,
               name            TEXT,
               summary         TEXT
             )`)
		if err != nil {
			return err
		}

		err = db.Exec(`INSERT INTO jobs_new
		               (uuid, target_uuid, store_uuid, schedule, retention_uuid,
		                priority, paused, name, summary)
		               SELECT uuid, target_uuid, store_uuid, schedule, retention_uuid,
		                      priority, paused, name, summary FROM jobs`)
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

	} else {
		err = db.Exec(`ALTER TABLE jobs DROP COLUMN schedule_uuid`)
		if err != nil {
			return err
		}

		err = db.Exec(`ALTER TABLE jobs ALTER COLUMN schedule TEXT NOT NULL`)
		if err != nil {
			return err
		}
	}

	err = db.Exec(`DROP TABLE schedules`)
	if err != nil {
		return err
	}

	err = db.Exec(`UPDATE schema_info set version = 4`)
	if err != nil {
		return err
	}

	return nil
}
