package api_agent

import (
	"bytes"
	"fmt"
	"github.com/starkandwayne/shield/db"
)

func FetchListJobs(target, store, schedule, retention, paused string) (*[]db.AnnotatedJob, error) {
	uri := ShieldURI("/v1/jobs")
	if target != "" {
		uri.AddParameter("target", target)
	}
	if store != "" {
		uri.AddParameter("store", store)
	}
	if schedule != "" {
		uri.AddParameter("schedule", schedule)
	}
	if retention != "" {
		uri.AddParameter("retention", retention)
	}
	if paused != "" {
		uri.AddParameter("paused", paused)
	}

	data := &[]db.AnnotatedJob{}
	return data, uri.Get(&data)
}

func GetJob(uuid string) (*db.AnnotatedJob, error) {
	data := &db.AnnotatedJob{}
	return data, ShieldURI("/v1/job/%s", uuid).Get(&data)
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
