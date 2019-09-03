package db

import "sort"

type v10Schema struct{}

func (s v10Schema) Deploy(db *DB) error {
	var err error

	/* delete all duplicate agents, except for the most recently
	   seen (via last_seen_at) to fix up some database issues
	   so that we can place a UNIQUE constraint on (address)
	*/

	err = db.exclusively(func() error {
		r, err := db.query(`SELECT uuid, address, last_seen_at FROM agents`)
		if err != nil {
			return err
		}
		defer r.Close()

		type migrationAgent struct {
			uuid         string
			address      string
			last_seen_at int64
		}
		agents := []migrationAgent{}
		for r.Next() {
			idx := len(agents)
			agents = append(agents, migrationAgent{})
			err = r.Scan(&agents[idx].uuid, &agents[idx].address, &agents[idx].last_seen_at)
			if err != nil {
				return err
			}
		}

		err = r.Close()
		if err != nil {
			return err
		}

		sort.Slice(agents, func(i, j int) bool {
			if agents[i].address < agents[j].address {
				return true
			}

			if agents[i].last_seen_at < agents[j].last_seen_at {
				return true
			}

			return agents[i].uuid < agents[j].uuid
		})

		lastAgentAddress := ""
		for i := len(agents) - 1; i >= 0; i-- {
			if agents[i].address != lastAgentAddress {
				lastAgentAddress = agents[i].address
				continue
			}

			err = db.exec(`DELETE FROM agents WHERE uuid = ?`, agents[i].uuid)
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
