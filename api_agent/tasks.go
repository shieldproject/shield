package api_agent

import (
	"fmt"
	"github.com/starkandwayne/shield/db"
)

func FetchListTasks(status string, debugFlag bool) (*[]db.AnnotatedTask, error) {
	uri := ShieldURI("/v1/tasks")
	if status != "" {
		uri.AddParameter("status", status)
	}
	if debugFlag != false {
		uri.AddParameter("debug", true)
	}

	data := &[]db.AnnotatedTask{}
	return data, uri.Get(&data)
}

func GetTask(uuid string) (*db.AnnotatedTask, error) {
	data := &db.AnnotatedTask{}
	return data, ShieldURI("v1/task/%s", uuid).Get(&data)
}

func CancelTask(uuid string) error {

	uri := fmt.Sprintf("v1/task/%s", uuid)

	data := struct {
		Status string `json:"ok"`
	}{}

	err := makeApiCall(&data, `DELETE`, uri, nil)

	return err
}
