package api_agent

import (
	"bytes"
	"fmt"
	"github.com/starkandwayne/shield/db"
)

func FetchListSchedules(unused string) (*[]db.AnnotatedSchedule, error) {

	// Data to be returned of proper type
	data := &[]db.AnnotatedSchedule{}

	// Make uri based on options
	uri := fmt.Sprintf("v1/schedules")
	joiner := "?"
	if unused != "" {
		uri = fmt.Sprintf("%s%sunused=%s", uri, joiner, unused)
		joiner = "&"
	}

	// Call generic API request
	err := makeApiCall(data, `GET`, uri, nil)
	return data, err
}

func GetSchedule(uuid string) (*db.AnnotatedSchedule, error) {

	// Data to be returned of proper type
	data := &db.AnnotatedSchedule{}

	// Make uri based on options
	uri := fmt.Sprintf("v1/schedule/%s", uuid)

	// Call generic API request
	err := makeApiCall(data, `GET`, uri, nil)
	return data, err
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
