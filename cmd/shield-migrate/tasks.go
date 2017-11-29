package main

import (
	"database/sql"
	"os"

	"github.com/jhunt/go-log"

	"github.com/starkandwayne/shield/db"
)

func migrateTasks(to, from *db.DB) {
	n := 0
	rs, err := from.Query(`
	   SELECT uuid, owner, op, job_uuid, archive_uuid, target_uuid, status, requested_at, started_at, stopped_at, log, store_uuid, target_plugin, target_endpoint, store_plugin, store_endpoint, restore_key, timeout_at, attempts, agent
	     FROM tasks`)
	if err != nil {
		log.Errorf("failed to migrate tasks; SELECT said: %s", err)
		os.Exit(3)
	}

	for rs.Next() {
		n += 1
		var a, b, c, d, e, f, g, h, i, j, k, l, m, n, o, p, q, r, s, t sql.RawBytes

		if err := rs.Scan(&a, &b, &c, &d, &e, &f, &g, &h, &i, &j, &k, &l, &m, &n, &o, &p, &q, &r, &s, &t); err != nil {
			log.Errorf("failed to read result from tasks source table: %s", err)
			os.Exit(3)
		}

		err = to.Exec(`
		   INSERT INTO tasks (uuid, owner, op, job_uuid, archive_uuid, target_uuid, status, requested_at, started_at, stopped_at, log, store_uuid, target_plugin, target_endpoint, store_plugin, store_endpoint, restore_key, timeout_at, attempts, agent)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			a, b, c, d, e, f, g, h, i, j, k, l, m, n, o, p, q, r, s, t)
		if err != nil {
			log.Errorf("failed to insert result into tasks dest table: %s", err)
			os.Exit(3)
		}
	}

	log.Infof("migrated %d tasks", n)
}
