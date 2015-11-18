package api_agent

import (
	"bytes"
	"fmt"
	"github.com/starkandwayne/shield/db"
)

func FetchListJobs(target, store, schedule, retention, paused string) (*[]db.AnnotatedJob, error) {

	// Data to be returned of proper type
	data := &[]db.AnnotatedJob{}

	// Make uri based on options
	uri := fmt.Sprintf("v1/jobs")
	joiner := "?"
	if target != "" {
		uri = fmt.Sprintf("%s%starget=%s", uri, joiner, target)
		joiner = "&"
	}
	if store != "" {
		uri = fmt.Sprintf("%s%sstore=%s", uri, joiner, store)
		joiner = "&"
	}
	if schedule != "" {
		uri = fmt.Sprintf("%s%sschedule=%s", uri, joiner, schedule)
		joiner = "&"
	}
	if retention != "" {
		uri = fmt.Sprintf("%s%sretention=%s", uri, joiner, retention)
		joiner = "&"
	}
	if paused != "" {
		uri = fmt.Sprintf("%s%spaused=%s", uri, joiner, paused)
		joiner = "&"
	}

	// Call generic API request
	err := makeApiCall(data, `GET`, uri, nil)
	return data, err
}

func GetJob(uuid string) (*db.AnnotatedJob, error) {

	// Data to be returned of proper type
	data := &db.AnnotatedJob{}

	// Make uri based on options
	uri := fmt.Sprintf("v1/job/%s", uuid)

	// Call generic API request
	err := makeApiCall(data, `GET`, uri, nil)
	return data, err
}

func IsPausedJob(uuid string) (bool, error) {
	// UUID validation can be handled by GetJob
	data, err := GetJob(uuid)
	if err != nil {
		return false, err
	}

	return data.Paused, err
}

func CreateJob(content string) (*db.AnnotatedJob, error) {

	data := struct {
		Status string `json:"ok"`
		UUID   string `json:"uuid"`
	}{}

	buf := bytes.NewBufferString(content)

	err := makeApiCall(&data, `POST`, `v1/jobs`, buf)

	if err == nil {
		return GetJob(data.UUID)
	}
	return nil, err
}

func UpdateJob(uuid string, content string) (*db.AnnotatedJob, error) {

	data := struct {
		Status string `json:"ok"`
	}{}

	buf := bytes.NewBufferString(content)

	uri := fmt.Sprintf("v1/job/%s", uuid)

	err := makeApiCall(&data, `PUT`, uri, buf)

	if err == nil {
		return GetJob(uuid)
	}
	return nil, err
}

func DeleteJob(uuid string) error {

	uri := fmt.Sprintf("v1/job/%s", uuid)

	data := struct {
		Status string `json:"ok"`
	}{}

	err := makeApiCall(&data, `DELETE`, uri, nil)

	return err
}

func PauseJob(uuid string) error {

	data := struct {
		Status string `json:"ok"`
	}{}

	buf := bytes.NewBufferString("")

	uri := fmt.Sprintf("v1/job/%s/pause", uuid)

	err := makeApiCall(&data, `POST`, uri, buf)

	return err
}

func UnpauseJob(uuid string) error {

	data := struct {
		Status string `json:"ok"`
	}{}

	buf := bytes.NewBufferString("")

	uri := fmt.Sprintf("v1/job/%s/unpause", uuid)

	err := makeApiCall(&data, `POST`, uri, buf)

	return err
}

func RunJob(uuid, owner string) error {

	data := struct {
		Status string `json:"ok"`
	}{}

	buf := bytes.NewBufferString(owner)

	uri := fmt.Sprintf("v1/job/%s/run", uuid)

	err := makeApiCall(&data, `POST`, uri, buf)

	return err
}
