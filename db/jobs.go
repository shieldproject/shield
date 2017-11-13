package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pborman/uuid"

	"github.com/starkandwayne/shield/timespec"
)

type Job struct {
	TenantUUID uuid.UUID `json:"-"`
	TargetUUID uuid.UUID `json:"-"`
	StoreUUID  uuid.UUID `json:"-"`
	PolicyUUID uuid.UUID `json:"-"`

	UUID     uuid.UUID `json:"uuid"`
	Name     string    `json:"name"`
	Summary  string    `json:"summary"`
	Expiry   int       `json:"expiry"`
	Schedule string    `json:"schedule"`
	Paused   bool      `json:"paused"`

	Target struct {
		UUID   uuid.UUID `json:"uuid"`
		Name   string    `json:"name"`
		Agent  string    `json:"agent"`
		Plugin string    `json:"plugin"`

		Endpoint string                 `json:"endpoint,omitempty"`
		Config   map[string]interface{} `json:"config,omitempty"`
	}

	Store struct {
		UUID    uuid.UUID `json:"uuid"`
		Name    string    `json:"name"`
		Agent   string    `json:"agent"`
		Plugin  string    `json:"plugin"`
		Summary string    `json:"summary"`

		Endpoint string                 `json:"endpoint,omitempty"`
		Config   map[string]interface{} `json:"config,omitempty"`
	} `json:"store"`

	Policy struct {
		UUID    uuid.UUID `json:"uuid"`
		Name    string    `json:"name"`
		Summary string    `json:"summary"`
	} `json:"policy"`

	Agent          string `json:"agent"`
	LastRun        int64  `json:"last_run"`
	LastTaskStatus string `json:"last_task_status"`

	Spec    *timespec.Spec `json:"-"`
	NextRun int64          `json:"-"`
}

func (j Job) Healthy() bool {
	return j.LastTaskStatus == "" || j.LastTaskStatus == "done"
}

type JobFilter struct {
	SkipPaused   bool
	SkipUnpaused bool

	Overdue bool

	SearchName string

	ForTenant  string
	ForTarget  string
	ForStore   string
	ForPolicy  string
	ExactMatch bool
}

func (f *JobFilter) Query(driver string) (string, []interface{}, error) {
	wheres := []string{"1"}
	args := []interface{}{}

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
	if f.ForTenant != "" {
		wheres = append(wheres, "j.tenant_uuid = ?")
		args = append(args, f.ForTenant)
	}
	if f.ForTarget != "" {
		wheres = append(wheres, "j.target_uuid = ?")
		args = append(args, f.ForTarget)
	}
	if f.ForStore != "" {
		wheres = append(wheres, "j.store_uuid = ?")
		args = append(args, f.ForStore)
	}
	if f.ForPolicy != "" {
		wheres = append(wheres, "j.retention_uuid = ?")
		args = append(args, f.ForPolicy)
	}
	if f.SkipPaused || f.SkipUnpaused {
		wheres = append(wheres, "j.paused = ?")
		if f.SkipPaused {
			args = append(args, 0)
		} else {
			args = append(args, 1)
		}
	}
	if f.Overdue {
		wheres = append(wheres, "j.next_run > ?")
		args = append(args, time.Now().Unix())
	}

	return `
	   WITH recent_tasks AS (
	           SELECT uuid AS task_uuid, job_uuid, started_at, status
	             FROM tasks
	            WHERE stopped_at IS NOT NULL
	         GROUP BY job_uuid
	        )

	   SELECT j.uuid, j.name, j.summary, j.paused, j.schedule, j.tenant_uuid,
	          r.name, r.summary, r.uuid, r.expiry,
	          s.uuid, s.name, s.plugin, s.endpoint, s.summary,
	          t.uuid, t.name, t.plugin, t.endpoint, t.agent,
	          k.started_at, k.status

	     FROM jobs j
	          INNER JOIN retention    r  ON  r.uuid = j.retention_uuid
	          INNER JOIN stores       s  ON  s.uuid = j.store_uuid
	          INNER JOIN targets      t  ON  t.uuid = j.target_uuid
	          LEFT  JOIN recent_tasks k  ON  j.uuid = k.job_uuid

	    WHERE ` + strings.Join(wheres, " AND ") + `
	 ORDER BY j.name, j.uuid ASC`, args, nil
}

