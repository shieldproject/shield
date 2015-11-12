package db

import (
	"github.com/pborman/uuid"
	"time"
	"fmt"
)

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

func (db *DB) CancelTask(id uuid.UUID, effective time.Time) error {
	return db.Exec(
		`UPDATE tasks SET status = ?, stopped_at = ? WHERE uuid = ?`,
		"canceled", effective, id.String(),
	)
}

func (db *DB) CompleteTask(id uuid.UUID, effective time.Time) error {
	return db.Exec(
		`UPDATE tasks SET status = ?, stopped_at = ? WHERE uuid = ?`,
		"done", effective, id.String(),
	)
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
			(uuid, target_uuid, store_uuid, store_key, taken_at, expires_at)
			SELECT ?, t.uuid, s.uuid, ?, ?, ?
				FROM tasks
					INNER JOIN jobs    j     ON j.uuid = tasks.job_uuid
					INNER JOIN targets t     ON t.uuid = j.target_uuid
					INNER JOIN stores  s     ON s.uuid = j.store_uuid
				WHERE tasks.uuid = ?`,
		archive_id.String(), key, effective, effective.Add(time.Duration(expiry) * time.Second), id.String(),
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
