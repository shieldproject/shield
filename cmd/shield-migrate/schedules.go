package main

import (
	"database/sql"
	"os"

	"github.com/jhunt/go-log"

	"github.com/starkandwayne/shield/db"
)

func migrateSchedules(to, from *db.DB) {
	n := 0
	rs, err := from.Query(`
	   SELECT uuid, name, summary, timespec
	     FROM schedules`)
	if err != nil {
		log.Errorf("failed to migrate schedules; SELECT said: %s", err)
		os.Exit(3)
	}

	for rs.Next() {
		n += 1
		var a, b, c, d sql.RawBytes

		if err := rs.Scan(&a, &b, &c, &d); err != nil {
			log.Errorf("failed to read result from schedules source table: %s", err)
			os.Exit(3)
		}

		err = to.Exec(`
		   INSERT INTO schedules (uuid, name, summary, timespec)
                VALUES (?, ?, ?, ?)`,
			a, b, c, d)
		if err != nil {
			log.Errorf("failed to insert result into schedules dest table: %s", err)
			os.Exit(3)
		}
	}

	log.Infof("migrated %d schedules", n)
}
