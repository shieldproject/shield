package api_agent

import (
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

func CreateSchedule(contentJSON string) (*db.AnnotatedSchedule, error) {
	data := struct {
		UUID string `json:"uuid"`
	}{}
	err := ShieldURI("/v1/schedules").Post(&data, contentJSON)
	if err == nil {
		return GetSchedule(data.UUID)
	}
	return nil, err
}

func UpdateSchedule(uuid string, contentJSON string) (*db.AnnotatedSchedule, error) {
	err := ShieldURI("/v1/schedule/%s", uuid).Put(nil, contentJSON)
	if err == nil {
		return GetSchedule(uuid)
	}
	return nil, err
}

func DeleteSchedule(uuid string) error {
	return ShieldURI("/v1/schedule/%s", uuid).Delete(nil)
}
