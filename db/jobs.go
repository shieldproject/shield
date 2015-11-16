package db

import (
	"github.com/pborman/uuid"
	"strings"
)

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
	Agent          string `json:"agent"`
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
	var wheres []string = []string{"1"}
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
		       t.plugin, t.endpoint, t.agent

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
	defer r.Close()

	for r.Next() {
		ann := &AnnotatedJob{}

		if err = r.Scan(
			&ann.UUID, &ann.Name, &ann.Summary, &ann.Paused,
			&ann.RetentionName, &ann.RetentionUUID, &ann.Expiry,
			&ann.ScheduleName, &ann.ScheduleUUID, &ann.Schedule,
			&ann.StorePlugin, &ann.StoreEndpoint,
			&ann.TargetPlugin, &ann.TargetEndpoint, &ann.Agent); err != nil {
			return l, err
		}

		l = append(l, ann)
	}

	return l, nil
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

func (db *DB) CreateJob(target, store, schedule, retention string, paused bool) (uuid.UUID, error) {
	id := uuid.NewRandom()
	return id, db.Exec(
		`INSERT INTO jobs (uuid, target_uuid, store_uuid, schedule_uuid, retention_uuid, paused)
			VALUES (?, ?, ?, ?, ?, ?)`,
		id.String(), target, store, schedule, retention, paused,
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
