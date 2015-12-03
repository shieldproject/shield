package api

import (
	"github.com/starkandwayne/shield/db"
)

type TaskFilter struct {
	Status string
	Debug  YesNo
}

func FetchListTasks(status string, debugFlag bool) (*[]db.AnnotatedTask, error) {
	// FIXME: legacy
	return GetTasks(TaskFilter{
		Status: status,
		Debug:  Maybe(debugFlag),
	})
}

func GetTasks(filter TaskFilter) (*[]db.AnnotatedTask, error) {
	uri := ShieldURI("/v1/tasks")
	uri.MaybeAddParameter("status", filter.Status)
	uri.MaybeAddParameter("debug", filter.Debug)

	data := &[]db.AnnotatedTask{}
	return data, uri.Get(&data)
}

func GetTask(uuid string) (*db.AnnotatedTask, error) {
	data := &db.AnnotatedTask{}
	return data, ShieldURI("v1/task/%s", uuid).Get(&data)
}

func CancelTask(uuid string) error {
	return ShieldURI("/v1/task/%s", uuid).Delete(nil)
}
