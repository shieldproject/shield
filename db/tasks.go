package db

import (
	"fmt"
	"github.com/pborman/uuid"
	"strings"
	"time"
)

type AnnotatedTask struct {
	UUID        string `json:"uuid"`
	Owner       string `json:"owner"`
	Op          string `json:"type"`
	JobUUID     string `json:"job_uuid"`
	ArchiveUUID string `json:"archive_uuid"`
	Status      string `json:"status"`
	StartedAt   string `json:"started_at"`
	StoppedAt   string `json:"stopped_at"`
	Log         string `json:"log"`
}

type TaskFilter struct {
	ForStatus string
}

func (f *TaskFilter) Args() []interface{} {
	var args []interface{}
	if f.ForStatus != "" {
		args = append(args, f.ForStatus)
	}
	return args
}

func (f *TaskFilter) Query() string {
	wheres := []string{"1"}
	if f.ForStatus != "" {
		wheres = append(wheres, "status = ?")
	}
	return `
		SELECT t.uuid, t.owner, t.op, j.uuid, a.uuid,
		       t.status, t.started_at, t.stopped_at, t.log

		FROM tasks t
			INNER JOIN jobs     j    ON j.uuid = t.job_uuid
			LEFT  JOIN archives a    ON a.uuid = t.archive_uuid

		WHERE ` + strings.Join(wheres, " AND ") + `
		ORDER BY t.started_at DESC, t.uuid ASC
	`
}

func (db *DB) GetAllAnnotatedTasks(filter *TaskFilter) ([]*AnnotatedTask, error) {
	l := []*AnnotatedTask{}
	r, err := db.Query(filter.Query(), filter.Args()...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		ann := &AnnotatedTask{}

		var archive, started, stopped []byte
		if err = r.Scan(
			&ann.UUID, &ann.Owner, &ann.Op, &ann.JobUUID, &archive,
			&ann.Status, &started, &stopped, &ann.Log); err != nil {
			return l, err
		}

		if archive != nil {
			ann.ArchiveUUID = string(archive)
		}
		if started != nil {
			ann.StartedAt = string(started)
		}
		if stopped != nil {
			ann.StoppedAt = string(stopped)
		}

		l = append(l, ann)
	}

	return l, nil
}

func (db *DB) CreateTask(owner, op, args string, job uuid.UUID) (uuid.UUID, error) {
	id := uuid.NewRandom()
	return id, db.Exec(
		`INSERT INTO tasks (uuid, owner, op, args, job_uuid, status, log)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id.String(), owner, op, args, job.String(),
		"pending", "",
	)
}

func (db *DB) StartTask(id uuid.UUID, effective time.Time) error {
	return db.Exec(
		`UPDATE tasks SET status = ?, started_at = ? WHERE uuid = ?`,
		"running", effective, id.String(),
	)
}

func (db *DB) updateTaskStatus(id uuid.UUID, status string, effective time.Time) error {
	return db.Exec(
		`UPDATE tasks SET status = ?, stopped_at = ? WHERE uuid = ?`,
		status, effective, id.String(),
	)
}
func (db *DB) CancelTask(id uuid.UUID, effective time.Time) error {
	return db.updateTaskStatus(id, "canceled", effective)
}

func (db *DB) FailTask(id uuid.UUID, effective time.Time) error {
	return db.updateTaskStatus(id, "failed", effective)
}

func (db *DB) CompleteTask(id uuid.UUID, effective time.Time) error {
	return db.updateTaskStatus(id, "done", effective)
}

func (db *DB) UpdateTaskLog(id uuid.UUID, more string) error {
	return db.Exec(
		`UPDATE tasks SET log = log || ? WHERE uuid = ?`,
		more, id.String(),
	)
}

func (db *DB) CreateTaskArchive(id uuid.UUID, key string, effective time.Time) error {
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
		return err
	}
	defer r.Close()

	if !r.Next() {
		return fmt.Errorf("failed to determine expiration for task %s", id)
	}

	var expiry int
	if err := r.Scan(&expiry); err != nil {
		return err
	}
	r.Close()

	// insert an archive with all proper references, expiration, etc.
	archive_id := uuid.NewRandom()
	err = db.Exec(
		`INSERT INTO archives
			(uuid, target_uuid, store_uuid, store_key, taken_at, expires_at, notes)
			SELECT ?, t.uuid, s.uuid, ?, ?, ?, ""
				FROM tasks
					INNER JOIN jobs    j     ON j.uuid = tasks.job_uuid
					INNER JOIN targets t     ON t.uuid = j.target_uuid
					INNER JOIN stores  s     ON s.uuid = j.store_uuid
				WHERE tasks.uuid = ?`,
		archive_id.String(), key, effective, effective.Add(time.Duration(expiry)*time.Second), id.String(),
	)
	if err != nil {
		return err
	}

	// and finally, associate task -> archive
	return db.Exec(
		`UPDATE tasks SET archive_uuid = ? WHERE uuid = ?`,
		archive_id.String(), id.String(),
	)
}
