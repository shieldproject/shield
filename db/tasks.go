package db

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jhunt/go-log"
)

const (
	BackupOperation      = "backup"
	RestoreOperation     = "restore"
	AgentStatusOperation = "agent-status"

	PendingStatus   = "pending"
	ScheduledStatus = "scheduled"
	RunningStatus   = "running"
	CanceledStatus  = "canceled"
	FailedStatus    = "failed"
	DoneStatus      = "done"
)

type Task struct {
	UUID           string         `json:"uuid"            mbus:"uuid"`
	Owner          string         `json:"owner"           mbus:"owner"`
	Op             string         `json:"type"            mbus:"op"`
	JobUUID        string         `json:"job_uuid"        mbus:"job_uuid"`
	ArchiveUUID    string         `json:"archive_uuid"    mbus:"archive_uuid"`
	TargetUUID     string         `json:"target_uuid"     mbus:"target_uuid"`
	TargetPlugin   string         `json:"-"`
	TargetEndpoint string         `json:"-"`
	Stream         string         `json:"-"`
	Bucket         string         `json:"-"`
	Status         string         `json:"status"          mbus:"status"`
	RequestedAt    int64          `json:"requested_at"    mbus:"requested_at"`
	StartedAt      int64          `json:"started_at"      mbus:"started_at"`
	StoppedAt      int64          `json:"stopped_at"      mbus:"stopped_at"`
	TimeoutAt      int64          `json:"-"`
	Attempts       int            `json:"-"`
	RestoreKey     string         `json:"-"`
	Agent          string         `json:"-"`
	Log            string         `json:"log"`
	OK             bool           `json:"ok"              mbus:"ok"`
	Notes          string         `json:"notes"           mbus:"notes"`
	Clear          string         `json:"clear"           mbus:"clear"`
	TaskUUIDChan   chan *TaskInfo `json:"-"`
}

type TaskInfo struct {
	Info string //UUID if not Err. Error message if Err.
	Err  bool
}

type TaskFilter struct {
	UUID          string
	ExactMatch    bool
	SkipActive    bool
	SkipInactive  bool
	OnlyRelevant  bool
	ForOp         string
	ForAgent      string
	SkipStopped   bool
	ForTarget     string
	ForStatus     string
	ForArchive    string
	Limit         int
	RequestedAt   int64
	Before        int64
	StartedAfter  *time.Duration
	StoppedAfter  *time.Duration
	StartedBefore *time.Duration
	StoppedBefore *time.Duration
}

func effectively(t time.Time) int64 {
	if t.Unix() <= 0 {
		return time.Now().Unix()
	}
	return t.Unix()
}

func (f *TaskFilter) Query() (string, []interface{}) {
	wheres := []string{"t.uuid = t.uuid"}
	var args []interface{}
	if f.ForStatus != "" {
		wheres = append(wheres, "t.status = ?")
		args = append(args, f.ForStatus)
	} else {
		if f.SkipActive {
			wheres = append(wheres, "t.stopped_at IS NOT NULL")
		} else if f.SkipInactive {
			wheres = append(wheres, "t.stopped_at IS NULL")
		}
	}

	if f.OnlyRelevant {
		wheres = append(wheres, "t.relevant = ?")
		args = append(args, true)
	}

	if f.UUID != "" {
		if f.ExactMatch {
			wheres = append(wheres, "t.uuid::text = ?")
			args = append(args, f.UUID)
		} else {
			wheres = append(wheres, "t.uuid::text LIKE ?")
			args = append(args, PatternPrefix(f.UUID))
		}
	}

	if f.ForArchive != "" {
		wheres = append(wheres, "t.archive_uuid = ?")
		args = append(args, f.ForArchive)
	}

	if f.ForOp != "" {
		wheres = append(wheres, "t.op = ?")
		args = append(args, f.ForOp)
	}

	if f.ForAgent != "" {
		wheres = append(wheres, "t.agent = ?")
		args = append(args, f.ForAgent)
	}

	if f.SkipStopped {
		wheres = append(wheres, "t.stopped_at IS NULL")
	}

	if f.ForTarget != "" {
		wheres = append(wheres, "t.target_uuid = ?")
		args = append(args, f.ForTarget)
	}

	if f.Before > 0 {
		wheres = append(wheres, "t.requested_at < ?")
		args = append(args, f.Before)
	}

	if f.RequestedAt > 0 {
		wheres = append(wheres, "t.requested_at = ?")
		args = append(args, f.RequestedAt)
	}

	if f.StartedAfter != nil {
		wheres = append(wheres, "t.started_at > ?")
		args = append(args, time.Now().Add(-*f.StartedAfter).Unix())
	}

	if f.StoppedAfter != nil {
		wheres = append(wheres, "t.stopped_at > ?")
		args = append(args, time.Now().Add(-*f.StoppedAfter).Unix())
	}

	if f.StartedBefore != nil {
		wheres = append(wheres, "t.started_at < ?")
		args = append(args, time.Now().Add(-*f.StartedBefore).Unix())
	}

	if f.StoppedBefore != nil {
		wheres = append(wheres, "t.stopped_at < ?")
		args = append(args, time.Now().Add(-*f.StoppedBefore).Unix())
	}

	limit := ""
	if f.Limit > 0 {
		limit = " LIMIT ?"
		args = append(args, f.Limit)
	}
	return `
        SELECT t.uuid, t.owner, t.op,
               t.job_uuid, t.archive_uuid,
               t.target_uuid, t.target_plugin, t.target_endpoint,
               t.status, t.requested_at, t.started_at, t.stopped_at, t.timeout_at,
               t.bucket, t.stream, t.restore_key, t.attempts, t.agent, t.log,
               t.ok, t.notes, t.clear

        FROM tasks t

        WHERE ` + strings.Join(wheres, " AND ") + `
        ORDER BY t.requested_at DESC, t.started_at DESC, t.uuid ASC
    ` + limit, args
}

