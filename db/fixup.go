package db

import (
	"fmt"
	"time"
)

type fixup struct {
	id      string
	name    string
	summary string
	created time.Time
	fn      func(*DB) error
}

var fixups []fixup

func init() {
	fixups = append(fixups, fixup{
		id:      "purge-task-516",
		name:    "Re-schedule Failed Purge Tasks",
		created: time.Date(2019, 05, 14, 15, 54, 00, 0, time.UTC),
		summary: `
There was an issue in versions of SHIELD between
8.1.0 and 8.2.0, where purge tasks would fail, but
still mark the archive as having been removed from
cloud storage.  This caused SHIELD to never retry
the purge operation, leading to ever-increasing
usage of cloud storage.

This fixup resets the "purged" status on archives
that do **not** have a successful purge operation
task attached to them.

See [issue #516](https://github.com/starkandwayne/shield/issues/516) in GitHub for details.`,
		fn: func(db *DB) error {
			/* look through all archives with status = 'purged',
			   and then cross-ref with all tasks, looking for those
			   without any op=purge/status=done attached.

			   set those archives to 'expired' */

			return db.Exec(`
				UPDATE archives
				   SET status = 'expired'
				 WHERE status = 'purged'
				   AND uuid NOT IN (SELECT archive_uuid
				                      FROM tasks
				                     WHERE op     = 'purge'
				                       AND status = 'done')`)
		},
	})
}

func (db *DB) RegisterFixups() error {
	for _, f := range fixups {
		n, err := db.Count(`SELECT id FROM fixups WHERE id = ?`, f.id)
		if err != nil {
			return fmt.Errorf("unable to register fixup '%s': %s", f.id, err)
		}
		if n == 0 {
			err = db.Exec(
				`INSERT INTO fixups
				    (id, name, summary, created_at, applied_at)
				  VALUES
				    (?, ?, ?, ?, NULL)`,
				f.id, f.name, f.summary, f.created)
			if err != nil {
				return fmt.Errorf("unable to register fixup '%s': %s", f.id, err)
			}
		}
	}
	return nil
}

func (db *DB) ApplyFixup(id string) error {
	for _, f := range fixups {
		if f.id == id {
			err := f.fn(db)
			if err != nil {
				return fmt.Errorf("unable to apply fixup '%s': %s", f.id, err)
			}

			err = db.Exec(`UPDATE fixups SET applied_at = ? WHERE id = ?`,
				time.Now().Unix(), f.id)
			if err != nil {
				return fmt.Errorf("unable to track application of fixup '%s' in database: %s", f.id, err)
			}
			return nil
		}
	}
	return fmt.Errorf("unrecognized fixup '%s' attempted", id)
}

func (db *DB) ApplyFixups() error {
	for _, f := range fixups {
		if err := db.ApplyFixup(f.id); err != nil {
			return err
		}
	}
	return nil
}