func (db *DB) GetAllJobs(filter *JobFilter) ([]*Job, error) {
	if filter == nil {
		filter = &JobFilter{}
	}

	l := []*Job{}
	query, args, err := filter.Query(db.Driver)
	if err != nil {
		return l, err
	}
	r, err := db.Query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		j := &Job{}
		var (
			last                                *int64
			this, policy, store, target, tenant NullUUID
			last_task_status                    sql.NullString
		)
		if err = r.Scan(
			&this, &j.Name, &j.Summary, &j.Paused, &j.Schedule, &tenant,
			&j.Policy.Name, &j.Policy.Summary, &policy, &j.Expiry,
			&store, &j.Store.Name, &j.Store.Plugin, &j.Store.Endpoint, &j.Store.Summary,
			&target, &j.Target.Name, &j.Target.Plugin, &j.Target.Endpoint,
			&j.Agent, &last, &last_task_status); err != nil {
			return l, err
		}
		j.UUID = this.UUID
		j.Policy.UUID = policy.UUID
		j.Store.UUID = store.UUID
		j.Target.UUID = target.UUID
		j.TenantUUID = tenant.UUID
		if last == nil {
			j.LastRun = 0
			j.LastTaskStatus = "pending"
		} else {
			j.LastRun = *last
		}

		l = append(l, j)
	}

	return l, nil
}

func (db *DB) GetJob(id uuid.UUID) (*Job, error) {
	r, err := db.Query(`
		SELECT j.uuid, j.name, j.summary, j.paused, j.schedule, j.tenant_uuid,
		       r.name, r.summary, r.uuid, r.expiry,
		       s.uuid, s.name, s.plugin, s.endpoint, s.summary,
		       t.uuid, t.name, t.plugin, t.endpoint, t.agent

			FROM jobs j
				INNER JOIN retention  r  ON  r.uuid = j.retention_uuid
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

	j := &Job{}
	var this, policy, store, target, tenant NullUUID
	if err = r.Scan(
		&this, &j.Name, &j.Summary, &j.Paused, &j.Schedule, &tenant,
		&j.Policy.Name, &j.Policy.Summary, &policy, &j.Expiry,
		&store, &j.Store.Name, &j.Store.Plugin, &j.Store.Endpoint, &j.Store.Summary,
		&target, &j.Target.Name, &j.Target.Plugin, &j.Target.Endpoint,
		&j.Agent); err != nil {
		return nil, err
	}
	j.UUID = this.UUID
	j.Policy.UUID = policy.UUID
	j.Store.UUID = store.UUID
	j.Target.UUID = target.UUID
	j.TenantUUID = tenant.UUID

	return j, nil
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

func (db *DB) CreateJob(job *Job) (*Job, error) {
	job.UUID = uuid.NewRandom()

	err := db.Exec(`
	   INSERT INTO jobs (uuid, tenant_uuid,
	                     name, summary, schedule, paused,
	                     target_uuid, store_uuid, retention_uuid)
	             VALUES (?, ?,
	                     ?, ?, ?, ?,
	                     ?, ?, ?)`,
		job.UUID.String(), job.TenantUUID.String(),
		job.Name, job.Summary, job.Schedule, job.Paused,
		job.TargetUUID.String(), job.StoreUUID.String(), job.PolicyUUID.String())
	if err != nil {
		return nil, err
	}

	return db.GetJob(job.UUID)
}

func (db *DB) UpdateJob(job *Job) error {
	return db.Exec(`
	   UPDATE jobs
	      SET name           = ?,
	          summary        = ?,
	          schedule       = ?,
	          target_uuid    = ?,
	          store_uuid     = ?,
	          retention_uuid = ?
	    WHERE uuid = ?`,
		job.Name, job.Summary, job.Schedule,
		job.TargetUUID.String(), job.StoreUUID.String(), job.PolicyUUID.String(),
		job.UUID.String())
}

func (db *DB) DeleteJob(id uuid.UUID) (bool, error) {
	return true, db.Exec(
		`DELETE FROM jobs WHERE uuid = ?`,
		id.String(),
	)
}

func (db *DB) RescheduleJob(j *Job, t time.Time) error {
	return db.Exec(`UPDATE jobs SET next_run = ? WHERE uuid = ?`, t.Unix(), j.UUID.String())
}

func (j *Job) Reschedule() error {
	var err error
	if j.Spec == nil {
		j.Spec, err = timespec.Parse(j.Schedule)
		if err != nil {
			return err
		}
	}
	next, err := j.Spec.Next(time.Now())
	if err != nil {
		return err
	}
	j.NextRun = next.Unix()
	return nil
}

func (j *Job) Runnable() bool {
	return j.Paused == false && j.NextRun <= time.Now().Unix()
}