func (db *DB) GetAllTasks(filter *TaskFilter) ([]*Task, error) {
	if filter == nil {
		filter = &TaskFilter{}
	}

	l := []*Task{}
	query, args := filter.Query()
	r, err := db.query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		t := &Task{}

		var (
			started, stopped, deadline *int64
			job, archive, target, log  sql.NullString
		)
		if err = r.Scan(
			&t.UUID, &t.Owner, &t.Op, &job, &archive,
			&target, &t.TargetPlugin, &t.TargetEndpoint,
			&t.Status, &t.RequestedAt, &started, &stopped, &deadline,
			&t.Bucket, &t.Stream, &t.RestoreKey, &t.Attempts, &t.Agent, &log,
			&t.OK, &t.Notes, &t.Clear); err != nil {
			return l, err
		}
		if job.Valid {
			t.JobUUID = job.String
		}
		if archive.Valid {
			t.ArchiveUUID = archive.String
		}
		if target.Valid {
			t.TargetUUID = target.String
		}
		if log.Valid {
			t.Log = log.String
		}
		if started == nil {
			t.StartedAt = 0
		} else {
			t.StartedAt = *started
		}
		if stopped == nil {
			t.StoppedAt = 0
		} else {
			t.StoppedAt = *stopped
		}
		if deadline == nil {
			t.TimeoutAt = 0
		} else {
			t.TimeoutAt = *deadline
		}

		l = append(l, t)
	}

	return l, nil
}

func (db *DB) GetTask(id string) (*Task, error) {
	all, err := db.GetAllTasks(&TaskFilter{UUID: id, ExactMatch: true})
	if err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return nil, nil
	}
	return all[0], nil
}

func (db *DB) ReinsertTask(task *Task) error {
	return db.Exec(`
        INSERT INTO tasks
            (uuid, owner, op,
             job_uuid, archive_uuid, target_uuid,
             status, requested_at, started_at, stopped_at, timeout_at,
             log, attempts, agent
             target_plugin, target_endpoint,
             bucket, stream, restore_key,
             ok, notes, clear)
        VALUES
            (?, ?, ?,
             ?, ?, ?,
             ?, ?, ?, ?, ?,
             ?, ?, ?,
             ?, ?,
             ?, ?, ?,
             ?, ?, ?)`,
		task.UUID, task.Owner, task.Op,
		task.JobUUID, task.ArchiveUUID, task.TargetUUID,
		task.Status, task.RequestedAt, task.StartedAt, task.StoppedAt, task.TimeoutAt,
		task.Log, task.Attempts, task.Agent,
		task.TargetPlugin, task.TargetEndpoint,
		task.Bucket, task.Stream, task.RestoreKey,
		task.OK, task.Notes, task.Clear)
}

