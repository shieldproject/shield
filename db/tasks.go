package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pborman/uuid"

	. "github.com/starkandwayne/goutils/timestamp"
)

const (
	BackupOperation  = "backup"
	RestoreOperation = "restore"
	PurgeOperation   = "purge"

	PendingStatus   = "pending"
	ScheduledStatus = "scheduled"
	RunningStatus   = "running"
	CanceledStatus  = "canceled"
	FailedStatus    = "failed"
	DoneStatus      = "done"
)

type Task struct {
	UUID           uuid.UUID      `json:"uuid"`
	Owner          string         `json:"owner"`
	Op             string         `json:"type"`
	JobUUID        uuid.UUID      `json:"job_uuid"`
	ArchiveUUID    uuid.UUID      `json:"archive_uuid"`
	StoreUUID      uuid.UUID      `json:"-"`
	StorePlugin    string         `json:"-"`
	StoreEndpoint  string         `json:"-"`
	TargetUUID     uuid.UUID      `json:"-"`
	TargetPlugin   string         `json:"-"`
	TargetEndpoint string         `json:"-"`
	Status         string         `json:"status"`
	StartedAt      Timestamp      `json:"started_at"`
	StoppedAt      Timestamp      `json:"stopped_at"`
	TimeoutAt      Timestamp      `json:"-"`
	Attempts       int            `json:"-"`
	RestoreKey     string         `json:"-"`
	Agent          string         `json:"-"`
	Log            string         `json:"log"`
	OK             bool           `json:"ok"`
	Notes          string         `json:"notes"`
	Clear          string         `json:"clear"`
	TaskUUIDChan   chan *TaskInfo `json:"-"`
}

type TaskInfo struct {
	Info string //UUID if not Err. Error message if Err.
	Err  bool
}

type TaskFilter struct {
	UUID         string
	SkipActive   bool
	SkipInactive bool
	OnlyRelevant bool
	ForOp        string
	ForTarget    string
	ForStatus    string
	ForArchive   string
	Limit        string
	// FIXME: add options for store
}

func ValidateEffectiveUnix(effective time.Time) int64 {
	if effective.Unix() <= 0 {
		return time.Now().Unix()
	}
	return effective.Unix()
}

func (f *TaskFilter) Query() (string, []interface{}) {
	wheres := []string{"t.uuid = t.uuid"}
	var args []interface{}
	if f.ForStatus != "" {
		wheres = append(wheres, "status = ?")
		args = append(args, f.ForStatus)
	} else {
		if f.SkipActive {
			wheres = append(wheres, "stopped_at IS NOT NULL")
		} else if f.SkipInactive {
			wheres = append(wheres, "stopped_at IS NULL")
		}
	}

	if f.OnlyRelevant {
		wheres = append(wheres, "relevant = ?")
		args = append(args, true)
	}

	if f.UUID != "" {
		wheres = append(wheres, "uuid = ?")
		args = append(args, f.UUID)
	}

	if f.ForArchive != "" {
		wheres = append(wheres, "archive_uuid = ?")
		args = append(args, f.ForArchive)
	}

	if f.ForOp != "" {
		wheres = append(wheres, "op = ?")
		args = append(args, f.ForOp)
	}

	if f.ForTarget != "" {
		wheres = append(wheres, "target_uuid = ?")
		args = append(args, f.ForTarget)
	}

	limit := ""
	if f.Limit != "" {
		limit = " LIMIT ?"
		args = append(args, f.Limit)
	}
	return `
		SELECT t.uuid, t.owner, t.op, t.job_uuid, t.archive_uuid,
		       t.store_uuid,  t.store_plugin,  t.store_endpoint,
		       t.target_uuid, t.target_plugin, t.target_endpoint,
		       t.status, t.started_at, t.stopped_at, t.timeout_at,
		       t.restore_key, t.attempts, t.agent, t.log,
		       t.ok, t.notes, t.clear

		FROM tasks t

		WHERE ` + strings.Join(wheres, " AND ") + `
		ORDER BY t.started_at DESC, t.uuid ASC
	` + limit, args
}

