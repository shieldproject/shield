package db

type v1Schema struct{}

func (s v1Schema) Deploy(db *DB) error {
	err := db.Exec(`CREATE TABLE schema_info (
               version INTEGER
             )`)
	if err != nil {
		return err
	}

	err = db.Exec(`INSERT INTO schema_info VALUES (1)`)
	if err != nil {
		return err
	}

	err = db.Exec(`CREATE TABLE targets (
               uuid      UUID PRIMARY KEY,
               name      TEXT,
               summary   TEXT,
               plugin    TEXT NOT NULL,
               endpoint  TEXT NOT NULL,
               agent     TEXT NOT NULL
             )`)
	if err != nil {
		return err
	}

	err = db.Exec(`CREATE TABLE stores (
               uuid      UUID PRIMARY KEY,
               name      TEXT,
               summary   TEXT,
               plugin    TEXT NOT NULL,
               endpoint  TEXT NOT NULL
             )`)
	if err != nil {
		return err
	}

	err = db.Exec(`CREATE TABLE schedules (
               uuid      UUID PRIMARY KEY,
               name      TEXT,
               summary   TEXT,
               timespec  TEXT NOT NULL
             )`)
	if err != nil {
		return err
	}

	err = db.Exec(`CREATE TABLE retention (
               uuid     UUID PRIMARY KEY,
               name     TEXT,
               summary  TEXT,
               expiry   INTEGER NOT NULL
             )`)
	if err != nil {
		return err
	}

	err = db.Exec(`CREATE TABLE jobs (
               uuid            UUID PRIMARY KEY,
               target_uuid     UUID NOT NULL,
               store_uuid      UUID NOT NULL,
               schedule_uuid   UUID NOT NULL,
               retention_uuid  UUID NOT NULL,
               priority        INTEGER DEFAULT 50,
               paused          BOOLEAN,
               name            TEXT,
               summary         TEXT
             )`)
	if err != nil {
		return err
	}

	err = db.Exec(`CREATE TABLE archives (
               uuid         UUID PRIMARY KEY,
               target_uuid  UUID NOT NULL,
               store_uuid   UUID NOT NULL,
               store_key    TEXT NOT NULL,

               taken_at     INTEGER NOT NULL,
               expires_at   INTEGER NOT NULL,
               notes        TEXT DEFAULT ''
             )`)
	if err != nil {
		return err
	}

	err = db.Exec(`CREATE TABLE tasks (
               uuid      UUID PRIMARY KEY,
               owner     TEXT,
               op        TEXT NOT NULL,

               job_uuid      UUID,
               archive_uuid  UUID,
               target_uuid   UUID,

               status       TEXT NOT NULL,
               requested_at INTEGER NOT NULL,
               started_at   INTEGER,
               stopped_at   INTEGER,

               log       TEXT
             )`)
	if err != nil {
		return err
	}

	return nil
}
