package api

import (
	"github.com/pborman/uuid"

	. "github.com/starkandwayne/goutils/timestamp"
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
	Limit  string
}

func GetTasks(filter TaskFilter) ([]Task, error) {
	uri, err := ShieldURI("/v1/tasks")
	if err != nil {
		return []Task{}, err
	}
	uri.MaybeAddParameter("debug", filter.Debug)
	uri.MaybeAddParameter("limit", filter.Limit)
	uri.MaybeAddParameter("status", filter.Status)

	var data []Task
	return data, uri.Get(&data)
}

func GetTask(id uuid.UUID) (Task, error) {
	var data Task
	uri, err := ShieldURI("/v1/task/%s", id)
	if err != nil {
		return Task{}, err
	}
	return data, uri.Get(&data)
}

func CancelTask(id uuid.UUID) error {
	uri, err := ShieldURI("/v1/task/%s", id)
	if err != nil {
		return err
	}
	return uri.Delete(nil)
}
