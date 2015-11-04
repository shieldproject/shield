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
	return nil
}

func (o *ORM) schemaVersion() (uint, error) {
	r, err := o.db.Query(`SELECT version FROM schema_info LIMIT 1`)
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
	o.db.Exec(`CREATE TABLE schema_info (
                           version INTEGER
               )`)
	o.db.Exec(`INSERT INTO schema_info VALUES (1)`)

	o.db.Exec(`CREATE TABLE targets (
                 uuid      UUID PRIMARY KEY,
                 name      TEXT,
                 summary   TEXT,
                 plugin    TEXT,
                 endpoint  TEXT
               )`)

	o.db.Exec(`CREATE TABLE stores (
                 uuid      UUID PRIMARY KEY,
                 name      TEXT,
                 summary   TEXT,
                 plugin    TEXT,
                 endpoint  TEXT
               )`)

	o.db.Exec(`CREATE TABLE schedules (
                 uuid      UUID PRIMARY KEY,
                 name      TEXT,
                 summary   TEXT,
                 timespec  TEXT
               )`)

	o.db.Exec(`CREATE TABLE retention (
                 uuid     UUID PRIMARY KEY,
                 name     TEXT,
                 summary  TEXT,
                 expiry   INTEGER
               )`)

	o.db.Exec(`CREATE TABLE jobs (
                 uuid            UUID PRIMARY KEY,
                 target_uuid     UUID,
                 store_uuid      UUID,
                 schedule_uuid   UUID,
                 retention_uuid  UUID,
                 paused          BOOLEAN,
                 name            TEXT,
                 summary         TEXT
               )`)

	o.db.Exec(`CREATE TABLE archives (
                 uuid         UUID PRIMARY KEY,
                 target_uuid  UUID,
                 store_uuid   UUID,
                 store_key    TEXT,

                 taken_at     timestamp without time zone,
                 expires_at   timestamp without time zone,
                 notes        TEXT
               )`)

	o.db.Exec(`CREATE TABLE tasks (
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
	return o.db.Exec(
		`UPDATE retention SET name = ?, summary = ? WHERE uuid = ?`,
		name, summary, id.String(),
	)
}

func (o *ORM) AnnotateSchedule(id uuid.UUID, name string, summary string) error {
	return o.db.Exec(
		`UPDATE schedules SET name = ?, summary = ? WHERE uuid = ?`,
		name, summary, id.String(),
	)
}

func (o *ORM) AnnotateStore(id uuid.UUID, name string, summary string) error {
	return o.db.Exec(
		`UPDATE stores SET name = ?, summary = ? WHERE uuid = ?`,
		name, summary, id.String(),
	)
}

func (o *ORM) AnnotateTarget(id uuid.UUID, name string, summary string) error {
	return o.db.Exec(
		`UPDATE targets SET name = ?, summary = ? WHERE uuid = ?`,
		name, summary, id.String(),
	)
}

// func (o *ORM) AnnotateTask(id uuid.UUID, owner string) error

type AnnotatedSchedule struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	When    string `json:"when"`
}

type ScheduleFilter struct {
	SkipUsed   bool
	SkipUnused bool
}

func (f *ScheduleFilter) Query() string {
	if !f.SkipUsed && !f.SkipUnused {
		return `
			SELECT uuid, name, summary, timespec, -1 AS n
				FROM schedules
				ORDER BY name, uuid ASC
		`
	}

	// by default, show schedules with no attached jobs (unused)
	having := `HAVING n = 0`
	if f.SkipUnused {
		// otherwise, only show schedules that have attached jobs
		having = `HAVING n > 0`
	}

	return `
		SELECT DISTINCT s.uuid, s.name, s.summary, s.timespec, COUNT(j.uuid) AS n
			FROM schedules s
				LEFT JOIN jobs j
					ON j.schedule_uuid = s.uuid
			GROUP BY s.uuid
			` + having + `
			ORDER BY s.name, s.uuid ASC
	`
}

func (o *ORM) GetAllAnnotatedSchedules(filter *ScheduleFilter) ([]*AnnotatedSchedule, error) {
	l := []*AnnotatedSchedule{}
	r, err := o.db.Query(filter.Query())
	if err != nil {
		return l, err
	}

	for r.Next() {
		ann := &AnnotatedSchedule{}
		var n int

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

type RetentionFilter struct {
	SkipUsed bool
	SkipUnused bool
}

func (f *RetentionFilter) Query() string {
	if !f.SkipUsed && !f.SkipUnused {
		return `
			SELECT uuid, name, summary, expiry, -1 AS n
				FROM retention
				ORDER BY name, uuid ASC
		`
	}

	// by default, show retention policies with no attached jobs (unused)
	having := `HAVING n = 0`
	if f.SkipUnused {
		// otherwise, only show retention policies that have attached jobs
		having = `HAVING n > 0`
	}

	return `
		SELECT DISTINCT r.uuid, r.name, r.summary, r.expiry, COUNT(j.uuid) AS n
			FROM retention r
				LEFT JOIN jobs j
					ON j.retention_uuid = r.uuid
			GROUP BY r.uuid
			` + having + `
			ORDER BY r.name, r.uuid ASC
	`
}

func (o *ORM) GetAllAnnotatedRetentionPolicies(filter *RetentionFilter) ([]*AnnotatedRetentionPolicy, error) {
	l := []*AnnotatedRetentionPolicy{}
	r, err := o.db.Query(filter.Query())
	if err != nil {
		return l, err
	}

	for r.Next() {
		ann := &AnnotatedRetentionPolicy{}
		var n int

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

type TargetFilter struct {
	SkipUsed bool
	SkipUnused bool
	ForPlugin string
}

func (f *TargetFilter) Args() []interface{} {
	args := []interface{}{}
	if f.ForPlugin != "" {
		args = append(args, f.ForPlugin)
	}
	return args
}

func (f *TargetFilter) Query() string {
	where := ""
	if f.ForPlugin != "" {
		where = "WHERE plugin = ?"
	}

	if !f.SkipUsed && !f.SkipUnused {
		return `
			SELECT uuid, name, summary, plugin, endpoint, -1 AS n
				FROM targets ` + where + `
				ORDER BY name, uuid ASC
		`
	}

	// by default, show targets with no attached jobs (unused)
	having := `HAVING n = 0`
	if f.SkipUnused {
		// otherwise, only show targets that have attached jobs
		having = `HAVING n > 0`
	}

	return `
		SELECT DISTINCT t.uuid, t.name, t.summary, t.plugin, t.endpoint, COUNT(j.uuid) AS n
			FROM targets t
				LEFT JOIN jobs j
					ON j.target_uuid = t.uuid
			` + where + ` GROUP BY t.uuid
			` + having + `
			ORDER BY t.name, t.uuid ASC
	`
}

func (o *ORM) GetAllAnnotatedTargets(filter *TargetFilter) ([]*AnnotatedTarget, error) {
	l := []*AnnotatedTarget{}
	r, err := o.db.Query(filter.Query(), filter.Args()...)
	if err != nil {
		return l, err
	}

	for r.Next() {
		ann := &AnnotatedTarget{}
		var n int

		if err = r.Scan(&ann.UUID, &ann.Name, &ann.Summary, &ann.Plugin, &ann.Endpoint, &n); err != nil {
			return l, err
		}

		l = append(l, ann)
	}

	return l, nil
}

func (o *ORM) CreateTarget(plugin string, endpoint interface{}) (uuid.UUID, error) {
	id := uuid.NewRandom()
	return id, o.db.Exec(
		`INSERT INTO targets (uuid, plugin, endpoint) VALUES (?, ?, ?)`,
		id.String(), plugin, endpoint,
	)
}

func (o *ORM) UpdateTarget(id uuid.UUID, plugin string, endpoint interface{}) error {
	return o.db.Exec(
		`UPDATE targets SET plugin = ?, endpoint = ? WHERE uuid = ?`,
		plugin, endpoint, id.String(),
	)
}

func (o *ORM) DeleteTarget(id uuid.UUID) (bool, error) {
	r, err := o.db.Query(
		`SELECT COUNT(uuid) FROM jobs WHERE jobs.target_uuid = ?`,
		id.String(),
	)
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
	return true, o.db.Exec(
		`DELETE FROM targets WHERE uuid = ?`,
		id.String(),
	)
}

type AnnotatedStore struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	Summary  string `json:"summary"`
	Plugin   string `json:"plugin"`
	Endpoint string `json:"endpoint"`
}

type StoreFilter struct {
	SkipUsed bool
	SkipUnused bool
	ForPlugin string
}

func (f *StoreFilter) Args() []interface{} {
	args := []interface{}{}
	if f.ForPlugin != "" {
		args = append(args, f.ForPlugin)
	}
	return args
}

func (f *StoreFilter) Query() string {
	where := ""
	if f.ForPlugin != "" {
		where = "WHERE plugin = ?"
	}

	if !f.SkipUsed && !f.SkipUnused {
		return `
			SELECT uuid, name, summary, plugin, endpoint, -1 AS n
				FROM stores ` + where + `
				ORDER BY name, uuid ASC
		`
	}

	// by default, show stores with no attached jobs (unused)
	having := `HAVING n = 0`
	if f.SkipUnused {
		// otherwise, only show stores that have attached jobs
		having = `HAVING n > 0`
	}

	return `
		SELECT DISTINCT s.uuid, s.name, s.summary, s.plugin, s.endpoint, COUNT(j.uuid) AS n
			FROM stores s
				LEFT JOIN jobs j
					ON j.store_uuid = s.uuid
			` + where + ` GROUP BY s.uuid
			` + having + `
			ORDER BY s.name, s.uuid ASC
	`
}

func (o *ORM) GetAllAnnotatedStores(filter *StoreFilter) ([]*AnnotatedStore, error) {
	l := []*AnnotatedStore{}
	r, err := o.db.Query(filter.Query(), filter.Args()...)
	if err != nil {
		return l, err
	}

	for r.Next() {
		ann := &AnnotatedStore{}
		var n int

		if err = r.Scan(&ann.UUID, &ann.Name, &ann.Summary, &ann.Plugin, &ann.Endpoint, &n); err != nil {
			return l, err
		}

		l = append(l, ann)
	}

	return l, nil
}

func (o *ORM) CreateStore(plugin string, endpoint interface{}) (uuid.UUID, error) {
	id := uuid.NewRandom()
	return id, o.db.Exec(
		`INSERT INTO stores (uuid, plugin, endpoint) VALUES (?, ?, ?)`,
		id.String(), plugin, endpoint,
	)
}

func (o *ORM) UpdateStore(id uuid.UUID, plugin string, endpoint interface{}) error {
	return o.db.Exec(
		`UPDATE stores SET plugin = ?, endpoint = ? WHERE uuid = ?`,
		plugin, endpoint, id.String(),
	)
}

func (o *ORM) DeleteStore(id uuid.UUID) (bool, error) {
	r, err := o.db.Query(
		`SELECT COUNT(uuid) FROM jobs WHERE jobs.store_uuid = ?`,
		id.String(),
	)
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
	return true, o.db.Exec(
		`DELETE FROM stores WHERE uuid = ?`,
		id.String(),
	)
}

func (o *ORM) CreateSchedule(timespec string) (uuid.UUID, error) {
	id := uuid.NewRandom()
	return id, o.db.Exec(
		`INSERT INTO schedules (uuid, timespec) VALUES (?, ?)`,
		id.String(), timespec,
	)
}

func (o *ORM) UpdateSchedule(id uuid.UUID, timespec string) error {
	return o.db.Exec(
		`UPDATE schedules SET timespec = ? WHERE uuid = ?`,
		timespec, id.String(),
	)
}

func (o *ORM) DeleteSchedule(id uuid.UUID) (bool, error) {
	r, err := o.db.Query(
		`SELECT COUNT(uuid) FROM jobs WHERE jobs.schedule_uuid = ?`,
		id.String(),
	)
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
	return true, o.db.Exec(
		`DELETE FROM schedules WHERE uuid = ?`,
		id.String(),
	)
}

func (o *ORM) CreateRetentionPolicy(expiry uint) (uuid.UUID, error) {
	id := uuid.NewRandom()
	return id, o.db.Exec(
		`INSERT INTO retention (uuid, expiry) VALUES (?, ?)`,
		id.String(), expiry,
	)
}

func (o *ORM) UpdateRetentionPolicy(id uuid.UUID, expiry uint) error {
	return o.db.Exec(
		`UPDATE retention SET expiry = ? WHERE uuid = ?`,
		expiry, id.String(),
	)
}

func (o *ORM) DeleteRetentionPolicy(id uuid.UUID) (bool, error) {
	r, err := o.db.Query(
		`SELECT COUNT(uuid) FROM jobs WHERE jobs.retention_uuid = ?`,
		id.String(),
	)
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
	return true, o.db.Exec(
		`DELETE FROM retention WHERE uuid = ?`,
		id.String(),
	)
}

func (o *ORM) GetAllJobs() ([]*supervisor.Job, error) {
	l := []*supervisor.Job{}
	result, err := o.db.Query(`
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
