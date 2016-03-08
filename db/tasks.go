package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pborman/uuid"

	. "github.com/starkandwayne/shield/timestamp"
)

type Operation int

const (
	BACKUP Operation = iota
	RESTORE
	PURGE
)

func (o Operation) String() string {
	switch o {
	case BACKUP:
		return "backup"
	case RESTORE:
		return "restore"
	case PURGE:
		return "purge"
	default:
		return "UNKNOWN"
	}
}

type Status int

const (
	PENDING Status = iota
	RUNNING
	CANCELED
	DONE
)

type Task struct {
	UUID           uuid.UUID `json:"uuid"`
	Owner          string    `json:"owner"`
	Op             string    `json:"type"`
	JobUUID        uuid.UUID `json:"job_uuid"`
	ArchiveUUID    uuid.UUID `json:"archive_uuid"`
	StoreUUID      uuid.UUID `json:"-"`
	StorePlugin    string    `json:"-"`
	StoreEndpoint  string    `json:"-"`
	TargetUUID     uuid.UUID `json:"-"`
	TargetPlugin   string    `json:"-"`
	TargetEndpoint string    `json:"-"`
	Status         string    `json:"status"`
	StartedAt      Timestamp `json:"started_at"`
	StoppedAt      Timestamp `json:"stopped_at"`
	TimeoutAt      Timestamp `json:"-"`
	Attempts       int       `json:"-"`
	RestoreKey     string    `json:"-"`
	Agent          string    `json:"-"`
	Log            string    `json:"log"`
}

type TaskFilter struct {
	UUID         string
	SkipActive   bool
	SkipInactive bool
	ForStatus    string
	Limit        string
	// FIXME: add options for store / target
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
	n := 1
	if f.ForStatus != "" {
		wheres = append(wheres, fmt.Sprintf("status = $%d", n))
		args = append(args, f.ForStatus)
		n++
	} else {
		if f.SkipActive {
			wheres = append(wheres, "stopped_at IS NOT NULL")
		} else if f.SkipInactive {
			wheres = append(wheres, "stopped_at IS NULL")
		}
	}

	if f.UUID != "" {
		wheres = append(wheres, fmt.Sprintf("uuid = $%d", n))
		args = append(args, f.UUID)
		n++
	}

	limit := ""
	if f.Limit != "" {
		limit = fmt.Sprintf(" LIMIT $%d", n)
		args = append(args, f.Limit)
		n++
	}
	return `
		SELECT t.uuid, t.owner, t.op, t.job_uuid, t.archive_uuid,
		       t.store_uuid,  t.store_plugin,  t.store_endpoint,
		       t.target_uuid, t.target_plugin, t.target_endpoint,
		       t.status, t.started_at, t.stopped_at, t.timeout_at,
		       t.attempts, t.agent, t.log

		FROM tasks t

		WHERE ` + strings.Join(wheres, " AND ") + `
		ORDER BY t.started_at DESC, t.uuid ASC
	` + limit, args
}

func (db *DB) GetAllTasks(filter *TaskFilter) ([]*Task, error) {
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
			&ann.Attempts, &ann.Agent, &log); err != nil {
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

func (db *DB) CreateBackupTask(owner string, job uuid.UUID) (uuid.UUID, error) {
	id := uuid.NewRandom()
	return id, db.Exec(
		`INSERT INTO tasks (uuid, owner, op, job_uuid, status, log, requested_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id.String(), owner, "backup", job.String(), "pending", "", time.Now().Unix(),
	)
}

func (db *DB) CreateRestoreTask(owner string, archive, target uuid.UUID) (uuid.UUID, error) {
	id := uuid.NewRandom()
	return id, db.Exec(
		`INSERT INTO tasks (uuid, owner, op, archive_uuid, target_uuid, status, log, requested_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		id.String(), owner, "restore", archive.String(), target.String(), "pending", "", time.Now().Unix(),
	)
}

func (db *DB) CreatePurgeTask(owner string, archive *Archive) (uuid.UUID, error) {
	id := uuid.NewRandom()
	return id, db.Exec(
		`INSERT INTO tasks (uuid, owner, op, archive_uuid, store_uuid, status, log, requested_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		id.String(), owner, "purge", archive.UUID.String(), archive.StoreUUID.String(), "pending", "", time.Now().Unix(),
	)
}

func (db *DB) StartTask(id uuid.UUID, effective time.Time) error {
	validtime := ValidateEffectiveUnix(effective)
	return db.Exec(
		`UPDATE tasks SET status = $1, started_at = $2 WHERE uuid = $3`,
		"running", validtime, id.String(),
	)
}

func (db *DB) updateTaskStatus(id uuid.UUID, status string, effective int64) error {
	return db.Exec(
		`UPDATE tasks SET status = $1, stopped_at = $2 WHERE uuid = $3`,
		status, effective, id.String(),
	)
}
func (db *DB) CancelTask(id uuid.UUID, effective time.Time) error {
	validtime := ValidateEffectiveUnix(effective)
	return db.updateTaskStatus(id, "canceled", validtime)
}

func (db *DB) FailTask(id uuid.UUID, effective time.Time) error {
	validtime := ValidateEffectiveUnix(effective)
	return db.updateTaskStatus(id, "failed", validtime)
}

func (db *DB) CompleteTask(id uuid.UUID, effective time.Time) error {
	validtime := ValidateEffectiveUnix(effective)
	return db.updateTaskStatus(id, "done", validtime)
}

func (db *DB) UpdateTaskLog(id uuid.UUID, more string) error {
	return db.Exec(
		`UPDATE tasks SET log = log || $1 WHERE uuid = $2`,
		more, id.String(),
	)
}

func (db *DB) CreateTaskArchive(id uuid.UUID, key string, effective time.Time) (uuid.UUID, error) {
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
			WHERE t.uuid = $1`,
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
	archive_id := uuid.NewRandom()
	validtime := ValidateEffectiveUnix(effective)
	err = db.Exec(
		`INSERT INTO archives
			(uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, notes)
			SELECT $1, t.uuid, s.uuid, $2, $3, $4, ''
				FROM tasks
					INNER JOIN jobs    j     ON j.uuid = tasks.job_uuid
					INNER JOIN targets t     ON t.uuid = j.target_uuid
					INNER JOIN stores  s     ON s.uuid = j.store_uuid
				WHERE tasks.uuid = $5`,
		archive_id.String(), key, validtime, effective.Add(time.Duration(expiry)*time.Second).Unix(), id.String(),
	)
	if err != nil {
		return nil, err
	}

	// and finally, associate task -> archive
	return archive_id, db.Exec(
		`UPDATE tasks SET archive_uuid = $1 WHERE uuid = $2`,
		archive_id.String(), id.String(),
	)
}
