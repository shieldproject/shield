package db

import (
	"fmt"
	"supervisor"
	"timespec"

	"github.com/pborman/uuid"
)

func (db *DB) Setup() error {
	v, err := db.schemaVersion()
	if err != nil {
		return err
	}

	if v == 0 {
		err = db.v1schema()
	} else {
		err = fmt.Errorf("Schema version %d is newer than this version of SHIELD", v)
	}

	if err != nil {
		return err
	}
	return nil
}

func (db *DB) GetAllJobs() ([]*supervisor.Job, error) {
	l := []*supervisor.Job{}
	result, err := db.Query(`
		SELECT j.uuid, j.paused,
		       t.plugin, t.endpoint,
		       s.plugin, s.endpoint,
		       sc.timespec, r.expiry
		FROM jobs j
			INNER JOIN targets   t    ON  t.uuid = j.target_uuid
			INNER JOIN stores    s    ON  s.uuid = j.store_uuid
			INNER JOIN schedules sc   ON sc.uuid = j.schedule_uuid
			INNER JOIN retention r    ON  r.uuid = j.retention_uuid
	`)
	if err != nil {
		return l, err
	}
	for result.Next() {
		j := &supervisor.Job{Target: &supervisor.PluginConfig{}, Store: &supervisor.PluginConfig{}}

		var id, tspec string
		var expiry int
		//var paused bool
		err = result.Scan(&id, &j.Paused,
			&j.Target.Plugin, &j.Target.Endpoint,
			&j.Store.Plugin, &j.Store.Endpoint,
			&tspec, &expiry)
		// FIXME: handle err
		j.UUID = uuid.Parse(id)
		j.Spec, err = timespec.Parse(tspec)
		// FIXME: handle err
		l = append(l, j)
	}
	return l, nil
}

// func (db *DB) AnnotateArchive(id uuid.UUID, notes string) error
// func (db *DB) AnnotateJob(id uuid.UUID, name string, summary string) error
// func (db *DB) AnnotateTask(id uuid.UUID, owner string) error

// func (db *DB) CreateJob(target uuid.UUID, store uuid.UUID, schedule uuid.UUID, retention uuid.UUID) (uuid.UUID, error)
// func (db *DB) PauseJob(id uuid.UUID) error
// func (db *DB) UnpauseJob(id uuid.UUID) error
// func (db *DB) DeleteJob(id uuid.UUID) error

// func (db *DB) CreateArchive(job uuid.UUID, key string) (id uuid.UUID, error)
// func (db *DB) DeleteArchive(id uuid.UUID) error

// func (db *DB) CreateTask(op string, args string, job uuid.UUID) (uuid.UUID, error)
// func (db *DB) CompleteTask(id uuid.UUID) error
// func (db *DB) CancelTask(id uuid.UUID) error
// func (db *DB) UpdateTaskLog(id uuid.UUID, log string) error
