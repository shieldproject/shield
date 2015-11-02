package db

import (
	"fmt"
	"supervisor"
	"timespec"

	"github.com/pborman/uuid"
)

type ORM struct {
	db *DB
}

func NewORM(db *DB) (*ORM, error) {
	if db == nil {
		return nil, fmt.Errorf("No database given to NewORM()")
	}
	if !db.Connected() {
		return nil, fmt.Errorf("Not connected to database yet")
	}

	return &ORM{db: db}, nil
}

func (o *ORM) Setup() error {
	v, err := o.schemaVersion()
	if err != nil {
		return err
	}

	if v == 0 {
		err = o.v1schema()
	} else {
		err = fmt.Errorf("Schema version %d is newer than this version of SHIELD", v)
	}

	if err != nil {
		return err
	}

	/* FIXME: cache all queries we're going to need */
	o.db.Cache("GetAllJobs", `SELECT jobs.uuid, jobs.paused,
														targets.plugin, targets.endpoint,
														stores.plugin, stores.endpoint,
														schedules.timespec, retention.expiry
														FROM jobs INNER JOIN targets ON targets.uuid = jobs.target_uuid
														INNER JOIN stores ON stores.uuid = jobs.store_uuid
														INNER JOIN schedules ON schedules.uuid = jobs.schedule_uuid
														INNER JOIN retention ON retention.uuid = jobs.retention_uuid`)

	return nil
}

func (o *ORM) schemaVersion() (uint, error) {
	err := o.db.Cache("schema:version", `SELECT version FROM schema_info LIMIT 1`)
	if err != nil {
		return 0, err
	}

	r, err := o.db.Query("schema:version")
	// failed query = no schema
	// FIXME: better error object introspection?
	if err != nil {
		return 0, nil
	}

	// no records = no schema
	if !r.Next() {
		return 0, nil
	}

	var v int
	err = r.Scan(&v)
	// failed unmarshall is an actual error
	if err != nil {
		return 0, err
	}

	// invalid (negative) schema version is an actual error
	if v < 0 {
		return 0, fmt.Errorf("Invalid schema version %d found", v)
	}

	return uint(v), nil
}

func (o *ORM) v1schema() error {
	o.db.ExecOnce(`CREATE TABLE schema_info (
                              version INTEGER
                  )`)
	o.db.ExecOnce(`INSERT INTO schema_info VALUES (1)`)

	o.db.ExecOnce(`CREATE TABLE targets (
                    uuid      UUID PRIMARY KEY,
                    name      TEXT,
                    summary   TEXT,
                    plugin    TEXT,
                    endpoint  TEXT
                  )`)

	o.db.ExecOnce(`CREATE TABLE stores (
                    uuid      UUID PRIMARY KEY,
                    name      TEXT,
                    summary   TEXT,
                    plugin    TEXT,
                    endpoint  TEXT
                  )`)

	o.db.ExecOnce(`CREATE TABLE schedules (
                    uuid      UUID PRIMARY KEY,
                    name      TEXT,
                    summary   TEXT,
                    timespec  TEXT
                  )`)

	o.db.ExecOnce(`CREATE TABLE retention (
                    uuid     UUID PRIMARY KEY,
                    name     TEXT,
                    summary  TEXT,
                    expiry   INTEGER
                  )`)

	o.db.ExecOnce(`CREATE TABLE jobs (
                    uuid            UUID PRIMARY KEY,
                    target_uuid     UUID,
                    store_uuid      UUID,
                    schedule_uuid   UUID,
                    retention_uuid  UUID,
                    paused          BOOLEAN,
                    name            TEXT,
                    summary         TEXT
                  )`)

	o.db.ExecOnce(`CREATE TABLE archives (
                    uuid         UUID PRIMARY KEY,
                    target_uuid  UUID,
                    store_uuid   UUID,
                    store_key    TEXT,

                    taken_at     timestamp without time zone,
                    expires_at   timestamp without time zone,
                    notes        TEXT
                  )`)

	o.db.ExecOnce(`CREATE TABLE tasks (
                    uuid      UUID PRIMARY KEY,
                    owner     TEXT,
                    op        TEXT,
                    args      TEXT,

                    job_uuid      UUID,
                    archive_uuid  UUID,

                    status      status,
                    started_at  timestamp without time zone,
                    stopped_at  timestamp without time zone,

                    log       TEXT,
                    debug     TEXT
                  )`)

	return nil
}

/* FIXME: determine what ORM layers we need */

// func annotateNameAndSummary(q string, id uuid.UUID, name string, summary string) error
// func AnnotateArchive(id uuid.UUID, notes string) error
// func AnnotateJob(id uuid.UUID, name string, summary string) error
// func AnnotateRetentionPolicy(id uuid.UUID, name string, summary string) error
// func AnnotateSchedule(id uuid.UUID, name string, summary string) error
// func AnnotateStore(id uuid.UUID, name string, summary string) error
// func AnnotateTarget(id uuid.UUID, name string, summary string) error
// func AnnotateTask(id uuid.UUID, owner string) error

// func CreateTarget(plugin string, endpoint interface{}) (uuid.UUID, error)
// func UpdateTarget(id uuid.UUID, plugin string, endpoint interface{}) error
// func DeleteTarget(id uuid.UUID) error

// func CreateStore(plugin string, endpoint interface{}) (uuid.UUID, error)
// func UpdateStore(id uuid.UUID, plugin string, endpoint interface{}) error
// func DeleteStore(id uuid.UUID) error

// func CreateSchedule(timespec string) (uuid.UUID, error)
// func UpdateSchedule(id uuid.UUID, timespec string) error
// func DeleteSchedule(id uuid.UUID)

// func CreateRetentionPolicy(expiry uint) (uuid.UUID, error)
// func UpdateRetentionPolicy(id uuid.UUID, expiry uint) error
// func DeleteRetentionPolicy(id uuid.UUID)

func (o *ORM) GetAllJobs() ([]*supervisor.Job, error) {
	l := []*supervisor.Job{}
	result, err := o.db.Query("GetAllJobs")
	if err != nil {
		return l, err
	}
	for result.Next() {
		j := &supervisor.Job{}
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
	}
	return l, nil
}

// func CreateJob(target uuid.UUID, store uuid.UUID, schedule uuid.UUID, retention uuid.UUID) (uuid.UUID, error)
// func PauseJob(id uuid.UUID) error
// func UnpauseJob(id uuid.UUID) error
// func DeleteJob(id uuid.UUID) error

// func CreateArchive(job uuid.UUID, key string) (id uuid.UUID, error)
// func DeleteArchive(id uuid.UUID) error

// func CreateTask(op string, args string, job uuid.UUID) (uuid.UUID, error)
// func CompleteTask(id uuid.UUID) error
// func CancelTask(id uuid.UUID) error
// func UpdateTaskLog(id uuid.UUID, log string) error
