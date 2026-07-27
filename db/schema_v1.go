package db

type v1Schema struct{}

func (s v1Schema) Deploy(db *DB) error {
	var err error

	err = db.Exec(`CREATE TABLE schema_info (
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
	                 uuid      VARCHAR(36) PRIMARY KEY,
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
	                 uuid      VARCHAR(36) PRIMARY KEY,
	                 name      TEXT,
	                 summary   TEXT,
	                 plugin    TEXT NOT NULL,
	                 endpoint  TEXT NOT NULL
	               )`)
	if err != nil {
		return err
	}

	err = db.Exec(`CREATE TABLE schedules (
	                 uuid      VARCHAR(36) PRIMARY KEY,
	                 name      TEXT,
	                 summary   TEXT,
	                 timespec  TEXT NOT NULL
	               )`)
	if err != nil {
		return err
	}

	err = db.Exec(`CREATE TABLE retention (
	                 uuid     VARCHAR(36) PRIMARY KEY,
	                 name     TEXT,
	                 summary  TEXT,
	                 expiry   INTEGER NOT NULL
	               )`)
	if err != nil {
		return err
	}

	err = db.Exec(`CREATE TABLE jobs (
	                 uuid            VARCHAR(36) PRIMARY KEY,
	                 target_uuid     VARCHAR(36) NOT NULL,
	                 store_uuid      VARCHAR(36) NOT NULL,
	                 schedule_uuid   VARCHAR(36) NOT NULL,
	                 retention_uuid  VARCHAR(36) NOT NULL,
	                 priority        INTEGER DEFAULT 50,
	                 paused          BOOLEAN,
	                 name            TEXT,
					 summary         TEXT
	               )`)
	if err != nil {
		return err
	}

	err = db.Exec(`CREATE TABLE archives (
	                 uuid         VARCHAR(36) PRIMARY KEY,
	                 target_uuid  VARCHAR(36) NOT NULL,
	                 store_uuid   VARCHAR(36) NOT NULL,
	                 store_key    TEXT NOT NULL,

	                 taken_at     INTEGER NOT NULL,
	                 expires_at   INTEGER NOT NULL,
	                 notes        TEXT DEFAULT ('')
	               )`)
	if err != nil {
		return err
	}

	err = db.Exec(`CREATE TABLE tasks (
	                 uuid      VARCHAR(36) PRIMARY KEY,
	                 owner     TEXT,
	                 op        TEXT NOT NULL,

	                 job_uuid      VARCHAR(36),
	                 archive_uuid  VARCHAR(36),
	                 target_uuid   VARCHAR(36),

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
