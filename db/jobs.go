package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/shield/timespec"
)

type Job struct {
	UUID           uuid.UUID `json:"uuid"`
	Name           string    `json:"name"`
	Summary        string    `json:"summary"`
	RetentionName  string    `json:"retention_name"`
	RetentionUUID  uuid.UUID `json:"retention_uuid"`
	Expiry         int       `json:"expiry"`
	ScheduleName   string    `json:"schedule_name"`
	ScheduleUUID   uuid.UUID `json:"schedule_uuid"`
	ScheduleWhen   string    `json:"schedule_when"`
	Paused         bool      `json:"paused"`
	StoreUUID      uuid.UUID `json:"store_uuid"`
	StoreName      string    `json:"store_name"`
	StorePlugin    string    `json:"store_plugin"`
	StoreEndpoint  string    `json:"store_endpoint"`
	TargetUUID     uuid.UUID `json:"target_uuid"`
	TargetName     string    `json:"target_name"`
	TargetPlugin   string    `json:"target_plugin"`
	TargetEndpoint string    `json:"target_endpoint"`
	Agent          string    `json:"agent"`

	Spec    *timespec.Spec `json:"-"`
	NextRun time.Time      `json:"-"`
}

type JobFilter struct {
	SkipPaused   bool
	SkipUnpaused bool

	SearchName string

	ForTarget    string
	ForStore     string
	ForSchedule  string
	ForRetention string
	ExactMatch   bool
}

func (f *JobFilter) Query() (string, []interface{}) {
	wheres := []string{"j.uuid = j.uuid"}
	var args []interface{}
	if f.SearchName != "" {
		comparator := "LIKE"
		toAdd := Pattern(f.SearchName)
		if f.ExactMatch {
			comparator = "="
			toAdd = f.SearchName
		}
		wheres = append(wheres, fmt.Sprintf("j.name %s ?", comparator))
		args = append(args, toAdd)
	}
	if f.ForTarget != "" {
		wheres = append(wheres, "target_uuid = ?")
		args = append(args, f.ForTarget)
	}
	if f.ForStore != "" {
		wheres = append(wheres, "store_uuid = ?")
		args = append(args, f.ForStore)
	}
	if f.ForSchedule != "" {
		wheres = append(wheres, "schedule_uuid = ?")
		args = append(args, f.ForSchedule)
	}
	if f.ForRetention != "" {
		wheres = append(wheres, "retention_uuid = ?")
		args = append(args, f.ForRetention)
	}
	if f.SkipPaused || f.SkipUnpaused {
		wheres = append(wheres, "paused = ?")
		if f.SkipPaused {
			args = append(args, 0)
		} else {
			args = append(args, 1)
		}
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
	`, args
}

func (db *DB) GetAllJobs(filter *JobFilter) ([]*Job, error) {
	if filter == nil {
		filter = &JobFilter{}
	}

	l := []*Job{}
	query, args := filter.Query()
	r, err := db.Query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		ann := &Job{}
		var this, retention, schedule, store, target NullUUID
		if err = r.Scan(
			&this, &ann.Name, &ann.Summary, &ann.Paused,
			&ann.RetentionName, &retention, &ann.Expiry,
			&ann.ScheduleName, &schedule, &ann.ScheduleWhen,
			&store, &ann.StoreName, &ann.StorePlugin, &ann.StoreEndpoint,
			&target, &ann.TargetName, &ann.TargetPlugin, &ann.TargetEndpoint,
			&ann.Agent); err != nil {
			return l, err
		}
		ann.UUID = this.UUID
		ann.RetentionUUID = retention.UUID
		ann.ScheduleUUID = schedule.UUID
		ann.StoreUUID = store.UUID
		ann.TargetUUID = target.UUID

		l = append(l, ann)
	}

	return l, nil
}

func (db *DB) GetJob(id uuid.UUID) (*Job, error) {
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

			WHERE j.uuid = ?`, id.String())
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	ann := &Job{}
	var this, retention, schedule, store, target NullUUID
	if err = r.Scan(
		&this, &ann.Name, &ann.Summary, &ann.Paused,
		&ann.RetentionName, &retention, &ann.Expiry,
		&ann.ScheduleName, &schedule, &ann.ScheduleWhen,
		&store, &ann.StoreName, &ann.StorePlugin, &ann.StoreEndpoint,
		&target, &ann.TargetName, &ann.TargetPlugin, &ann.TargetEndpoint,
		&ann.Agent); err != nil {
		return nil, err
	}
	ann.UUID = this.UUID
	ann.RetentionUUID = retention.UUID
	ann.ScheduleUUID = schedule.UUID
	ann.StoreUUID = store.UUID
	ann.TargetUUID = target.UUID

	return ann, nil
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

func (j *Job) Reschedule() (err error) {
	if j.Spec == nil {
		j.Spec, err = timespec.Parse(j.ScheduleWhen)
		if err != nil {
			return
		}
	}
	j.NextRun, err = j.Spec.Next(time.Now())
	return
}

func (j *Job) Runnable() bool {
	return j.Paused == false && !j.NextRun.After(time.Now())
}