func (db *DB) GetAllTasks(filter *TaskFilter) ([]*Task, error) {
	if filter == nil {
		filter = &TaskFilter{}
	}

	l := []*Task{}
	query, args := filter.Query()
	r, err := db.Query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		ann := &Task{}
		var (
			log                               sql.NullString
			this, archive, job, store, target NullUUID
			started, stopped, deadline        *int64
		)
		if err = r.Scan(
			&this, &ann.Owner, &ann.Op, &job, &archive,
			&store, &ann.StorePlugin, &ann.StoreEndpoint,
			&target, &ann.TargetPlugin, &ann.TargetEndpoint,
			&ann.Status, &started, &stopped, &deadline,
			&ann.RestoreKey, &ann.Attempts, &ann.Agent, &log,
			&ann.OK, &ann.Notes, &ann.Clear); err != nil {
			return l, err
		}
		ann.UUID = this.UUID

		if job.Valid {
			ann.JobUUID = job.UUID
		}
		if archive.Valid {
			ann.ArchiveUUID = archive.UUID
		}
		if store.Valid {
			ann.StoreUUID = store.UUID
		}
		if target.Valid {
			ann.TargetUUID = target.UUID
		}
		if log.Valid {
			ann.Log = log.String
		}
		if started != nil {
			ann.StartedAt = parseEpochTime(*started)
		}
		if stopped != nil {
			ann.StoppedAt = parseEpochTime(*stopped)
		}
		if deadline != nil {
			ann.TimeoutAt = parseEpochTime(*deadline)
		}

		l = append(l, ann)
	}

	return l, nil
}

func (db *DB) GetTask(id uuid.UUID) (*Task, error) {
	filter := TaskFilter{UUID: id.String()}
	r, err := db.GetAllTasks(&filter)
	if err != nil {
		return nil, err
	}
	if len(r) == 0 {
		return nil, nil
	}
	return r[0], nil
}

func (db *DB) CreateBackupTask(owner string, job *Job) (*Task, error) {
	id := uuid.NewRandom()
	err := db.Exec(
		`INSERT INTO tasks
		    (uuid, owner, op, job_uuid, status, log, requested_at,
		     store_uuid, store_plugin, store_endpoint,
		     target_uuid, target_plugin, target_endpoint, restore_key, agent, attempts)
		  VALUES
		    (?, ?, ?, ?, ?, ?, ?,
		     ?, ?, ?,
		     ?, ?, ?, ?, ?, ?)`,
		id.String(), owner, BackupOperation, job.UUID.String(), PendingStatus, "", time.Now().Unix(),
		job.StoreUUID.String(), job.StorePlugin, job.StoreEndpoint,
		job.TargetUUID.String(), job.TargetPlugin, job.TargetEndpoint, "", job.Agent, 0,
	)

	if err != nil {
		return nil, err
	}
	return db.GetTask(id)
}

func (db *DB) CreateRestoreTask(owner string, archive *Archive, target *Target) (*Task, error) {
	id := uuid.NewRandom()
	err := db.Exec(
		`INSERT INTO tasks
		    (uuid, owner, op, archive_uuid, status, log, requested_at,
		     store_uuid, store_plugin, store_endpoint,
		     target_uuid, target_plugin, target_endpoint,
		     restore_key, agent, attempts)
		  VALUES
		    (?, ?, ?, ?, ?, ?, ?,
		     ?, ?, ?,
		     ?, ?, ?,
		     ?, ?, ?)`,
		id.String(), owner, RestoreOperation, archive.UUID.String(), PendingStatus, "", time.Now().Unix(),
		archive.StoreUUID.String(), archive.StorePlugin, archive.StoreEndpoint,
		target.UUID.String(), target.Plugin, target.Endpoint,
		archive.StoreKey, target.Agent, 0,
	)

	if err != nil {
		return nil, err
	}
	return db.GetTask(id)
}

func (db *DB) CreatePurgeTask(owner string, archive *Archive, agent string) (*Task, error) {
	id := uuid.NewRandom()
	err := db.Exec(
		`INSERT INTO tasks
		    (uuid, owner, op, archive_uuid, status, log, requested_at,
		     store_uuid, store_plugin, store_endpoint,
		     target_plugin, target_endpoint,
		     restore_key, agent, attempts)
		  VALUES
		    (?, ?, ?, ?, ?, ?, ?,
		     ?, ?, ?,
		     ?, ?,
		     ?, ?, ?)`,
		id.String(), owner, PurgeOperation, archive.UUID.String(), PendingStatus, "", time.Now().Unix(),
		archive.StoreUUID.String(), archive.StorePlugin, archive.StoreEndpoint,
		"", "",
		archive.StoreKey, agent, 0,
	)

	if err != nil {
		return nil, err
	}
	return db.GetTask(id)
}

func (db *DB) IsTaskRunnable(task *Task) (bool, error) {
	/* tasks without targets (i.e. purge tasks) are always valid */
	if task.TargetUUID.String() == "" {
		return true, nil
	}
	r, err := db.Query(`
		SELECT uuid FROM tasks
		  WHERE target_uuid = ? AND status = ? LIMIT 1`, task.TargetUUID.String(), RunningStatus)
	if err != nil {
		return false, err
	}
	defer r.Close()

	if !r.Next() {
		return true, nil
	}
	return false, nil
}

