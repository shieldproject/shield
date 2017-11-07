package main

import (
	"database/sql"
	"os"

	"github.com/jhunt/go-log"

	"github.com/starkandwayne/shield/db"
)

func migrateStores(to, from *db.DB) {
	n := 0
	rs, err := from.Query(`
	   SELECT uuid, name, summary, plugin, endpoint
	     FROM stores`)
	if err != nil {
		log.Errorf("failed to migrate stores; SELECT said: %s", err)
		os.Exit(3)
	}

	for rs.Next() {
		n += 1
		var a, b, c, d, e sql.RawBytes

		if err := rs.Scan(&a, &b, &c, &d, &e); err != nil {
			log.Errorf("failed to read result from stores source table: %s", err)
			os.Exit(3)
		}

		err = to.Exec(`
		   INSERT INTO stores (uuid, name, summary, plugin, endpoint)
                VALUES (?, ?, ?, ?, ?)`,
			a, b, c, d, e)
		if err != nil {
			log.Errorf("failed to insert result into stores dest table: %s", err)
			os.Exit(3)
		}
	}

	log.Infof("migrated %d stores", n)
}
