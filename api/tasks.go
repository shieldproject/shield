package api

import (
	"github.com/pborman/uuid"

	. "github.com/starkandwayne/shield/timestamp"
)

type Task struct {
	UUID        string    `json:"uuid"`
	Owner       string    `json:"owner"`
	Op          string    `json:"type"`
	JobUUID     string    `json:"job_uuid"`
	ArchiveUUID string    `json:"archive_uuid"`
	Status      string    `json:"status"`
	StartedAt   Timestamp `json:"started_at"`
	StoppedAt   Timestamp `json:"stopped_at"`
	TimeoutAt   Timestamp `json:"timeout_at"`
	Log         string    `json:"log"`
}

type TaskFilter struct {
	Status string
	Debug  YesNo
}

func GetTasks(filter TaskFilter) ([]Task, error) {
	uri := ShieldURI("/v1/tasks")
	uri.MaybeAddParameter("status", filter.Status)
	uri.MaybeAddParameter("debug", filter.Debug)

	var data []Task
	return data, uri.Get(&data)
}

func GetTask(id uuid.UUID) (Task, error) {
	var data Task
	return data, ShieldURI("/v1/task/%s", id).Get(&data)
}

func CancelTask(id uuid.UUID) error {
	return ShieldURI("/v1/task/%s", id).Delete(nil)
}
