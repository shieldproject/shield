package main

import (
	"database/sql"
	"os"

	"github.com/jhunt/go-log"

	"github.com/starkandwayne/shield/db"
)

func migrateArchives(to, from *db.DB) {
	n := 0
	rs, err := from.Query(`
	   SELECT uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, notes, purge_reason, status
	     FROM archives`)
	if err != nil {
		log.Errorf("failed to migrate archives; SELECT said: %s", err)
		os.Exit(3)
	}

	for rs.Next() {
		n += 1
		var a, b, c, d, e, f, g, h, i sql.RawBytes

		if err := rs.Scan(&a, &b, &c, &d, &e, &f, &g, &h, &i); err != nil {
			log.Errorf("failed to read result from archives source table: %s", err)
			os.Exit(3)
		}

		err = to.Exec(`
		   INSERT INTO archives (uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, notes, purge_reason, status)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			a, b, c, d, e, f, g, h, i)
		if err != nil {
			log.Errorf("failed to insert result into archives dest table: %s", err)
			os.Exit(3)
		}
	}

	log.Infof("migrated %d archives", n)
}
