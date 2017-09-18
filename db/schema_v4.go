package db

import (
	"github.com/pborman/uuid"
)

type v4Schema struct{}

func (s v4Schema) Deploy(db *DB) error {
	var err error

	// Set up Multi-Tenancy
	switch db.Driver {
	case "mysql":
		err = db.Exec(`CREATE TABLE tenants (
		                 uuid          VARCHAR(36) NOT NULL,
		                 name          TEXT NOT NULL DEFAULT '',

		                 PRIMARY KEY (uuid)
		               )`)

	case "postgres", "sqlite3":
		err = db.Exec(`CREATE TABLE tenants (
		                 uuid          UUID PRIMARY KEY,
		                 name          TEXT NOT NULL DEFAULT ''
		               )`)
	}
	if err != nil {
		return err
	}

	// FIXME: backfill tenant UUIDs everywhere
	tenant := uuid.NewRandom()
	err = db.Exec(`INSERT INTO tenants (uuid, name) VALUES (?, ?)`, tenant.String(), "tenant1")
	if err != nil {
		return err
	}

	switch db.Driver {
	case "mysql":
		err = db.Exec(`ALTER TABLE jobs ADD COLUMN tenant_uuid VARCHAR(36)`)
	case "postgres", "sqlite3":
		err = db.Exec(`ALTER TABLE jobs ADD COLUMN tenant_uuid UUID`)
	}
	if err != nil {
		return err
	}
	err = db.Exec(`UPDATE jobs SET tenant_uuid = ?`, tenant.String())
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
	if db.Driver == "sqlite3" {
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
	} else {
		err = db.Exec(`ALTER TABLE jobs DROP COLUMN schedule_uuid`)
		if err != nil {
			return err
		}
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
	// FIXME - need to backfill archives.job based on heuristics

	switch db.Driver {
	case "mysql":
		err = db.Exec(`CREATE TABLE agents (
		                 uuid          VARCHAR(36) NOT NULL,
		                 name          TEXT NOT NULL DEFAULT '',
		                 address       TEXT NOT NULL DEFAULT '',
		                 version       TEXT NOT NULL DEFAULT '',
		                 hidden        BOOLEAN,
		                 last_seen_at  INTEGER NOT NULL,
		                 last_error    TEXT NOT NULL DEFAULT '',
		                 status        TEXT NOT NULL,
		                 metadata      TEXT NOT NULL DEFAULT '',

		                 PRIMARY KEY (uuid)
		               )`)

	case "postgres", "sqlite3":
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
	}
	if err != nil {
		return err
	}

	//backfill tenant uuids into all of the relevant tables
	//TODO prepare transaction so you can roll back if shit goes wrong
	switch db.Driver {
	case "mysql":
		err = db.Exec("ALTER TABLE stores ADD tenant_uuid VARCHAR(36) NOT NULL DEFAULT 'default'")
		if err != nil {
			return err
		}
		err = db.Exec("ALTER TABLE retention ADD tenant_uuid VARCHAR(36) NOT NULL DEFAULT 'default'")
		if err != nil {
			return err
		}
		err = db.Exec("ALTER TABLE archives ADD tenant_uuid VARCHAR(36) NOT NULL DEFAULT 'default'")
		if err != nil {
			return err
		}
		err = db.Exec("ALTER TABLE tasks ADD tenant_uuid VARCHAR(36) NOT NULL DEFAULT 'default'")
		if err != nil {
			return err
		}
		err = db.Exec("ALTER TABLE targets ADD tenant_uuid VARCHAR(36) NOT NULL DEFAULT 'default'")
		if err != nil {
			return err
		}
	case "postgres", "sqlite3":
		err = db.Exec("ALTER TABLE stores ADD tenant_uuid UUID NOT NULL DEFAULT 'default'")
		if err != nil {
			return err
		}
		err = db.Exec("ALTER TABLE retention ADD tenant_uuid UUID NOT NULL DEFAULT 'default'")
		if err != nil {
			return err
		}
		err = db.Exec("ALTER TABLE archives ADD tenant_uuid UUID NOT NULL DEFAULT 'default'")
		if err != nil {
			return err
		}
		err = db.Exec("ALTER TABLE tasks ADD tenant_uuid UUID NOT NULL DEFAULT 'default'")
		if err != nil {
			return err
		}
		err = db.Exec("ALTER TABLE targets ADD tenant_uuid UUID NOT NULL DEFAULT 'default'")
		if err != nil {
			return err
		}
	}

	switch db.Driver {
	case "mysql":
		err = db.Exec(`CREATE TABLE users (
		                 uuid          VARCHAR(36) NOT NULL,
		                 name          TEXT NOT NULL DEFAULT '',
		                 account       TEXT NOT NULL DEFAULT '',
		                 backend       VARCHAR(100) NOT NULL,
		                 pwhash        TEXT, -- only for local accounts
		                 sysrole       VARCHAR(100) NOT NULL DEFAULT '',

		                 UNIQUE KEY (account, backend),
		                 PRIMARY KEY (uuid)
		               )`)

	case "postgres", "sqlite3":
		err = db.Exec(`CREATE TABLE users (
		                 uuid          UUID PRIMARY KEY,
		                 name          TEXT NOT NULL DEFAULT '',
		                 account       TEXT NOT NULL DEFAULT '',
		                 backend       VARCHAR(100) NOT NULL,
		                 pwhash        TEXT, -- only for local accounts
		                 sysrole       VARCHAR(100) NOT NULL DEFAULT '',

		                 UNIQUE (account, backend)
		               )`)
	}
	if err != nil {
		return err
	}

	switch db.Driver {
	case "mysql":
		err = db.Exec(`CREATE TABLE memberships (
		                 user_uuid     VARCHAR(36) NOT NULL,
		                 tenant_uuid   VARCHAR(36) NOT NULL,
		                 role          VARCHAR(100) NOT NULL,
		                 PRIMARY KEY (user_uuid, tenant_uuid)
		               )`)

	case "postgres", "sqlite3":
		err = db.Exec(`CREATE TABLE memberships (
		                 user_uuid     UUID NOT NULL,
		                 tenant_uuid   UUID NOT NULL,
		                 role          VARCHAR(100) NOT NULL,
		                 PRIMARY KEY (user_uuid, tenant_uuid)
		               )`)
	}
	if err != nil {
		return err
	}

	switch db.Driver {
	case "mysql":
		err = db.Exec(`CREATE TABLE sessions (
		                 uuid          VARCHAR(36) NOT NULL,
		                 user_uuid     VARCHAR(36) NOT NULL,

		                 PRIMARY KEY (uuid)
		               )`)

	case "postgres", "sqlite3":
		err = db.Exec(`CREATE TABLE sessions (
		                 uuid          UUID PRIMARY KEY,
		                 user_uuid     UUID NOT NULL
		               )`)
	}
	if err != nil {
		return err
	}

	switch db.Driver {
	case "mysql":
		err = db.Exec(`CREATE TABLE org_team_tenant_role (
		                 org         VARCHAR(36) NOT NULL,
						 team        VARCHAR(36) NOT NULL,
						 tenant_uuid VARCHAR(36) NOT NULL,
						 role        VARCHAR(36) NOT NULL,

						 PRIMARY KEY (org, team)
		               )`)

	case "postgres", "sqlite3":
		err = db.Exec(`CREATE TABLE org_team_tenant_role (
						org         VARCHAR(36) NOT NULL,
						team        VARCHAR(36) NOT NULL,
						tenant_uuid UUID NOT NULL,
						role        VARCHAR(36) NOT NULL,

						PRIMARY KEY (org, team)						
		               )`)
	}
	if err != nil {
		return err
	}

	switch db.Driver {
	case "mysql":
		err = db.Exec(`CREATE TABLE shield_roles (
						 role_id    INT NOT NULL AUTO_INCREMENT,
						 name	 	VARCHAR(36) NOT NULL,
						 right   	VARCHAR(36) NOT NULL,

		                 PRIMARY KEY (role_id)
		               )`)

	case "postgres", "sqlite3":
		err = db.Exec(`CREATE TABLE shield_roles (
						 role_id   	INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
						 name		VARCHAR(36) NOT NULL,
						 right 		VARCHAR(36) NOT NULL
		               )`)
	}
	if err != nil {
		return err
	}

	err = db.Exec(`UPDATE schema_info set version = 4`)
	if err != nil {
		return err
	}

	return nil
}
