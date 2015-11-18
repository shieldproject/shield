package api_agent

import (
	"fmt"
	"github.com/starkandwayne/shield/db"
)

func FetchListTasks(status string, debugFlag bool) (*[]db.AnnotatedTask, error) {

	// Data to be returned of proper type
	data := &[]db.AnnotatedTask{}

	// Make uri based on options
	uri := fmt.Sprintf("v1/tasks")
	joiner := "?"
	if status != "" {
		uri = fmt.Sprintf("%s%sstatus=%s", uri, joiner, status)
		joiner = "&"
	}
	if debugFlag != false {
		uri = fmt.Sprintf("%s%sdebug", uri, joiner)
		joiner = "&"
	}

	// Call generic API request
	err := makeApiCall(data, `GET`, uri, nil)
	return data, err
}

func GetTask(uuid string) (*db.AnnotatedTask, error) {

	// Data to be returned of proper type
	data := &db.AnnotatedTask{}

	// Make uri based on options
	uri := fmt.Sprintf("v1/task/%s", uuid)

	// Call generic API request
	err := makeApiCall(data, `GET`, uri, nil)
	return data, err
}

func CancelTask(uuid string) error {

	uri := fmt.Sprintf("v1/task/%s", uuid)

	data := struct {
		Status string `json:"ok"`
	}{}

	err := makeApiCall(&data, `DELETE`, uri, nil)

	return err
}