func (db *DB) StartTask(id uuid.UUID, effective time.Time) error {
	validtime := ValidateEffectiveUnix(effective)
	return db.Exec(
		`UPDATE tasks SET status = ?, started_at = ? WHERE uuid = ?`,
		RunningStatus, validtime, id.String(),
	)
}

func (db *DB) ScheduledTask(id uuid.UUID) error {
	return db.Exec(
		`UPDATE tasks SET status = ? WHERE uuid = ?`,
		ScheduledStatus, id.String())
}

func (db *DB) updateTaskStatus(id uuid.UUID, status string, effective int64, ok int) error {
	return db.Exec(
		`UPDATE tasks SET status = ?, stopped_at = ?, ok = ? WHERE uuid = ?`,
		status, effective, ok, id.String())
}
func (db *DB) CancelTask(id uuid.UUID, effective time.Time) error {
	validtime := ValidateEffectiveUnix(effective)
	return db.updateTaskStatus(id, CanceledStatus, validtime, 1)
}

func (db *DB) FailTask(id uuid.UUID, effective time.Time) error {
	validtime := ValidateEffectiveUnix(effective)
	return db.updateTaskStatus(id, FailedStatus, validtime, 0)
}

func (db *DB) CompleteTask(id uuid.UUID, effective time.Time) error {
	validtime := ValidateEffectiveUnix(effective)
	return db.updateTaskStatus(id, DoneStatus, validtime, 1)
}

func (db *DB) UpdateTaskLog(id uuid.UUID, more string) error {
	return db.Exec(
		`UPDATE tasks SET log = log || ? WHERE uuid = ?`,
		more, id.String(),
	)
}

func (db *DB) CreateTaskArchive(id uuid.UUID, archive_id uuid.UUID, key string, effective time.Time, encryptionType string) (uuid.UUID, error) {
	// fail on empty store_key, as '' seems to satisfy the NOT NULL constraint in postgres
	if key == "" {
		return nil, fmt.Errorf("cannot create an archive without a store_key")
	}
	// determine how long we need to keep this specific archive for
	r, err := db.Query(
		`SELECT r.expiry
			FROM retention r
				INNER JOIN jobs  j    ON r.uuid = j.retention_uuid
				INNER JOIN tasks t    ON j.uuid = t.job_uuid
			WHERE t.uuid = ?`,
		id.String(),
	)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, fmt.Errorf("failed to determine expiration for task %s", id)
	}

	var expiry int
	if err := r.Scan(&expiry); err != nil {
		return nil, err
	}
	r.Close()

	// insert an archive with all proper references, expiration, etc.
	//archive_id := uuid.NewRandom()
	validtime := ValidateEffectiveUnix(effective)
	err = db.Exec(
		`INSERT INTO archives
			(uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, notes, status, purge_reason, job, encryption_type)
			SELECT ?, t.uuid, s.uuid, ?, ?, ?, '', ?, '', j.Name, ?
				FROM tasks
					INNER JOIN jobs    j     ON j.uuid = tasks.job_uuid
					INNER JOIN targets t     ON t.uuid = j.target_uuid
					INNER JOIN stores  s     ON s.uuid = j.store_uuid
				WHERE tasks.uuid = ?`,
		archive_id.String(), key, validtime, effective.Add(time.Duration(expiry)*time.Second).Unix(), "valid", encryptionType, id.String(),
	)
	if err != nil {
		return nil, err
	}

	// and finally, associate task -> archive
	return archive_id, db.Exec(
		`UPDATE tasks SET archive_uuid = ? WHERE uuid = ?`,
		archive_id.String(), id.String(),
	)
}

type TaskAnnotation struct {
	Disposition string
	Notes       string
	Clear       string
}

func (db *DB) AnnotateTargetTask(target uuid.UUID, id string, ann *TaskAnnotation) error {
	updates := []string{}
	args := []interface{}{}

	updates = append(updates, "ok = ?")
	args = append(args, ann.Disposition == "ok")

	if ann.Notes != "" {
		updates = append(updates, "notes = ?")
		args = append(args, ann.Notes)
	}

	if ann.Clear != "" {
		updates = append(updates, "clear = ?")
		args = append(args, ann.Clear)
	}

	args = append(args, target.String(), id)
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
		        INNER JOIN jobs      ON jobs.uuid = tasks.job_uuid
		        INNER JOIN retention ON retention.uuid = jobs.retention_uuid
		             WHERE retention.expiry + tasks.started_at < ?`, time.Now().Unix())

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
