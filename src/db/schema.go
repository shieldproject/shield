package db

import (
	"fmt"
)

func (o *ORM) schemaVersion() (uint, error) {
	r, err := o.db.Query(`SELECT version FROM schema_info LIMIT 1`)
	// failed query = no schema
	// FIXME: better error object introspection?
	if err != nil {
		return 0, nil
	}

	// no records = no schema
	if !r.Next() {
		return 0, nil
	}

	var v int
	err = r.Scan(&v)
	// failed unmarshall is an actual error
	if err != nil {
		return 0, err
	}

	// invalid (negative) schema version is an actual error
	if v < 0 {
		return 0, fmt.Errorf("Invalid schema version %d found", v)
	}

	return uint(v), nil
}

func (o *ORM) v1schema() error {
	o.db.Exec(`CREATE TABLE schema_info (
                           version INTEGER
               )`)
	o.db.Exec(`INSERT INTO schema_info VALUES (1)`)

	o.db.Exec(`CREATE TABLE targets (
                 uuid      UUID PRIMARY KEY,
                 name      TEXT,
                 summary   TEXT,
                 plugin    TEXT,
                 endpoint  TEXT
               )`)

	o.db.Exec(`CREATE TABLE stores (
                 uuid      UUID PRIMARY KEY,
                 name      TEXT,
                 summary   TEXT,
                 plugin    TEXT,
                 endpoint  TEXT
               )`)

	o.db.Exec(`CREATE TABLE schedules (
                 uuid      UUID PRIMARY KEY,
                 name      TEXT,
                 summary   TEXT,
                 timespec  TEXT
               )`)

	o.db.Exec(`CREATE TABLE retention (
                 uuid     UUID PRIMARY KEY,
                 name     TEXT,
                 summary  TEXT,
                 expiry   INTEGER
               )`)

	o.db.Exec(`CREATE TABLE jobs (
                 uuid            UUID PRIMARY KEY,
                 target_uuid     UUID,
                 store_uuid      UUID,
                 schedule_uuid   UUID,
                 retention_uuid  UUID,
                 paused          BOOLEAN,
                 name            TEXT,
                 summary         TEXT
               )`)

	o.db.Exec(`CREATE TABLE archives (
                 uuid         UUID PRIMARY KEY,
                 target_uuid  UUID,
                 store_uuid   UUID,
                 store_key    TEXT,

                 taken_at     timestamp without time zone,
                 expires_at   timestamp without time zone,
                 notes        TEXT
               )`)

	o.db.Exec(`CREATE TABLE tasks (
                 uuid      UUID PRIMARY KEY,
                 owner     TEXT,
                 op        TEXT,
                 args      TEXT,

                 job_uuid      UUID,
                 archive_uuid  UUID,

                 status      status,
                 started_at  timestamp without time zone,
                 stopped_at  timestamp without time zone,

                 log       TEXT,
                 debug     TEXT
               )`)

	return nil
}
