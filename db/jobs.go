package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pborman/uuid"
	. "github.com/starkandwayne/goutils/timestamp"

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
	LastRun        Timestamp `json:"last_run"`
	LastTaskStatus string    `json:"last_task_status"`

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

func (f *JobFilter) Query(driver string) (string, []interface{}) {
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
		wheres = append(wheres, "j.target_uuid = ?")
		args = append(args, f.ForTarget)
	}
	if f.ForStore != "" {
		wheres = append(wheres, "j.store_uuid = ?")
		args = append(args, f.ForStore)
	}
	if f.ForSchedule != "" {
		wheres = append(wheres, "j.schedule_uuid = ?")
		args = append(args, f.ForSchedule)
	}
	if f.ForRetention != "" {
		wheres = append(wheres, "j.retention_uuid = ?")
		args = append(args, f.ForRetention)
	}
	if f.SkipPaused || f.SkipUnpaused {
		wheres = append(wheres, "j.paused = ?")
		if f.SkipPaused {
			args = append(args, 0)
		} else {
			args = append(args, 1)
		}
	}

	switch driver {
	case "postgres", "sqlite3":
		return `
		WITH most_recent_job_task_at AS (
			SELECT job_uuid,  max(requested_at) AS requested_at FROM tasks WHERE status <> 'pending' GROUP BY job_uuid 
		),
		most_recent_job_task AS (
			SELECT t.started_at, t.status, t.job_uuid, t.uuid FROM tasks t, most_recent_job_task_at mr WHERE t.job_uuid = mr.job_uuid AND t.requested_at = mr.requested_at
		)
			SELECT j.uuid, j.name, j.summary, j.paused,
						 r.name, r.uuid, r.expiry,
						 sc.name, sc.uuid, sc.timespec,
						 s.uuid, s.name, s.plugin, s.endpoint,
						 t.uuid, t.name, t.plugin, t.endpoint, t.agent,
						 k.started_at, k.status

				FROM jobs j
					INNER JOIN retention  r  ON  r.uuid = j.retention_uuid
					INNER JOIN schedules sc  ON sc.uuid = j.schedule_uuid
					INNER JOIN stores     s  ON  s.uuid = j.store_uuid
					INNER JOIN targets    t  ON  t.uuid = j.target_uuid
					LEFT  JOIN most_recent_job_task k  ON  j.uuid = k.job_uuid

				WHERE ` + strings.Join(wheres, " AND ") + `
				ORDER BY j.name, j.uuid ASC
		`, args

	default:
		return `
			SELECT j.uuid, j.name, j.summary, j.paused,
			       r.name, r.uuid, r.expiry,
			       sc.name, sc.uuid, sc.timespec,
			       s.uuid, s.name, s.plugin, s.endpoint,
			       t.uuid, t.name, t.plugin, t.endpoint, t.agent,
			       null AS started_at, '' AS status

				FROM jobs j
					INNER JOIN retention  r  ON  r.uuid = j.retention_uuid
					INNER JOIN schedules sc  ON sc.uuid = j.schedule_uuid
					INNER JOIN stores     s  ON  s.uuid = j.store_uuid
					INNER JOIN targets    t  ON  t.uuid = j.target_uuid

				WHERE ` + strings.Join(wheres, " AND ") + `
				ORDER BY j.name, j.uuid ASC
		`, args
	}

}

func (db *DB) GetAllJobs(filter *JobFilter) ([]*Job, error) {
	if filter == nil {
		filter = &JobFilter{}
	}

	l := []*Job{}
	query, args := filter.Query(db.Driver)
	r, err := db.Query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		ann := &Job{}
		var (
			this, retention, schedule, store, target NullUUID
			last_run                                 *int64
			last_task_status                         sql.NullString
		)
		if err = r.Scan(
			&this, &ann.Name, &ann.Summary, &ann.Paused,
			&ann.RetentionName, &retention, &ann.Expiry,
			&ann.ScheduleName, &schedule, &ann.ScheduleWhen,
			&store, &ann.StoreName, &ann.StorePlugin, &ann.StoreEndpoint,
			&target, &ann.TargetName, &ann.TargetPlugin, &ann.TargetEndpoint,
			&ann.Agent, &last_run, &last_task_status); err != nil {
			return l, err
		}
		ann.UUID = this.UUID
		ann.RetentionUUID = retention.UUID
		ann.ScheduleUUID = schedule.UUID
		ann.StoreUUID = store.UUID
		ann.TargetUUID = target.UUID
		if last_run != nil {
			ann.LastRun = parseEpochTime(*last_run)
			if last_task_status.Valid {
				ann.LastTaskStatus = last_task_status.String
			}
		} else {
			ann.LastRun = Timestamp{}
			ann.LastTaskStatus = "pending"
		}

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
