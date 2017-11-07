package main

import (
	"database/sql"
	"os"

	"github.com/jhunt/go-log"

	"github.com/starkandwayne/shield/db"
)

func migrateTargets(to, from *db.DB) {
	n := 0
	rs, err := from.Query(`
	   SELECT uuid, name, summary, plugin, endpoint, agent
	     FROM targets`)
	if err != nil {
		log.Errorf("failed to migrate targets; SELECT said: %s", err)
		os.Exit(3)
	}

	for rs.Next() {
		n += 1
		var a, b, c, d, e, f sql.RawBytes

		if err := rs.Scan(&a, &b, &c, &d, &e, &f); err != nil {
			log.Errorf("failed to read result from targets source table: %s", err)
			os.Exit(3)
		}

		err = to.Exec(`
		   INSERT INTO targets (uuid, name, summary, plugin, endpoint, agent)
                VALUES (?, ?, ?, ?, ?, ?)`,
			a, b, c, d, e, f)
		if err != nil {
			log.Errorf("failed to insert result into targets dest table: %s", err)
			os.Exit(3)
		}
	}

	log.Infof("migrated %d targets", n)
}
