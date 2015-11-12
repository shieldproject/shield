package db

import (
	"github.com/pborman/uuid"
	"time"
)

func (db *DB) CreateTask(owner, op, args string, job, archive uuid.UUID) (uuid.UUID, error) {
	id := uuid.NewRandom()
	return id, db.Exec(
		`INSERT INTO tasks (uuid, owner, op, args, job_uuid, archive_uuid, status, log)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id.String(), owner, op, args, job.String(), archive.String(),
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
