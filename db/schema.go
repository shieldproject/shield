package db

import (
	"fmt"
)

var CurrentSchema uint = 1

func (db *DB) Setup() error {
	v, err := db.SchemaVersion()
	if err != nil {
		return err
	}

	if v == 0 {
		err = db.v1schema()
	} else if v > 1 {
		err = fmt.Errorf("Schema version %d is newer than this version of SHIELD", v)
	}

	if err != nil {
		return err
	}
	return nil
}

func (db *DB) SchemaVersion() (uint, error) {
	r, err := db.Query(`SELECT version FROM schema_info LIMIT 1`)
	if err != nil {
		if err.Error() == "no such table: schema_info" {
			return 0, nil
		}
		if err.Error() == `pq: relation "schema_info" does not exist` {
			return 0, nil
		}
		return 0, err
	}
	defer r.Close()

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

func (db *DB) CheckCurrentSchema() error {
	v, err := db.SchemaVersion()
	if err != nil {
		return err
	}
	if v != CurrentSchema {
		return fmt.Errorf("wrong schema version (%d, but want to be at %d)", v, CurrentSchema)
	}
	return nil
}

func (db *DB) v1schema() error {
	db.Exec(`CREATE TABLE schema_info (
               version INTEGER
             )`)
	db.Exec(`INSERT INTO schema_info VALUES (?)`, CurrentSchema)

	db.Exec(`CREATE TABLE targets (
               uuid      UUID PRIMARY KEY,
               name      TEXT,
               summary   TEXT,
               plugin    TEXT,
               endpoint  TEXT
             )`)

	db.Exec(`CREATE TABLE stores (
               uuid      UUID PRIMARY KEY,
               name      TEXT,
               summary   TEXT,
               plugin    TEXT,
               endpoint  TEXT
             )`)

	db.Exec(`CREATE TABLE schedules (
               uuid      UUID PRIMARY KEY,
               name      TEXT,
               summary   TEXT,
               timespec  TEXT
             )`)

	db.Exec(`CREATE TABLE retention (
               uuid     UUID PRIMARY KEY,
               name     TEXT,
               summary  TEXT,
               expiry   INTEGER
             )`)

	db.Exec(`CREATE TABLE jobs (
               uuid            UUID PRIMARY KEY,
               target_uuid     UUID,
               store_uuid      UUID,
               schedule_uuid   UUID,
               retention_uuid  UUID,
               paused          BOOLEAN,
               name            TEXT,
               summary         TEXT
             )`)

	db.Exec(`CREATE TABLE archives (
               uuid         UUID PRIMARY KEY,
               target_uuid  UUID,
               store_uuid   UUID,
               store_key    TEXT,

               taken_at     timestamp without time zone,
               expires_at   timestamp without time zone,
               notes        TEXT DEFAULT ""
             )`)

	db.Exec(`CREATE TABLE tasks (
               uuid      UUID PRIMARY KEY,
               owner     TEXT,
               op        TEXT,

               job_uuid      UUID,
               archive_uuid  UUID,
               target_uuid   UUID,

               status      TEXT,
               started_at  timestamp without time zone,
               stopped_at  timestamp without time zone,

               log       TEXT,
               debug     TEXT
             )`)

	return nil
}
