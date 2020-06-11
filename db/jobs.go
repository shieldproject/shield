package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/shieldproject/shield/timespec"
)

type Job struct {
	TenantUUID string `json:"-" mbus:"tenant_uuid"`
	TargetUUID string `json:"-" mbus:"target_uuid"`

	UUID     string `json:"uuid"      mbus:"uuid"`
	Name     string `json:"name"      mbus:"name"`
	Summary  string `json:"summary"   mbus:"summary"`
	Bucket   string `json:"bucket"    mbus:"bucket"`
	KeepN    int    `json:"keep_n"    mbus:"keep_n"`
	KeepDays int    `json:"keep_days" mbus:"keep_days"`
	Schedule string `json:"schedule"  mbus:"schedule"`
	Paused   bool   `json:"paused"    mbus:"paused"`
	FixedKey bool   `json:"fixed_key" mbus:"fixed_key"`

	Target struct {
		UUID        string `json:"uuid"`
		Name        string `json:"name"`
		Agent       string `json:"agent"`
		Plugin      string `json:"plugin"`
		Compression string `json:"compression"`

		Endpoint string                 `json:"endpoint,omitempty"`
		Config   map[string]interface{} `json:"config,omitempty"`
	} `json:"target"`

	Agent string `json:"agent"`

	Healthy        bool   `json:"healthy" mbus:"healthy"`
	LastRun        int64  `json:"last_run"`
	LastTaskStatus string `json:"last_task_status"`

	Spec    *timespec.Spec `json:"-"`
	NextRun int64          `json:"-"`
}

type JobFilter struct {
	UUID         string
	SkipPaused   bool
	SkipUnpaused bool

	Overdue bool

	SearchName string

	ForTenant  string
	ForTarget  string
	ForBucket  string
	ExactMatch bool
}

func (f *JobFilter) Query() (string, []interface{}) {
	wheres := []string{}
	args := []interface{}{}

	if f.UUID != "" {
		if f.ExactMatch {
			wheres = []string{"j.uuid::text = ?"}
			args = append(args, f.UUID)
		} else {
			wheres = []string{"j.uuid::text LIKE ?"}
			args = append(args, PatternPrefix(f.UUID))
		}
	}

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

	if len(wheres) == 0 {
		wheres = []string{"true"}
	} else if len(wheres) > 1 {
		wheres = []string{strings.Join(wheres, " OR ")}
	}

	if f.ForTenant != "" {
		wheres = append(wheres, "j.tenant_uuid = ?")
		args = append(args, f.ForTenant)
	}
	if f.ForTarget != "" {
		wheres = append(wheres, "j.target_uuid = ?")
		args = append(args, f.ForTarget)
	}
	if f.ForBucket != "" {
		wheres = append(wheres, "j.bucket = ?")
		args = append(args, f.ForBucket)
	}
	if f.SkipPaused || f.SkipUnpaused {
		wheres = append(wheres, "j.paused = ?")
		if f.SkipPaused {
			args = append(args, false)
		} else {
			args = append(args, true)
		}
	}
	if f.Overdue {
		wheres = append(wheres, "j.next_run <= ?")
		args = append(args, time.Now().Unix())
	}

	return `
	   WITH recent_tasks AS (
	           SELECT DISTINCT ON (job_uuid) uuid, job_uuid, started_at, status
	             FROM tasks
	            WHERE stopped_at IS NOT NULL
	         ORDER BY job_uuid ASC, started_at DESC
	        )

	   SELECT j.uuid, j.name, j.summary, j.paused, j.schedule, j.bucket,
	          j.tenant_uuid, j.fixed_key, j.healthy, j.keep_n, j.keep_days,
	          t.uuid, t.name, t.plugin, t.endpoint, t.agent, t.compression,
	          k.started_at, k.status

	     FROM jobs j
	          INNER JOIN targets      t  ON  t.uuid = j.target_uuid
	          LEFT  JOIN recent_tasks k  ON  j.uuid = k.job_uuid

	    WHERE ` + strings.Join(wheres, " AND ") + `
	 ORDER BY j.name, j.uuid ASC`, args
}

func (db *DB) GetAllJobs(filter *JobFilter) ([]*Job, error) {
	if filter == nil {
		filter = &JobFilter{}
	}

	l := []*Job{}
	query, args := filter.Query()
	r, err := db.query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		j := &Job{}

		var (
			last   *int64
			status sql.NullString
		)
		if err = r.Scan(
			&j.UUID, &j.Name, &j.Summary, &j.Paused, &j.Schedule, &j.Bucket,
			&j.TenantUUID, &j.FixedKey, &j.Healthy, &j.KeepN, &j.KeepDays,
			&j.Target.UUID, &j.Target.Name, &j.Target.Plugin, &j.Target.Endpoint,
			&j.Agent, &j.Target.Compression, &last, &status); err != nil {
			return l, err
		}
		if last != nil {
			j.LastRun = *last
		}
		if status.Valid {
			j.LastTaskStatus = status.String
		}

		j.TargetUUID = j.Target.UUID
		l = append(l, j)
	}

	return l, nil
}

func (db *DB) GetJob(id string) (*Job, error) {
	all, err := db.GetAllJobs(&JobFilter{UUID: id})
	if err != nil {
		return nil, err
	}
	if len(all) > 0 {
		return all[0], nil
	}
	return nil, nil
}

