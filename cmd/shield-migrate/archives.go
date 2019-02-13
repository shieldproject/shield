package main

import (
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
		var uuid, target_uuid, store_uuid, store_key, notes, purge_reason, status *string
		var taken_at, expires_at *int64

		if err := rs.Scan(&uuid, &target_uuid, &store_uuid, &store_key, &taken_at, &expires_at, &notes, &purge_reason, &status); err != nil {
			log.Errorf("failed to read result from archives source table: %s", err)
			os.Exit(3)
		}

		err = to.Exec(`
		   INSERT INTO archives (uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, notes, purge_reason, status)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, notes, purge_reason, status)
		if err != nil {
			log.Errorf("failed to insert result into archives dest table: %s", err)
			os.Exit(3)
		}
	}

	log.Infof("migrated %d archives", n)
}
