package db

type v10Schema struct{}

func (s v10Schema) Deploy(db *DB) error {
	var err error

	/* delete all duplicate agents, except for the most recently
	   seen (via last_seen_at) to fix up some database issues
	   so that we can place a UNIQUE constraint on (address)
	*/

	err = db.exclusively(func() error {
		r, err := db.query(`
		  SELECT uuid, address
		  FROM agents 
		  ORDER BY address ASC, last_seen_at DESC, uuid ASC
		`)
		if err != nil {
			return err
		}
		defer r.Close()

		var lastAgentAddress string
		toDelete := []string{}
		for r.Next() {
			var uuid, address string
			err = r.Scan(&uuid, &address)
			if err != nil {
				return err
			}

			if address != lastAgentAddress {
				lastAgentAddress = address
				continue
			}

			toDelete = append(toDelete, uuid)
		}

		err = r.Close()
		if err != nil {
			return err
		}

		for _, uuid := range toDelete {
			err = db.Exec(`DELETE FROM agents WHERE uuid = ?`, uuid)
			if err != nil {
				return err
			}
		}

		return nil
	})

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
	err = db.Exec(`CREATE UNIQUE INDEX address ON agents (address)`)
	if err != nil {
		return err
	}

	/* hello, v10! */
	err = db.Exec(`UPDATE schema_info set version = 10`)
	if err != nil {
		return err
	}

	return nil
}