func (db *DB) PauseOrUnpauseJob(id string, pause bool) (bool, error) {
	n, err := db.Count(`SELECT uuid FROM jobs WHERE uuid = ? AND paused = ?`, id, !pause)
	if n == 0 || err != nil {
		return false, err
	}
	err = db.Exec(`UPDATE jobs SET paused = ? WHERE uuid = ? AND paused = ?`, pause, id, !pause)
	if err != nil {
		return true, err
	}
	job, err := db.GetJob(id)
	if err != nil {
		return true, err
	}
	if job == nil {
		return true, fmt.Errorf("unable to update job [%s]: not found in database", id)
	}

	db.sendUpdateObjectEvent(job, "tenant:"+job.TenantUUID)
	return true, nil
}

func (db *DB) PauseJob(id string) (bool, error) {
	return db.PauseOrUnpauseJob(id, true)
}

func (db *DB) UnpauseJob(id string) (bool, error) {
	return db.PauseOrUnpauseJob(id, false)
}

func (db *DB) CreateJob(job *Job) (*Job, error) {
	job.UUID = RandomID()
	job.Healthy = true

	err := db.exclusively(func() error {
		/* validate the tenant */
		if err := db.tenantShouldExist(job.TenantUUID); err != nil {
			return fmt.Errorf("unable to create job: %s", err)
		}

		/* validate the target */
		if err := db.targetShouldExist(job.TargetUUID); err != nil {
			return fmt.Errorf("unable to create job: %s", err)
		}

		return db.Exec(`
		   INSERT INTO jobs (uuid, tenant_uuid,
		                     name, summary, schedule, keep_n, keep_days, paused,
		                     target_uuid, bucket, fixed_key, healthy)
		             VALUES (?, ?,
		                     ?, ?, ?, ?, ?, ?,
		                     ?, ?, ?, ?)`,
			job.UUID, job.TenantUUID,
			job.Name, job.Summary, job.Schedule, job.KeepN, job.KeepDays, job.Paused,
			job.TargetUUID, job.Bucket, job.FixedKey, job.Healthy)
	})
	if err != nil {
		return nil, err
	}

	job, err = db.GetJob(job.UUID)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, fmt.Errorf("failed to retrieve newly-inserted job [%s]: not found in database", job.UUID)
	}

	db.sendCreateObjectEvent(job, "tenant:"+job.TenantUUID)
	return job, nil
}

func (db *DB) UpdateJob(job *Job) error {
	err := db.exclusively(func() error {
		/* validate the target */
		if ok, err := db.Exists(`SELECT uuid FROM targets WHERE uuid = ?`, job.TargetUUID); err != nil {
			return fmt.Errorf("unable to validate existence of target with UUID [%s]: %s", job.TargetUUID, err)
		} else if !ok {
			return fmt.Errorf("unable to set job target to [%s]: no such target in database", job.TargetUUID)
		}

		return db.Exec(`
		   UPDATE jobs
		      SET name           = ?,
		          summary        = ?,
		          schedule       = ?,
		          keep_n         = ?,
		          keep_days      = ?,
		          target_uuid    = ?,
		          bucket         = ?,
		          fixed_key      = ?
		    WHERE uuid = ?`,
			job.Name, job.Summary, job.Schedule, job.KeepN, job.KeepDays,
			job.TargetUUID, job.Bucket, job.FixedKey,
			job.UUID)
	})
	if err != nil {
		return err
	}

	update, err := db.GetJob(job.UUID)
	if err != nil {
		return err
	}
	if update == nil {
		return fmt.Errorf("unable to retrieve job %s after update", job.UUID)
	}

	db.sendUpdateObjectEvent(update, "tenant:"+update.TenantUUID)
	return nil
}

func (db *DB) UpdateJobHealth(id string, status bool) error {
	job, err := db.GetJob(id)
	if err != nil {
		return fmt.Errorf("error when finding job with uuid `%s' to update health: %s", id, err)
	}
	if job == nil {
		return fmt.Errorf("no job with uuid `%s' was found to update health", id)
	}
	err = db.Exec(`
        UPDATE jobs
            SET healthy = ?
        WHERE uuid = ?`,
		status,
		job.UUID)
	if err != nil {
		return err
	}

	db.sendHealthUpdateEvent(job, "tenant:"+job.TenantUUID)
	return nil
}

func (db *DB) DeleteJob(id string) (bool, error) {
	job, err := db.GetJob(id)
	if err != nil {
		return false, err
	}

	if job == nil {
		/* already deleted */
		return true, nil
	}

	err = db.Exec(`DELETE FROM jobs WHERE uuid = ?`, job.UUID)
	if err != nil {
		return false, err
	}

	db.sendDeleteObjectEvent(job, "tenant:"+job.TenantUUID)

	jobs, err := db.GetAllJobs(&JobFilter{ForTarget: job.TargetUUID})
	if err != nil {
		return false, nil
	}
	// if target has no jobs, it the target should be healthy
	if len(jobs) == 0 {
		db.UpdateTargetHealth(job.TargetUUID, true)
	}

	return true, nil
}

func (db *DB) RescheduleJob(j *Job, t time.Time) error {
	/* note: this update does not require a message bus notification */
	return db.Exec(`UPDATE jobs SET next_run = ? WHERE uuid = ?`, t.Unix(), j.UUID)
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
	return !j.Paused && j.NextRun <= time.Now().Unix()
}
