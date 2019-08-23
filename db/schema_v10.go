package db

type v10Schema struct{}

func (s v10Schema) Deploy(db *DB) error {
	var err error
	db.exclusive.Lock()
	defer db.exclusive.Unlock()

	/* delete all duplicate agents, except for the most recently
	   seen (via last_seen_at) to fix up some database issues
	   so that we can place a UNIQUE constraint on (address)
	 */
	err = db.exec(`
		DELETE FROM agents
		      WHERE last_seen_at != (SELECT MAX(last_seen_at)
		                               FROM agents a
		                              WHERE a.address = agents.address)`)
	if err != nil {
		return err
	}

	/* place a uniqueness constraint on agent addresses, since they
	   (logically) should be unique.  active fabric makes this case
	   stronger, since address does not necessarily map to  network
	   endpoint, and the same agent process will be able to register
	   multiple (tenant-scoped) virtual agents.

	   a UNIQUE INDEX is how SQLite actually implements its
	   uniqueness constraints.

	   (see https://www.sqlite.org/lang_createtable.html#constraints)
	 */
	err = db.exec(`CREATE UNIQUE INDEX address ON agents (address)`)
	if err != nil {
		return err
	}

	/* hello, v10! */
	err = db.exec(`UPDATE schema_info set version = 10`)
	if err != nil {
		return err
	}

	return nil
}
