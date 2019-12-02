package db

type v12Schema struct{}

func (s v12Schema) Deploy(db *DB) error {
	var err error

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
                    tenant_uuid        UUID NOT NULL DEFAULT '%s',
                    name               TEXT NOT NULL,
                    summary            TEXT NOT NULL DEFAULT '',
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

	err = db.Exec(`UPDATE schema_info set version = 12`)
	if err != nil {
		return err
	}

	return nil
}
