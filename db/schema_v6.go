package db

import (
	"github.com/starkandwayne/shield/timespec"
)

type v6Schema struct{}

func (s v6Schema) Deploy(db *DB) error {
	var err error

	// set the tenant_uuid column to NOT NULL
	err = db.Exec(`CREATE TABLE jobs_new (
	               uuid               UUID PRIMARY KEY,
	               target_uuid        UUID NOT NULL,
	               store_uuid         UUID NOT NULL,
	               tenant_uuid        UUID NOT NULL,
	               name               TEXT,
	               summary            TEXT,
	               schedule           TEXT NOT NULL,
	               keep_n             INTEGER NOT NULL DEFAULT 0,
	               keep_days          INTEGER NOT NULL DEFAULT 0,
	               next_run           INTEGER DEFAULT 0,
	               priority           INTEGER DEFAULT 50,
	               paused             BOOLEAN,
	               fixed_key          INTEGER DEFAULT 0
	             )`)
	if err != nil {
		return err
	}
	err = db.Exec(`INSERT INTO jobs_new (uuid, target_uuid, store_uuid, tenant_uuid,
	                                     schedule, next_run, keep_days,
	                                     priority, paused, name, summary)
	                              SELECT j.uuid, j.target_uuid, j.store_uuid, j.tenant_uuid,
	                                     j.schedule, j.next_run, r.expiry / 86400,
	                                     j.priority, j.paused, j.name, j.summary
	                                FROM jobs j INNER JOIN retention r
	                                  ON j.retention_uuid = r.uuid`)
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
	err = db.Exec(`DROP TABLE retention`)
	if err != nil {
		return err
	}

	/* fix keep_n on all jobs */
	jobs, err := db.GetAllJobs(nil)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if sched, err := timespec.Parse(job.Schedule); err != nil {
			job.KeepN = sched.KeepN(job.KeepDays)
			db.UpdateJob(job)
		}
	}

	err = db.Exec(`UPDATE schema_info set version = 6`)
	if err != nil {
		return err
	}

	return nil
}