func (db *DB) CreateBackupTask(owner string, job *Job) (*Task, error) {
	id := RandomID()
	archive := RandomID()

	err := db.exclusively(func() error {
		/* validate the target */
		if err := db.targetShouldExist(job.TargetUUID); err != nil {
			return fmt.Errorf("unable to create backup task: %s", err)
		}

		/* note: the archive has not yet been created;
		   we record it in the task record so that when the
		   store plugin gives us the details, we know where
		   to put it already. */

		return db.Exec(
			`INSERT INTO tasks
                (uuid, owner, op, job_uuid, status, log, requested_at,
                 bucket, archive_uuid,
                 target_uuid, target_plugin, target_endpoint, restore_key,
                 agent, attempts)
              VALUES
                (?, ?, ?, ?, ?, ?, ?,
                 ?, ?,
                 ?, ?, ?, ?,
                 ?, ?)`,
			id, owner, BackupOperation, job.UUID, PendingStatus, "", time.Now().Unix(),
			job.Bucket, archive,
			job.Target.UUID, job.Target.Plugin, job.Target.Endpoint, "",
			job.Agent, 0)
	})
	if err != nil {
		return nil, err
	}

	task, err := db.GetTask(id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("failed to retrieve newly-inserted task [%s]: not found in database", id)
	}

	db.sendCreateObjectEvent(task, "*")
	return task, nil
}

func (db *DB) SkipBackupTask(owner string, job *Job, msg string) (*Task, error) {
	id := RandomID()
	now := time.Now().Unix()

	err := db.exclusively(func() error {
		/* validate the target */
		if err := db.targetShouldExist(job.TargetUUID); err != nil {
			return fmt.Errorf("unable to create (skipped) backup task: %s", err)
		}

		return db.Exec(
			`INSERT INTO tasks
                (uuid, owner, op, job_uuid, status, log,
                 requested_at, started_at, stopped_at, ok,
                 target_uuid, target_plugin, target_endpoint, restore_key,
                 agent, attempts)
              VALUES
                (?, ?, ?, ?, ?, ?,
                 ?, ?, ?, ?,
                 ?, ?, ?, ?,
                 ?, ?)`,
			id, owner, BackupOperation, job.UUID, CanceledStatus, msg,
			now, now, now, 0,
			job.Target.UUID, job.Target.Plugin, job.Target.Endpoint, "",
			job.Agent, 0)
	})
	if err != nil {
		return nil, err
	}

	task, err := db.GetTask(id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("failed to retrieve newly-inserted task [%s]: not found in database", id)
	}

	db.sendCreateObjectEvent(task, "*")
	return task, nil
}

func (db *DB) CreateRestoreTask(owner string, archive *Archive, target *Target) (*Task, error) {
	endpoint, err := target.ConfigJSON()
	if err != nil {
		return nil, err
	}

	id := RandomID()
	err = db.exclusively(func() error {
		/* validate the archive */
		if err := db.archiveShouldExist(archive.UUID); err != nil {
			return fmt.Errorf("unable to create restore task: %s", err)
		}

		/* validate the target */
		if err := db.targetShouldExist(target.UUID); err != nil {
			return fmt.Errorf("unable to create restore task: %s", err)
		}

		return db.Exec(
			`INSERT INTO tasks
                (uuid, owner, op, archive_uuid, status, log, requested_at,
                 target_uuid, target_plugin, target_endpoint,
                 restore_key, agent, attempts)
              VALUES
                (?, ?, ?, ?, ?, ?, ?,
                 ?, ?, ?,
                 ?, ?, ?)`,
			id, owner, RestoreOperation, archive.UUID, PendingStatus, "", time.Now().Unix(),
			target.UUID, target.Plugin, endpoint,
			archive.StoreKey, target.Agent, 0)
	})
	if err != nil {
		return nil, err
	}

	task, err := db.GetTask(id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("failed to retrieve newly-inserted task [%s]: not found in database", id)
	}

	db.sendCreateObjectEvent(task, "*")
	return task, nil
}

