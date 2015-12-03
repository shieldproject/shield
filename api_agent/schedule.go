package api_agent

import (
	"bytes"
	"fmt"
	"github.com/starkandwayne/shield/db"
)

func FetchListSchedules(unused string) (*[]db.AnnotatedSchedule, error) {
	uri := ShieldURI("/v1/schedules")
	if unused != "" {
		uri.AddParameter("unused", unused)
	}

	data := &[]db.AnnotatedSchedule{}
	return data, uri.Get(&data)
}

func GetSchedule(uuid string) (*db.AnnotatedSchedule, error) {
	data := &db.AnnotatedSchedule{}
	return data, ShieldURI("/v1/schedule/%s", uuid).Get(&data)
}

func CreateSchedule(content string) (*db.AnnotatedSchedule, error) {

	data := struct {
		Status string `json:"ok"`
		UUID   string `json:"uuid"`
	}{}

	buf := bytes.NewBufferString(content)

	err := makeApiCall(&data, `POST`, `v1/schedules`, buf)

	if err == nil {
		return GetSchedule(data.UUID)
	}
	return nil, err
}

func UpdateSchedule(uuid string, content string) (*db.AnnotatedSchedule, error) {

	data := struct {
		Status string `json:"ok"`
	}{}

	buf := bytes.NewBufferString(content)

	uri := fmt.Sprintf("v1/schedule/%s", uuid)

	err := makeApiCall(&data, `PUT`, uri, buf)

	if err == nil {
		return GetSchedule(uuid)
	}
	return nil, err
}

func DeleteSchedule(uuid string) error {

	uri := fmt.Sprintf("v1/schedule/%s", uuid)

	data := struct {
		Status string `json:"ok"`
	}{}

	err := makeApiCall(&data, `DELETE`, uri, nil)

	return err
}
