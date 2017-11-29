package main

import (
	"database/sql"
	"os"

	"github.com/jhunt/go-log"

	"github.com/starkandwayne/shield/db"
)

func migrateJobs(to, from *db.DB) {
	n := 0
	rs, err := from.Query(`
	   SELECT uuid, target_uuid, store_uuid, schedule_uuid, retention_uuid, priority, paused, name, summary
	     FROM jobs`)
	if err != nil {
		log.Errorf("failed to migrate jobs; SELECT said: %s", err)
		os.Exit(3)
	}

	for rs.Next() {
		n += 1
		var a, b, c, d, e, f, g, h, i sql.RawBytes

		if err := rs.Scan(&a, &b, &c, &d, &e, &f, &g, &h, &i); err != nil {
			log.Errorf("failed to read result from jobs source table: %s", err)
			os.Exit(3)
		}

		err = to.Exec(`
		   INSERT INTO jobs (uuid, target_uuid, store_uuid, schedule_uuid, retention_uuid, priority, paused, name, summary)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			a, b, c, d, e, f, g, h, i)
		if err != nil {
			log.Errorf("failed to insert result into jobs dest table: %s", err)
			os.Exit(3)
		}
	}

	log.Infof("migrated %d jobs", n)
}
