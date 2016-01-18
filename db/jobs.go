package db

import (
	"fmt"
	"strings"

	"github.com/pborman/uuid"
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
	ScheduleWhen   string `json:"schedule_when"`
	Paused         bool   `json:"paused"`
	StoreUUID      string `json:"store_uuid"`
	StoreName      string `json:"store_name"`
	StorePlugin    string `json:"store_plugin"`
	StoreEndpoint  string `json:"store_endpoint"`
	TargetUUID     string `json:"target_uuid"`
	TargetName     string `json:"target_name"`
	TargetPlugin   string `json:"target_plugin"`
	TargetEndpoint string `json:"target_endpoint"`
	Agent          string `json:"agent"`
}

type JobFilter struct {
	SkipPaused   bool
	SkipUnpaused bool

	SearchName string

	ForTarget    string
	ForStore     string
	ForSchedule  string
	ForRetention string
}

func (f *JobFilter) Args() []interface{} {
	var args []interface{}
	if f.SearchName != "" {
		args = append(args, Pattern(f.SearchName))
	}
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
	var wheres []string = []string{"j.uuid = j.uuid"}
	n := 1
	if f.SearchName != "" {
		wheres = append(wheres, fmt.Sprintf("j.name LIKE $%d", n))
		n++
	}
	if f.ForTarget != "" {
		wheres = append(wheres, fmt.Sprintf("target_uuid = $%d", n))
		n++
	}
	if f.ForStore != "" {
		wheres = append(wheres, fmt.Sprintf("store_uuid = $%d", n))
		n++
	}
	if f.ForSchedule != "" {
		wheres = append(wheres, fmt.Sprintf("schedule_uuid = $%d", n))
		n++
	}
	if f.ForRetention != "" {
		wheres = append(wheres, fmt.Sprintf("retention_uuid = $%d", n))
		n++
	}
	if f.SkipPaused || f.SkipUnpaused {
		wheres = append(wheres, fmt.Sprintf("paused = $%d", n))
		n++
	}

	return `
		SELECT j.uuid, j.name, j.summary, j.paused,
		       r.name, r.uuid, r.expiry,
		       sc.name, sc.uuid, sc.timespec,
		       s.uuid, s.name, s.plugin, s.endpoint,
		       t.uuid, t.name, t.plugin, t.endpoint, t.agent

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
			&ann.ScheduleName, &ann.ScheduleUUID, &ann.ScheduleWhen,
			&ann.StoreUUID, &ann.StoreName, &ann.StorePlugin, &ann.StoreEndpoint,
			&ann.TargetUUID, &ann.TargetName, &ann.TargetPlugin, &ann.TargetEndpoint,
			&ann.Agent); err != nil {
			return l, err
		}

		l = append(l, ann)
	}

	return l, nil
}

func (db *DB) GetAnnotatedJob(id uuid.UUID) (*AnnotatedJob, error) {
	r, err := db.Query(`
		SELECT j.uuid, j.name, j.summary, j.paused,
		       r.name, r.uuid, r.expiry,
		       sc.name, sc.uuid, sc.timespec,
		       s.uuid, s.name, s.plugin, s.endpoint,
		       t.uuid, t.name, t.plugin, t.endpoint, t.agent

			FROM jobs j
				INNER JOIN retention  r  ON  r.uuid = j.retention_uuid
				INNER JOIN schedules sc  ON sc.uuid = j.schedule_uuid
				INNER JOIN stores     s  ON  s.uuid = j.store_uuid
				INNER JOIN targets    t  ON  t.uuid = j.target_uuid

			WHERE j.uuid = $1`, id.String())
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	ann := &AnnotatedJob{}

	if err = r.Scan(
		&ann.UUID, &ann.Name, &ann.Summary, &ann.Paused,
		&ann.RetentionName, &ann.RetentionUUID, &ann.Expiry,
		&ann.ScheduleName, &ann.ScheduleUUID, &ann.ScheduleWhen,
		&ann.StoreUUID, &ann.StoreName, &ann.StorePlugin, &ann.StoreEndpoint,
		&ann.TargetUUID, &ann.TargetName, &ann.TargetPlugin, &ann.TargetEndpoint,
		&ann.Agent); err != nil {
		return nil, err
	}

	return ann, nil
}

func (db *DB) PauseOrUnpauseJob(id uuid.UUID, pause bool) (bool, error) {
	n, err := db.Count(
		`SELECT uuid FROM jobs WHERE uuid = $1 AND paused = $2`,
		id.String(), !pause)
	if n == 0 || err != nil {
		return false, err
	}

	return true, db.Exec(
		`UPDATE jobs SET paused = $1 WHERE uuid = $2 AND paused = $3`,
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
		`UPDATE jobs SET name = $1, summary = $2 WHERE uuid = $3`,
		name, summary, id.String(),
	)
}

func (db *DB) CreateJob(target, store, schedule, retention string, paused bool) (uuid.UUID, error) {
	id := uuid.NewRandom()
	return id, db.Exec(
		`INSERT INTO jobs (uuid, target_uuid, store_uuid, schedule_uuid, retention_uuid, paused)
			VALUES ($1, $2, $3, $4, $5, $6)`,
		id.String(), target, store, schedule, retention, paused,
	)
}

func (db *DB) UpdateJob(id uuid.UUID, target, store, schedule, retention string) error {
	return db.Exec(
		`UPDATE jobs SET target_uuid = $1, store_uuid = $2, schedule_uuid = $3, retention_uuid = $4 WHERE uuid = $5`,
		target, store, schedule, retention, id.String(),
	)
}

func (db *DB) DeleteJob(id uuid.UUID) (bool, error) {
	return true, db.Exec(
		`DELETE FROM jobs WHERE uuid = $1`,
		id.String(),
	)
}
