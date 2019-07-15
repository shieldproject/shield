package db

type v9Schema struct{}

func (s v9Schema) Deploy(db *DB) error {
	var err error

	// track last_checked_at for each agent, alongside last_seen_at,
	// to ensure that operrators can differentiate between agent->core
	// pings (last_seen_at) and core->agent pings (last_checked_at)
	err = db.Exec(`ALTER TABLE agents ADD COLUMN last_checked_at INTEGER DEFAULT NULL`)
	if err != nil {
		return err
	}

	err = db.Exec(`UPDATE agents SET last_checked_at = last_seen_at`)
	if err != nil {
		return err
	}

	err = db.Exec(`UPDATE schema_info set version = 9`)
	if err != nil {
		return err
	}

	return nil
}
