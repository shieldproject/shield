package db

import (
	"fmt"
	"strings"
	"supervisor"
	"timespec"

	"github.com/pborman/uuid"
)

type JobRepresentation struct {
	UUID uuid.UUID
	Tspec string
	Error error
}
type JobFailedError struct {
	FailedJobs []JobRepresentation
}

func (e JobFailedError) Error() string {
	var jobList []string
	for _, j := range e.FailedJobs {
		jobList = append(jobList, string(j.UUID))
	}
	return fmt.Sprintf("the following job(s) failed: %s", strings.Join(jobList, ", "))
}

type AnnotatedJob struct {
	UUID           string `json:"uuid"`
	Name           string `json:"name"`
	Summary        string `json:"summary"`
	RetentionName  string `json:"retention_name"`
	RetentionUUID  string `json:"retention_uuid"`
	Expiry         int    `json:"expiry"`
	ScheduleName   string `json:"schedule_name"`
	ScheduleUUID   string `json:"schedule_uuid"`
	Schedule       string `json:"schedule"`
	Paused         bool   `json:"paused"`
	StorePlugin    string `json:"store_plugin"`
	StoreEndpoint  string `json:"store_endpoint"`
	TargetPlugin   string `json:"target_plugin"`
	TargetEndpoint string `json:"target_endpoint"`
}

type JobFilter struct {
	SkipPaused   bool
	SkipUnpaused bool

	ForTarget    string
	ForStore     string
	ForSchedule  string
	ForRetention string
}

func (f *JobFilter) Args() []interface{} {
	var args []interface{}
	if f.ForTarget != "" {
		args = append(args, f.ForTarget)
	}
	if f.ForStore != "" {
		args = append(args, f.ForStore)
	}
	if f.ForSchedule != "" {
		args = append(args, f.ForSchedule)
	}
	if f.ForRetention != "" {
		args = append(args, f.ForRetention)
	}
	if f.SkipPaused || f.SkipUnpaused {
		if f.SkipPaused {
			args = append(args, 0)
		} else {
			args = append(args, 1)
		}
	}
	return args
}

func (f *JobFilter) Query() string {
	var wheres []string = []string{ "1" }
	if f.ForTarget != "" {
		wheres = append(wheres, "target_uuid = ?")
	}
	if f.ForStore != "" {
		wheres = append(wheres, "store_uuid = ?")
	}
	if f.ForSchedule != "" {
		wheres = append(wheres, "schedule_uuid = ?")
	}
	if f.ForRetention != "" {
		wheres = append(wheres, "retention_uuid = ?")
	}
	if f.SkipPaused || f.SkipUnpaused {
		wheres = append(wheres, "paused = ?")
	}

	return `
		SELECT j.uuid, j.name, j.summary, j.paused,
		       r.name, r.uuid, r.expiry,
		       sc.name, sc.uuid, sc.timespec,
		       s.plugin, s.endpoint,
		       t.plugin, t.endpoint

			FROM jobs j
				INNER JOIN retention  r  ON  r.uuid = j.retention_uuid
				INNER JOIN schedules sc  ON sc.uuid = j.schedule_uuid
				INNER JOIN stores     s  ON  s.uuid = j.store_uuid
				INNER JOIN targets    t  ON  t.uuid = j.target_uuid

			WHERE ` + strings.Join(wheres, " AND ") + `
			ORDER BY j.name, j.uuid ASC
	`
}

func (db *DB) GetAllAnnotatedJobs(filter *JobFilter) ([]*AnnotatedJob, error) {
	l := []*AnnotatedJob{}
	r, err := db.Query(filter.Query(), filter.Args()...)
	if err != nil {
		return l, err
	}

	for r.Next() {
		ann := &AnnotatedJob{}

		if err = r.Scan(
				&ann.UUID, &ann.Name, &ann.Summary, &ann.Paused,
				&ann.RetentionName, &ann.RetentionUUID, &ann.Expiry,
				&ann.ScheduleName,  &ann.ScheduleUUID, &ann.Schedule,
				&ann.StorePlugin, &ann.StoreEndpoint,
				&ann.TargetPlugin, &ann.TargetEndpoint); err != nil {
			return l, err
		}

		l = append(l, ann)
	}

	return l, nil
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
	e := JobFailedError{}
	for result.Next() {
		j := &supervisor.Job{Target: &supervisor.PluginConfig{}, Store: &supervisor.PluginConfig{}}
		var id, tspec string
		var expiry int
		//var paused bool
		err = result.Scan(&id, &j.Paused,
			&j.Target.Plugin, &j.Target.Endpoint,
			&j.Store.Plugin, &j.Store.Endpoint,
			&tspec, &expiry)
		j.UUID = uuid.Parse(id)
		if err != nil {
			e.FailedJobs = append(e.FailedJobs, JobRepresentation{j.UUID, tspec, err})
		}
		j.Spec, err = timespec.Parse(tspec)
		if err != nil {
			e.FailedJobs = append(e.FailedJobs, JobRepresentation{j.UUID, tspec, err})
		}
		l = append(l, j)
	}
	if len(e.FailedJobs) == 0 {
		return l, nil
	}
	return l, e
}

func (db *DB) PauseOrUnpauseJob(id uuid.UUID, pause bool) (bool, error) {
	n, err := db.Count(
		`SELECT uuid FROM jobs WHERE uuid = ? AND paused = ?`,
		id.String(), !pause)
	if n == 0 || err != nil {
		return false, err
	}

	return true, db.Exec(
		`UPDATE jobs SET paused = ? WHERE uuid = ? AND paused = ?`,
		pause, id.String(), !pause)
}

func (db *DB) PauseJob(id uuid.UUID) (bool, error) {
	return db.PauseOrUnpauseJob(id, true)
}

func (db *DB) UnpauseJob(id uuid.UUID) (bool, error) {
	return db.PauseOrUnpauseJob(id, false)
}

func (db *DB) AnnotateJob(id uuid.UUID, name string, summary string) error {
	return db.Exec(
		`UPDATE jobs SET name = ?, summary = ? WHERE uuid = ?`,
		name, summary, id.String(),
	)
}

func (db *DB) CreateJob(target, store, schedule, retention string) (uuid.UUID, error) {
	id := uuid.NewRandom()
	return id, db.Exec(
		`INSERT INTO jobs (uuid, target_uuid, store_uuid, schedule_uuid, retention_uuid)
			VALUES (?, ?, ?, ?, ?)`,
		id.String(), target, store, schedule, retention,
	)
}

func (db *DB) UpdateJob(id uuid.UUID, target, store, schedule, retention string) error {
	return db.Exec(
		`UPDATE jobs SET target_uuid = ?, store_uuid = ?, schedule_uuid = ?, retention_uuid = ? WHERE uuid = ?`,
		target, store, schedule, retention, id.String(),
	)
}

func (db *DB) DeleteJob(id uuid.UUID) (bool, error) {
	return true, db.Exec(
		`DELETE FROM jobs WHERE uuid = ?`,
		id.String(),
	)
}