func (db *DB) CreateAgentStatusTask(owner string, agent *Agent) (*Task, error) {
	tasks, err := db.GetAllTasks(&TaskFilter{
		ForOp:       AgentStatusOperation,
		ForAgent:    agent.Address,
		SkipStopped: true,
	})
	if err != nil {
		return nil, err
	}
	if len(tasks) > 0 {
		return tasks[0], nil
	}

	id := RandomID()
	err = db.Exec(`
       INSERT INTO tasks (uuid, op, status, log, requested_at,
                          agent, attempts, owner)

                  VALUES (?, ?, ?, ?, ?,
                          ?, ?, ?)`,
		id, AgentStatusOperation, PendingStatus, "", time.Now().Unix(),
		agent.Address, 0, owner,
	)

	if err != nil {
		return nil, err
	}

	task, err := db.GetTask(id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("failed to retrieve newly-inserted task [%s]: not found in database", id)
	}

	db.sendCreateObjectEvent(task, "admins")
	return task, nil
}

func (db *DB) IsTaskRunnable(task *Task) (bool, error) {
	/* tasks without targets (i.e. purge tasks) are always valid */
	if task.TargetUUID == "" {
		return true, nil
	}

	r, err := db.query(`
        SELECT uuid FROM tasks
          WHERE target_uuid = ? AND status = ? LIMIT 1`, task.TargetUUID, RunningStatus)
	if err != nil {
		return false, err
	}
	defer r.Close()

	if !r.Next() {
		return true, nil
	}
	return false, nil
}

func (db *DB) StartTask(id string, at time.Time) error {
	err := db.Exec(
		`UPDATE tasks SET status = ?, started_at = ? WHERE uuid = ?`,
		RunningStatus, effectively(at), id,
	)
	if err != nil {
		return err
	}

	task, err := db.GetTask(id)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("task '%s' not found", id)
	}

	db.sendTaskStatusUpdateEvent(task, "*")
	return nil
}

func (db *DB) ScheduledTask(id string) error {
	err := db.Exec(
		`UPDATE tasks SET status = ? WHERE uuid = ?`,
		ScheduledStatus, id)
	if err != nil {
		return err
	}

	task, err := db.GetTask(id)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("task '%s' not found", id)
	}

	db.sendTaskStatusUpdateEvent(task, "*")
	return nil
}

func (db *DB) updateTaskStatus(id, status string, at int64, ok int) error {
	err := db.Exec(
		`UPDATE tasks SET status = ?, stopped_at = ?, ok = ? WHERE uuid = ?`,
		status, at, ok, id)
	if err != nil {
		return err
	}

	task, err := db.GetTask(id)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("task '%s' not found", id)
	}

	db.sendTaskStatusUpdateEvent(task, "*")
	return nil
}

func (db *DB) CancelTask(id string, at time.Time) error {
	return db.updateTaskStatus(id, CanceledStatus, effectively(at), 1)
}

