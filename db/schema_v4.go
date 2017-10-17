package db

import (
	"fmt"

	"github.com/pborman/uuid"
)

type v4Schema struct{}

func (s v4Schema) Deploy(db *DB) error {
	var err error

	// Set up Multi-Tenancy
	err = db.Exec(`CREATE TABLE tenants (
	                 uuid          UUID PRIMARY KEY,
	                 name          TEXT NOT NULL DEFAULT ''
	               )`)
	if err != nil {
		return err
	}

	err = db.Exec(fmt.Sprintf("ALTER TABLE stores ADD agent TEXT NOT NULL DEFAULT ''"))
	if err != nil {
		return err
	}

	err = db.Exec(fmt.Sprintf("ALTER TABLE stores ADD public_config TEXT NOT NULL DEFAULT '[]'"))
	if err != nil {
		return err
	}

	err = db.Exec(fmt.Sprintf("ALTER TABLE stores ADD private_config TEXT NOT NULL DEFAULT '[]'"))
	if err != nil {
		return err
	}

	tenant := uuid.NewRandom()
	err = db.Exec(`INSERT INTO tenants (uuid, name) VALUES (?, ?)`, tenant.String(), "tenant1")
	if err != nil {
		return err
	}
	err = db.Exec(fmt.Sprintf("ALTER TABLE jobs ADD tenant_uuid UUID NOT NULL DEFAULT '%s'", tenant.String()))
	if err != nil {
		return err
	}
	err = db.Exec(fmt.Sprintf("ALTER TABLE stores ADD tenant_uuid UUID NOT NULL DEFAULT  '%s'", tenant.String()))
	if err != nil {
		return err
	}
	err = db.Exec(fmt.Sprintf("ALTER TABLE retention ADD tenant_uuid UUID NOT NULL DEFAULT  '%s'", tenant.String()))
	if err != nil {
		return err
	}
	err = db.Exec(fmt.Sprintf("ALTER TABLE archives ADD tenant_uuid UUID NOT NULL DEFAULT  '%s'", tenant.String()))
	if err != nil {
		return err
	}
	err = db.Exec(fmt.Sprintf("ALTER TABLE tasks ADD tenant_uuid UUID NOT NULL DEFAULT  '%s'", tenant.String()))
	if err != nil {
		return err
	}
	err = db.Exec(fmt.Sprintf("ALTER TABLE targets ADD tenant_uuid UUID NOT NULL DEFAULT  '%s'", tenant.String()))
	if err != nil {
		return err
	}

	// Add a next_run timestamp to the jobs
	err = db.Exec(`ALTER TABLE jobs ADD COLUMN next_run INTEGER DEFAULT 0`)
	if err != nil {
		return err
	}

	// Move schedule to be a field on jobs
	err = db.Exec(`ALTER TABLE jobs ADD COLUMN schedule TEXT`)
	if err != nil {
		return err
	}
	err = db.Exec(`UPDATE jobs SET schedule =
	                  (SELECT timespec FROM schedules
	                   WHERE schedules.uuid = jobs.schedule_uuid)`)
	if err != nil {
		return err
	}
	// ... and remove the schedule_uuid field
	err = db.Exec(`CREATE TABLE jobs_new (
	               uuid            UUID PRIMARY KEY,
	               target_uuid     UUID NOT NULL,
	               store_uuid      UUID NOT NULL,
	               tenant_uuid     UUID,
	               schedule        TEXT NOT NULL,
	               next_run        INTEGER DEFAULT 0,
	               retention_uuid  UUID NOT NULL,
	               priority        INTEGER DEFAULT 50,
	               paused          BOOLEAN,
	               name            TEXT,
	               summary         TEXT
	             )`)
	if err != nil {
		return err
	}
	err = db.Exec(`INSERT INTO jobs_new (uuid, target_uuid, store_uuid, tenant_uuid,
	                                     schedule, next_run, retention_uuid,
	                                     priority, paused, name, summary)
	                              SELECT uuid, target_uuid, store_uuid, ?,
	                                     schedule, next_run, retention_uuid,
	                                     priority, paused, name, summary
	                                FROM jobs`, tenant.String())
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

	err = db.Exec(`DROP TABLE schedules`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE tasks ADD COLUMN ok INT NOT NULL DEFAULT 1`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE tasks ADD COLUMN notes TEXT NOT NULL DEFAULT ''`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE tasks ADD COLUMN clear TEXT NOT NULL DEFAULT 'normal'`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE tasks ADD COLUMN relevant INT NOT NULL DEFAULT 1`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE archives ADD COLUMN job TEXT NOT NULL DEFAULT ''`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE archives ADD COLUMN encryption_type TEXT NOT NULL DEFAULT ''`)
	if err != nil {
		return err
	}
	// FIXME - need to backfill archives.job based on heuristics

	err = db.Exec(`CREATE TABLE agents (
	                 uuid          UUID PRIMARY KEY,
	                 name          TEXT NOT NULL DEFAULT '',
	                 address       TEXT NOT NULL DEFAULT '',
	                 version       TEXT NOT NULL DEFAULT '',
	                 hidden        BOOLEAN,
	                 last_seen_at  INTEGER NOT NULL,
	                 last_error    TEXT NOT NULL DEFAULT '',
	                 status        TEXT NOT NULL,
	                 metadata      TEXT NOT NULL DEFAULT ''
	               )`)
	if err != nil {
		return err
	}

	err = db.Exec(`CREATE TABLE users (
	                 uuid          UUID PRIMARY KEY,
	                 name          TEXT NOT NULL DEFAULT '',
	                 account       TEXT NOT NULL DEFAULT '',
	                 backend       VARCHAR(100) NOT NULL,
	                 pwhash        TEXT, -- only for local accounts
	                 sysrole       VARCHAR(100) NOT NULL DEFAULT '',

	                 UNIQUE (account, backend)
	               )`)
	if err != nil {
		return err
	}

	err = db.Exec(`CREATE TABLE memberships (
	                 user_uuid     UUID NOT NULL,
	                 tenant_uuid   UUID NOT NULL,
	                 role          VARCHAR(100) NOT NULL,
	                 PRIMARY KEY (user_uuid, tenant_uuid)
	               )`)
	if err != nil {
		return err
	}

	err = db.Exec(`CREATE TABLE sessions (
	                 uuid          UUID PRIMARY KEY,
	                 user_uuid     UUID NOT NULL,
	                 provider      TEXT,
	                 provider_data TEXT
	               )`)
	if err != nil {
		return err
	}

	err = db.Exec(`UPDATE schema_info set version = 4`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE stores ADD COLUMN daily_increase INTEGER DEFAULT NULL`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE stores ADD COLUMN storage_used INTEGER DEFAULT NULL`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE stores ADD COLUMN archive_count INTEGER DEFAULT NULL`)
	if err != nil {
		return err
	}

	return nil
}
