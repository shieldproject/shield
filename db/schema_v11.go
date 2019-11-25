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
                            j.next_run, j.priority, IFNULL(j.paused, 0), j.fixed_key, 0
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

	jobs, err := db.GetAllJobs(&JobFilter{})
	if err != nil {
		return nil
	}
	for _, job := range jobs {
		healthy := job.LastTaskStatus == "done"
		db.UpdateJobHealth(job.UUID, healthy)
	}

	err = db.Exec(`CREATE TABLE targets_new (
                    uuid               UUID PRIMARY KEY,
                    tenant_uuid        UUID NOT NULL,
                    name               TEXT NOT NULL,
                    summary            TEXT NOT NULL,
                    plugin             TEXT NOT NULL,
                    endpoint           TEXT NOT NULL,
                    agent              TEXT NOT NULL,
                    compression        TEXT NOT NULL DEFAULT 'none',
                    healthy            BOOLEAN NOT NULL DEFAULT 0
                )`)
	if err != nil {
		return err
	}

	err = db.Exec(`INSERT INTO targets_new (uuid, tenant_uuid, name, summary,
                                            plugin, endpoint, agent, compression,
                                            healthy)
                        SELECT t.uuid, t.tenant_uuid, t.name, t.summary,
                                    t.plugin, t.endpoint, t.agent, t.compression,
                                    0
                            FROM targets t`)
	if err != nil {
		return err
	}
	err = db.Exec(`DROP TABLE targets`)
	if err != nil {
		return err
	}
	err = db.Exec(`ALTER TABLE targets_new RENAME TO targets`)
	if err != nil {
		return err
	}

	targets, err := db.GetAllTargets(&TargetFilter{})
	if err != nil {
		return nil
	}
	for _, target := range targets {
		jobs, err = db.GetAllJobs(&JobFilter{ForTarget: target.UUID})
		if err != nil {
			return nil
		}
		healthy := true
		for _, job := range jobs {
			if !job.Healthy {
				healthy = false
			}
		}
		db.UpdateTargetHealth(target.UUID, healthy)
	}

	err = db.Exec(`UPDATE schema_info set version = 11`)
	if err != nil {
		return err
	}

	return nil
}