func (db *DB) FailTask(id string, at time.Time) error {
	err := db.updateTaskStatus(id, FailedStatus, effectively(at), 0)
	if err != nil {
		return err
	}
	task, err := db.GetTask(id)
	if err != nil {
		return err
	}
	if task.JobUUID != "" {
		err = db.UpdateJobHealth(task.JobUUID, false)
		if err != nil {
			return err
		}
		err = db.UpdateTargetHealth(task.TargetUUID, false)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) CompleteTask(id string, at time.Time) error {
	err := db.updateTaskStatus(id, DoneStatus, effectively(at), 1)
	if err != nil {
		return err
	}
	task, err := db.GetTask(id)
	if err != nil {
		return err
	}
	if task.JobUUID != "" {
		err = db.UpdateJobHealth(task.JobUUID, true)
		if err != nil {
			return err
		}
		err = db.UpdateTargetHealth(task.TargetUUID, true)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) UpdateTaskLog(id string, more string) error {
	err := db.Exec(
		`UPDATE tasks SET log = log || ? WHERE uuid = ?`,
		more, id,
	)
	if err != nil {
		return err
	}

	db.sendTaskLogUpdateEvent(id, more, "*")
	return nil
}

func (db *DB) getTaskArchiveRetention(id string) (int, error) {
	r, err := db.query(`
	         SELECT j.keep_days
	           FROM jobs j
	     INNER JOIN tasks t ON j.uuid = t.job_uuid
	          WHERE t.uuid = ?`, id)
	if err != nil {
		return 0, err
	}
	defer r.Close()

	if !r.Next() {
		return 0, fmt.Errorf("failed to determine expiration for task %s", id)
	}

	var n int
	if err = r.Scan(&n); err != nil {
		return 0, fmt.Errorf("failed to determine expiration for task %s: %s", id, err)
	}
	return n, nil
}

func (db *DB) CreateTaskArchive(id, archive_id, key string, at time.Time, archive_size int64) (string, error) {
	if key == "" {
		return "", fmt.Errorf("cannot create an archive without a store_key")
	}

	// determine how long we need to keep this specific archive for
	n, err := db.getTaskArchiveRetention(id)
	if err != nil {
		return "", err
	}

	// insert an archive with all proper references, expiration, etc.
	archive, err := db.createArchiveFromTask(id, Archive{
		UUID:      archive_id,
		StoreKey:  key,
		TakenAt:   effectively(at),
		ExpiresAt: at.Add(time.Duration(n*24) * time.Hour).Unix(),
		Size:      archive_size,
	})
	if err != nil {
		log.Errorf("failed to insert archive with UUID %s into database: %s", archive.UUID, err)
		return "", err
	}
	db.sendCreateObjectEvent(archive, "*")

	// and finally, associate task -> archive
	return archive_id, db.Exec(
		`UPDATE tasks SET archive_uuid = ? WHERE uuid = ?`,
		archive_id, id)
}

type TaskAnnotation struct {
	Disposition string
	Notes       string
	Clear       string
}

func (db *DB) AnnotateTargetTask(target, id string, t *TaskAnnotation) error {
	updates := []string{}
	args := []interface{}{}

	updates = append(updates, "ok = ?")
	args = append(args, t.Disposition == "ok")

	if t.Notes != "" {
		updates = append(updates, "notes = ?")
		args = append(args, t.Notes)
	}

	if t.Clear != "" {
		updates = append(updates, "clear = ?")
		args = append(args, t.Clear)
	}

	args = append(args, target, id)
	return db.Exec(
		`UPDATE tasks SET `+strings.Join(updates, ", ")+
			`WHERE target_uuid = ? AND uuid = ?`, args...)
}

func (db *DB) MarkTasksIrrelevant() error {
	err := db.Exec(
		`UPDATE tasks SET relevant = 0
      WHERE relevant = 1
        AND clear = 'immediate'`)

	if err != nil {
		return err
	}

	err = db.Exec(
		`UPDATE tasks SET relevant = 0
      WHERE relevant = 1 AND clear = 'normal'
        AND uuid IN (
          SELECT tasks.uuid FROM tasks
            INNER JOIN jobs ON jobs.uuid = tasks.job_uuid
                 WHERE jobs.keep_days * 86400 + tasks.started_at < ?)`, time.Now().Unix())

	if err != nil {
		return err
	}

	err = db.Exec(
		`UPDATE tasks SET relevant = 1
      WHERE relevant = 0 AND clear = 'manual'`)
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) RedactTaskLog(task *Task) {
	re := regexp.MustCompile("(<redacted>.*?</redacted>)")
	matches := re.FindAllStringSubmatch(task.Log, -1)
	for _, match := range matches {
		task.Log = strings.Replace(task.Log, match[1], "«REDACTED»", -1)
	}
}

func (db *DB) RedactAllTaskLogs(tasks []*Task) {
	for _, task := range tasks {
		db.RedactTaskLog(task)
	}
}

//UnscheduleAllTasks takes all tasks which are in the scheduled state and puts
//them back in a pending state.
func (db *DB) UnscheduleAllTasks() error {
	return db.Exec(`UPDATE tasks SET status = 'pending' WHERE status = 'scheduled'`)
}

func (db *DB) archiveShouldExist(uuid string) error {
	if ok, err := db.Exists(`SELECT uuid FROM archives WHERE uuid = ?`, uuid); err != nil {
		return fmt.Errorf("unable to look up archive [%s]: %s", uuid, err)
	} else if !ok {
		return fmt.Errorf("archive [%s] does not exist", uuid)
	}
	return nil
}

func (db *DB) TruncateTaskLogs(age int) error {
	return db.Exec(`
       UPDATE tasks SET log = '(log truncated)'
                  WHERE stopped_at IS NOT NULL
                    AND stopped_at < ?`,
		(int)(time.Now().Unix())-age)
}
