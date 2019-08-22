package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/jhunt/go-log"

	"github.com/shieldproject/shield/timespec"
)

type Fixup struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Summary   string `json:"summary"`
	CreatedAt int64  `json:"created_at"`
	AppliedAt int64  `json:"applied_at"`
	fn        func(*DB) error
}

func (f *Fixup) Apply(db *DB) error {
	err := f.fn(db)
	if err != nil {
		return fmt.Errorf("unable to apply fixup '%s': %s", f.ID, err)
	}

	db.exclusive.Lock()
	err = db.exec(`UPDATE fixups SET applied_at = ? WHERE id = ?`, time.Now().Unix(), f.ID)
	db.exclusive.Unlock()
	if err != nil {
		return fmt.Errorf("unable to track application of fixup '%s' in database: %s", f.ID, err)
	}

	return nil
}

type FixupFilter struct {
	ID          string
	SkipApplied bool
}

func (f *FixupFilter) Query() (string, []interface{}) {
	wheres := []string{"f.id = f.id"}
	var args []interface{}

	if f.SkipApplied {
		wheres = append(wheres, "f.applied_at IS NOT NULL")
	}

	if f.ID != "" {
		wheres = append(wheres, "f.id = ?")
		args = append(args, f.ID)
	}

	return `
		SELECT f.id, f.name, f.summary, f.created_at, f.updated_at
		 WHERE ` + strings.Join(wheres, " AND ") + `
		 ORDER BY created_at ASC`, args
}

var fixups map[string]*Fixup

func init() {
	fixups = make(map[string]*Fixup)

	fixups["purge-task-516"] = &Fixup{
		Name:      "Re-schedule Failed Purge Tasks",
		CreatedAt: time.Date(2019, 05, 14, 15, 54, 00, 0, time.UTC).Unix(),
		Summary: `
There was an issue in versions of SHIELD between
8.1.0 and 8.2.0, where purge tasks would fail, but
still mark the archive as having been removed from
cloud storage.  This caused SHIELD to never retry
the purge operation, leading to ever-increasing
usage of cloud storage.

This fixup resets the "purged" status on archives
that do **not** have a successful purge operation
task attached to them.

See [issue #516](https://github.com/shieldproject/shield/issues/516) in GitHub for details.`,
		fn: func(db *DB) error {
			/* look through all archives with status = 'purged',
			   and then cross-ref with all tasks, looking for those
			   without any op=purge/status=done attached.

			   set those archives to 'expired' */

			db.exclusive.Lock()
			defer db.exclusive.Unlock()
			return db.exec(`
				UPDATE archives
				   SET status = 'expired'
				 WHERE status = 'purged'
				   AND uuid NOT IN (SELECT archive_uuid
				                      FROM tasks
				                     WHERE op     = 'purge'
				                       AND status = 'done')`)
		},
	}

	fixups["agent-status-task-tenant-uuid-522"] = &Fixup{
		Name:      "Associate Orphaned agent-status Tasks with Global Tenant",
		CreatedAt: time.Date(2019, 05, 15, 12, 32, 00, 0, time.UTC).Unix(),
		Summary: `
There was an issue in versions of SHIELD prior to
8.2.0, where <code>agent-status</code> tasks were
inserted into the database with an empty tenant UUID.

This renders them inaccessible to the
<code>/v2/tasks/:uuid</code> endpoint, which drives
the <code>shield task $uuid</code> command.

This fixup re-associates all agent-status tasks with
the global tenant UUID, to fix that.

See [issue #522](https://github.com/shieldproject/shield/issues/522) in GitHub for details.`,
		fn: func(db *DB) error {
			db.exclusive.Lock()
			defer db.exclusive.Unlock()
			return db.exec(`
				UPDATE tasks
				   SET tenant_uuid = ?
				 WHERE tenant_uuid = ''
				   AND op          = 'agent-status'`, GlobalTenantUUID)
		},
	}

	fixups["keep-n-460"] = &Fixup{
		Name:      "Job Keep-N=0 Fix",
		CreatedAt: time.Date(2019, 05, 15, 15, 43, 00, 0, time.UTC).Unix(),
		Summary: `
The holistic <code>/v2/tenants/systems/...</code> API
handlers were incorrectly skipping the calculation and
population of the number of kept backups, leading to
front-end display of jobs created via the Web UI (only)
that claim to be keeping "0 backups / X days".

This has been fixed as of 8.2.0, and this data fixup
re-calculates the "keep-n" attribute of all jobs that
were affected.

See [issue #460](https://github.com/shieldproject/shield/issues/460) in GitHub for details.`,
		fn: func(db *DB) error {
			db.exclusive.Lock()
			defer db.exclusive.Unlock()
			r, err := db.query(`
				SELECT uuid
				  FROM jobs
				 WHERE keep_n = 0`)
			if err != nil {
				return fmt.Errorf("unable to retrieve jobs with keep_n == 0: %s", err)
			}

			var uuids []string
			for r.Next() {
				var uuid string
				if err = r.Scan(&uuid); err != nil {
					r.Close()
					return fmt.Errorf("unable to retrieve jobs with keep_n == 0: %s", err)
				}

				uuids = append(uuids, uuid)
			}
			r.Close()

			for _, uuid := range uuids {
				job, err := db.doGetJob(uuid)
				if err != nil {
					return fmt.Errorf("unable to retrieve job '%s': %s", uuid, err)
				}

				sched, err := timespec.Parse(job.Schedule)
				if err != nil {
					log.Errorf("failed to parse job schedule '%s': %s", job.Schedule, err)
					continue
				}
				job.KeepN = sched.KeepN(job.KeepDays)
				err = db.doUpdateJob(job)
				if err != nil {
					log.Errorf("failed to apply UpdateJob fixup: %s", err)
					continue
				}
			}
			return nil
		},
	}

	/* put the IDs back in so that we don't have to do it manually */
	for id, f := range fixups {
		f.ID = id
	}
}

func (db *DB) GetAllFixups(filter *FixupFilter) ([]*Fixup, error) {
	if filter == nil {
		filter = &FixupFilter{}
	}

	l := []*Fixup{}
	query, args := filter.Query()
	r, err := db.query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		f := &Fixup{}

		if err = r.Scan(r, &f.ID, &f.Name, f.Summary, f.CreatedAt, f.AppliedAt); err != nil {
			return l, err
		}

		l = append(l, f)
	}

	return l, nil
}

func (db *DB) GetFixup(id string) (*Fixup, error) {
	r, err := db.GetAllFixups(&FixupFilter{ID: id})
	if err != nil {
		return nil, err
	}
	if len(r) == 0 {
		return nil, nil
	}
	return r[0], nil
}

func (db *DB) ApplyFixups() error {
	db.Exec(`DELETE FROM fixups WHERE id = ""`)

	for _, f := range fixups {
		if existing, err := db.GetFixup(f.ID); err != nil {
			log.Infof("INITIALIZING: creating fixup record for %s...", f.ID)

			err = db.Exec(`
				INSERT INTO fixups (id, name, summary, created_at, applied_at)
				              VALUES (?,  ?,    ?,       ?,          NULL)`,
				f.ID, f.Name, f.Summary, f.CreatedAt)
			if err != nil {
				return fmt.Errorf("unable to register fixup '%s': %s", f.ID, err)
			}
		} else if existing == nil {
			return fmt.Errorf("unable to find fixup '%s' in database: %s", f.ID, err)
		}

		log.Infof("INITIALIZING: applying fixup %s...", f.ID)
		if err := f.Apply(db); err != nil {
			return err
		}
	}

	return nil
}
