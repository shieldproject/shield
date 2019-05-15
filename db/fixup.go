package db

import (
	"fmt"
	"time"

	"github.com/starkandwayne/shield/timespec"
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
	}, fixup{
		id:      "agent-status-task-tenant-uuid-522",
		name:    "Associate Orphaned agent-status Tasks with Global Tenant",
		created: time.Date(2019, 05, 15, 12, 32, 00, 0, time.UTC),
		summary: `
There was an issue in versions of SHIELD prior to
8.2.0, where <code>agent-status</code> tasks were
inserted into the database with an empty tenant UUID.

This renders them inaccessible to the
<code>/v2/tasks/:uuid</code> endpoint, which drives
the <code>shield task $uuid</code> command.

This fixup re-associates all agent-status tasks with
the global tenant UUID, to fix that.

See [issue #522](https://github.com/starkandwayne/shield/issues/522) in GitHub for details.`,
		fn: func(db *DB) error {
			return db.Exec(`
				UPDATE tasks
				   SET tenant_uuid = ?
				 WHERE tenant_uuid = ''
				   AND op          = 'agent-status'`, GlobalTenantUUID)
		},
	}, fixup{
		id:      "",
		name:    "",
		created: time.Date(2019, 05, 15, 15, 43, 00, 0, time.UTC),
		summary: `
The holistic <code>/v2/tenants/systems/...</code> API
handlers were incorrectly skipping the calculation and
population of the number of kept backups, leading to
front-end display of jobs created via the Web UI (only)
that claim to be keeping "0 backups / X days".

This has been fixed as of 8.2.0, and this data fixup
re-calculates the "keep-n" attribute of all jobs that
were affected.

See [issue #460](https://github.com/starkandwayne/shield/issues/460) in GitHub for details.`,
		fn: func(db *DB) error {
			r, err := db.Query(`
				SELECT uuid
				  FROM jobs
				 WHERE keep_n = 0`)
			if err != nil {
				return fmt.Errorf("unable to retrieve jobs with keep_n == 0: %s", err)
			}
			defer r.Close()

			for r.Next() {
				var uuid string
				if err = r.Scan(&uuid); err != nil {
					return fmt.Errorf("unable to retrieve jobs with keep_n == 0: %s", err)
				}

				job, err := db.GetJob(uuid)
				if err != nil {
					return fmt.Errorf("unable to retrieve job '%s': %s", uuid, err)
				}

				sched, err := timespec.Parse(job.Schedule)
				if err != nil {
					/* FIXME log stuff! */
					continue
				}
				job.KeepN = sched.KeepN(job.KeepDays)
				err = db.UpdateJob(job)
				if err != nil {
					/* FIXME log stuff! */
					continue
				}
			}

			return nil
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
