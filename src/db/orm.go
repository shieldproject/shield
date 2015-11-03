package db

import (
	"fmt"
	"supervisor"
	"timespec"

	"database/sql"
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
	o.db.Cache("GetAllAnnotatedSchedules",
		`SELECT uuid, name, summary, timespec, 0 AS n FROM schedules ORDER BY name, uuid ASC`)
	o.db.Cache("GetAllAnnotatedUnusedSchedules",
		`SELECT DISTINCT s.uuid, s.name, s.summary, s.timespec, COUNT(j.uuid) AS n
			FROM schedules s
				LEFT JOIN jobs j
					ON j.schedule_uuid = s.uuid
			GROUP BY s.uuid
			HAVING n = 0
			ORDER BY s.name, s.uuid ASC`)
	o.db.Cache("GetAllAnnotatedUsedSchedules",
		`SELECT DISTINCT s.uuid, s.name, s.summary, s.timespec, COUNT(j.uuid) AS n
			FROM schedules s
				LEFT JOIN jobs j
					ON j.schedule_uuid = s.uuid
			GROUP BY s.uuid
			HAVING n > 0
			ORDER BY s.name, s.uuid ASC`)
	o.db.Cache("CreateSchedule",
		`INSERT INTO schedules (uuid, timespec) VALUES (?, ?)`)
	o.db.Cache("UpdateSchedule",
		`UPDATE schedules SET timespec = ? WHERE uuid = ?`)
	o.db.Cache("AnnotateSchedule",
		`UPDATE schedules SET name = ?, summary = ? WHERE uuid = ?`)
	o.db.Cache("JobsUsingSchedule",
		`SELECT COUNT(uuid) FROM jobs WHERE jobs.schedule_uuid = ?`)
	o.db.Cache("DeleteSchedule",
		`DELETE FROM schedules WHERE uuid = ?`)

	o.db.Cache("GetAllAnnotatedRetentionPolicies",
		`SELECT uuid, name, summary, expiry, 0 AS n FROM retention ORDER BY name, uuid ASC`)
	o.db.Cache("GetAllAnnotatedUnusedRetentionPolicies",
		`SELECT DISTINCT r.uuid, r.name, r.summary, r.expiry, COUNT(j.uuid) AS n
			FROM retention r
				LEFT JOIN jobs j
					ON j.retention_uuid = r.uuid
			GROUP BY r.uuid
			HAVING n = 0
			ORDER BY r.name, r.uuid ASC`)
	o.db.Cache("GetAllAnnotatedUsedRetentionPolicies",
		`SELECT DISTINCT r.uuid, r.name, r.summary, r.expiry, COUNT(j.uuid) AS n
			FROM retention r
				LEFT JOIN jobs j
					ON j.retention_uuid = r.uuid
			GROUP BY r.uuid
			HAVING n > 0
			ORDER BY r.name, r.uuid ASC`)
	o.db.Cache("CreateRetentionPolicy",
		`INSERT INTO retention (uuid, expiry) VALUES (?, ?)`)
	o.db.Cache("UpdateRetentionPolicy",
		`UPDATE retention SET expiry = ? WHERE uuid = ?`)
	o.db.Cache("AnnotateRetentionPolicy",
		`UPDATE retention SET name = ?, summary = ? WHERE uuid = ?`)
	o.db.Cache("JobsUsingRetentionPolicy",
		`SELECT COUNT(uuid) FROM jobs WHERE jobs.retention_uuid = ?`)
	o.db.Cache("DeleteRetentionPolicy",
		`DELETE FROM retention WHERE uuid = ?`)

	o.db.Cache("GetAllAnnotatedTargets",
		`SELECT uuid, name, summary, plugin, endpoint, 0 AS n
			FROM targets
			ORDER BY name, uuid ASC`)
	o.db.Cache("GetAllAnnotatedUnusedTargets",
		`SELECT DISTINCT t.uuid, t.name, t.summary, t.plugin, t.endpoint, COUNT(j.uuid) AS n
			FROM targets t
				LEFT JOIN jobs j
					ON j.target_uuid = t.uuid
			GROUP BY t.uuid
			HAVING n = 0
			ORDER BY t.name, t.uuid ASC`)
	o.db.Cache("GetAllAnnotatedUsedTargets",
		`SELECT DISTINCT t.uuid, t.name, t.summary, t.plugin, t.endpoint, COUNT(j.uuid) AS n
			FROM targets t
				LEFT JOIN jobs j
					ON j.target_uuid = t.uuid
			GROUP BY t.uuid
			HAVING n > 0
			ORDER BY t.name, t.uuid ASC`)
	o.db.Cache("GetAllAnnotatedTargetsFiltered",
		`SELECT uuid, name, summary, plugin, endpoint, 0 AS n
			FROM targets
			WHERE plugin = ?
			ORDER BY name, uuid ASC`)
	o.db.Cache("GetAllAnnotatedUnusedTargetsFiltered",
		`SELECT DISTINCT t.uuid, t.name, t.summary, t.plugin, t.endpoint, COUNT(j.uuid) AS n
			FROM targets t
				LEFT JOIN jobs j
					ON j.target_uuid = t.uuid
			WHERE t.plugin = ?
			GROUP BY t.uuid
			HAVING n = 0
			ORDER BY t.name, t.uuid ASC`)
	o.db.Cache("GetAllAnnotatedUsedTargetsFiltered",
		`SELECT DISTINCT t.uuid, t.name, t.summary, t.plugin, t.endpoint, COUNT(j.uuid) AS n
			FROM targets t
				LEFT JOIN jobs j
					ON j.target_uuid = t.uuid
			WHERE t.plugin = ?
			GROUP BY t.uuid
			HAVING n > 0
			ORDER BY t.name, t.uuid ASC`)
	o.db.Cache("CreateTarget",
		`INSERT INTO targets (uuid, plugin, endpoint) VALUES (?, ?, ?)`)
	o.db.Cache("UpdateTarget",
		`UPDATE targets SET plugin = ?, endpoint = ? WHERE uuid = ?`)
	o.db.Cache("AnnotateTarget",
		`UPDATE targets SET name = ?, summary = ? WHERE uuid = ?`)
	o.db.Cache("JobsUsingTarget",
		`SELECT COUNT(uuid) FROM jobs WHERE jobs.target_uuid = ?`)
	o.db.Cache("DeleteTarget",
		`DELETE FROM targets WHERE uuid = ?`)

	o.db.Cache("GetAllAnnotatedStores",
		`SELECT uuid, name, summary, plugin, endpoint FROM stores ORDER BY name, uuid ASC`)
	o.db.Cache("CreateStore",
		`INSERT INTO stores (uuid, plugin, endpoint) VALUES (?, ?, ?)`)
	o.db.Cache("UpdateStore",
		`UPDATE stores SET plugin = ?, endpoint = ? WHERE uuid = ?`)
	o.db.Cache("AnnotateStore",
		`UPDATE stores SET name = ?, summary = ? WHERE uuid = ?`)
	o.db.Cache("JobsUsingStore",
		`SELECT COUNT(uuid) FROM jobs WHERE jobs.store_uuid = ?`)
	o.db.Cache("DeleteStore",
		`DELETE FROM stores WHERE uuid = ?`)

	o.db.Cache("GetAllJobs",
		`SELECT jobs.uuid, jobs.paused, targets.plugin, targets.endpoint, stores.plugin, stores.endpoint, schedules.timespec, retention.expiry
		FROM jobs
		INNER JOIN targets ON targets.uuid = jobs.target_uuid
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

// func (o *ORM) AnnotateArchive(id uuid.UUID, notes string) error
// func (o *ORM) AnnotateJob(id uuid.UUID, name string, summary string) error

func (o *ORM) AnnotateRetentionPolicy(id uuid.UUID, name string, summary string) error {
	return o.db.Exec("AnnotateRetentionPolicy", name, summary, id.String())
}

func (o *ORM) AnnotateSchedule(id uuid.UUID, name string, summary string) error {
	return o.db.Exec("AnnotateSchedule", name, summary, id.String())
}

func (o *ORM) AnnotateStore(id uuid.UUID, name string, summary string) error {
	return o.db.Exec("AnnotateStore", name, summary, id.String())
}

func (o *ORM) AnnotateTarget(id uuid.UUID, name string, summary string) error {
	return o.db.Exec("AnnotateTarget", name, summary, id.String())
}

// func (o *ORM) AnnotateTask(id uuid.UUID, owner string) error

type AnnotatedSchedule struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	When    string `json:"when"`
}

func (o *ORM) GetAllAnnotatedSchedules(subset bool, unused bool) ([]*AnnotatedSchedule, error) {
	l := []*AnnotatedSchedule{}
	var q string
	switch {
	case subset && unused: q = "GetAllAnnotatedUnusedSchedules"
	case subset && !unused: q = "GetAllAnnotatedUsedSchedules"
	default: q = "GetAllAnnotatedSchedules"
	}

	r, err := o.db.Query(q)
	if err != nil {
		return l, err
	}

	for r.Next() {
		ann := &AnnotatedSchedule{}
		var n uint

		if err = r.Scan(&ann.UUID, &ann.Name, &ann.Summary, &ann.When, &n); err != nil {
			return l, err
		}

		l = append(l, ann)
	}

	return l, nil
}

type AnnotatedRetentionPolicy struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Expires uint   `json:"expires"`
}

func (o *ORM) GetAllAnnotatedRetentionPolicies(subset bool, unused bool) ([]*AnnotatedRetentionPolicy, error) {
	l := []*AnnotatedRetentionPolicy{}
	var q string
	switch {
	case subset && unused: q = "GetAllAnnotatedUnusedRetentionPolicies"
	case subset && !unused: q = "GetAllAnnotatedUsedRetentionPolicies"
	default: q = "GetAllAnnotatedRetentionPolicies"
	}

	r, err := o.db.Query(q)
	if err != nil {
		return l, err
	}

	for r.Next() {
		ann := &AnnotatedRetentionPolicy{}
		var n uint

		if err = r.Scan(&ann.UUID, &ann.Name, &ann.Summary, &ann.Expires, &n); err != nil {
			return l, err
		}

		l = append(l, ann)
	}

	return l, nil
}

type AnnotatedTarget struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	Summary  string `json:"summary"`
	Plugin   string `json:"plugin"`
	Endpoint string `json:"endpoint"`
}

func (o *ORM) GetAllAnnotatedTargets(filter1 bool, unused bool, filter2 bool, plugin string) ([]*AnnotatedTarget, error) {
	l := []*AnnotatedTarget{}
	var q string
	switch {
	case filter1 &&  unused: q = "GetAllAnnotatedUnusedTargets"
	case filter1 && !unused: q = "GetAllAnnotatedUsedTargets"
	default: q = "GetAllAnnotatedTargets"
	}

	var r *sql.Rows
	var err error

	if filter2 {
		r, err = o.db.Query(q + "Filtered", plugin)
	} else {
		r, err = o.db.Query(q)
	}
	if err != nil {
		return l, err
	}

	for r.Next() {
		ann := &AnnotatedTarget{}
		var n uint

		if err = r.Scan(&ann.UUID, &ann.Name, &ann.Summary, &ann.Plugin, &ann.Endpoint, &n); err != nil {
			return l, err
		}

		l = append(l, ann)
	}

	return l, nil
}

func (o *ORM) CreateTarget(plugin string, endpoint interface{}) (uuid.UUID, error) {
	id := uuid.NewRandom()
	return id, o.db.Exec("CreateTarget", id.String(), plugin, endpoint)
}

func (o *ORM) UpdateTarget(id uuid.UUID, plugin string, endpoint interface{}) error {
	return o.db.Exec("UpdateTarget", plugin, endpoint, id.String())
}

func (o *ORM) DeleteTarget(id uuid.UUID) (bool, error) {
	r, err := o.db.Query("JobsUsingTarget", id.String())
	if err != nil {
		return false, err
	}
	defer r.Close()

	// already deleted?
	if !r.Next() {
		return true, nil
	}

	var numJobs int
	if err = r.Scan(&numJobs); err != nil {
		return false, err
	}

	if numJobs < 0 {
		return false, fmt.Errorf("Target %s is in used by %d (negative) Jobs", id.String(), numJobs)
	}
	if numJobs > 0 {
		return false, nil
	}

	r.Close()
	return true, o.db.Exec("DeleteTarget", id.String())
}

type AnnotatedStore struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	Summary  string `json:"summary"`
	Plugin   string `json:"plugin"`
	Endpoint string `json:"endpoint"`
}

func (o *ORM) GetAllAnnotatedStores() ([]*AnnotatedStore, error) {
	l := []*AnnotatedStore{}

	r, err := o.db.Query("GetAllAnnotatedStores")
	if err != nil {
		return l, err
	}

	for r.Next() {
		ann := &AnnotatedStore{}
		if err = r.Scan(&ann.UUID, &ann.Name, &ann.Summary, &ann.Plugin, &ann.Endpoint); err != nil {
			return l, err
		}

		l = append(l, ann)
	}

	return l, nil
}

func (o *ORM) CreateStore(plugin string, endpoint interface{}) (uuid.UUID, error) {
	id := uuid.NewRandom()
	return id, o.db.Exec("CreateStore", id.String(), plugin, endpoint)
}

func (o *ORM) UpdateStore(id uuid.UUID, plugin string, endpoint interface{}) error {
	return o.db.Exec("UpdateStore", plugin, endpoint, id.String())
}

func (o *ORM) DeleteStore(id uuid.UUID) (bool, error) {
	r, err := o.db.Query("JobsUsingStore", id.String())
	if err != nil {
		return false, err
	}
	defer r.Close()

	// already deleted?
	if !r.Next() {
		return true, nil
	}

	var numJobs int
	if err = r.Scan(&numJobs); err != nil {
		return false, err
	}

	if numJobs < 0 {
		return false, fmt.Errorf("Store %s is in used by %d (negative) Jobs", id.String(), numJobs)
	}
	if numJobs > 0 {
		return false, nil
	}

	r.Close()
	return true, o.db.Exec("DeleteStore", id.String())
}

func (o *ORM) CreateSchedule(timespec string) (uuid.UUID, error) {
	id := uuid.NewRandom()
	return id, o.db.Exec("CreateSchedule", id.String(), timespec)
}

func (o *ORM) UpdateSchedule(id uuid.UUID, timespec string) error {
	return o.db.Exec("UpdateSchedule", timespec, id.String())
}

func (o *ORM) DeleteSchedule(id uuid.UUID) (bool, error) {
	r, err := o.db.Query("JobsUsingSchedule", id.String())
	if err != nil {
		return false, err
	}
	defer r.Close()

	// already deleted?
	if !r.Next() {
		return true, nil
	}

	var numJobs int
	if err = r.Scan(&numJobs); err != nil {
		return false, err
	}

	if numJobs < 0 {
		return false, fmt.Errorf("Schedule %s is in used by %d (negative) Jobs", id.String(), numJobs)
	}
	if numJobs > 0 {
		return false, nil
	}

	r.Close()
	return true, o.db.Exec("DeleteSchedule", id.String())
}

func (o *ORM) CreateRetentionPolicy(expiry uint) (uuid.UUID, error) {
	id := uuid.NewRandom()
	return id, o.db.Exec("CreateRetentionPolicy", id.String(), expiry)
}

func (o *ORM) UpdateRetentionPolicy(id uuid.UUID, expiry uint) error {
	return o.db.Exec("UpdateRetentionPolicy", expiry, id.String())
}

func (o *ORM) DeleteRetentionPolicy(id uuid.UUID) (bool, error) {
	r, err := o.db.Query("JobsUsingRetentionPolicy", id.String())
	if err != nil {
		return false, err
	}
	defer r.Close()

	// already deleted?
	if !r.Next() {
		return true, nil
	}

	var numJobs int
	if err = r.Scan(&numJobs); err != nil {
		return false, err
	}

	if numJobs < 0 {
		return false, fmt.Errorf("Retention policy %s is in used by %d (negative) Jobs", id.String(), numJobs)
	}
	if numJobs > 0 {
		return false, nil
	}

	r.Close()
	return true, o.db.Exec("DeleteRetentionPolicy", id.String())
}

func (o *ORM) GetAllJobs() ([]*supervisor.Job, error) {
	l := []*supervisor.Job{}
	result, err := o.db.Query("GetAllJobs")
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

// func (o *ORM) CreateJob(target uuid.UUID, store uuid.UUID, schedule uuid.UUID, retention uuid.UUID) (uuid.UUID, error)
// func (o *ORM) PauseJob(id uuid.UUID) error
// func (o *ORM) UnpauseJob(id uuid.UUID) error
// func (o *ORM) DeleteJob(id uuid.UUID) error

// func (o *ORM) CreateArchive(job uuid.UUID, key string) (id uuid.UUID, error)
// func (o *ORM) DeleteArchive(id uuid.UUID) error

// func (o *ORM) CreateTask(op string, args string, job uuid.UUID) (uuid.UUID, error)
// func (o *ORM) CompleteTask(id uuid.UUID) error
// func (o *ORM) CancelTask(id uuid.UUID) error
// func (o *ORM) UpdateTaskLog(id uuid.UUID, log string) error
