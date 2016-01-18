package api

import (
	"github.com/pborman/uuid"
)

type Schedule struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	When    string `json:"when"`
}

type ScheduleFilter struct {
	Unused YesNo
	Name   string
}

func GetSchedules(filter ScheduleFilter) ([]Schedule, error) {
	uri := ShieldURI("/v1/schedules")
	uri.MaybeAddParameter("name", filter.Name)
	uri.MaybeAddParameter("unused", filter.Unused)

	var data []Schedule
	return data, uri.Get(&data)
}

func GetSchedule(id uuid.UUID) (Schedule, error) {
	var data Schedule
	return data, ShieldURI("/v1/schedule/%s", id).Get(&data)
}

func CreateSchedule(contentJSON string) (Schedule, error) {
	data := struct {
		UUID string `json:"uuid"`
	}{}
	err := ShieldURI("/v1/schedules").Post(&data, contentJSON)
	if err == nil {
		return GetSchedule(uuid.Parse(data.UUID))
	}
	return Schedule{}, err
}

func UpdateSchedule(id uuid.UUID, contentJSON string) (Schedule, error) {
	err := ShieldURI("/v1/schedule/%s", id).Put(nil, contentJSON)
	if err == nil {
		return GetSchedule(id)
	}
	return Schedule{}, err
}

func DeleteSchedule(id uuid.UUID) error {
	return ShieldURI("/v1/schedule/%s", id).Delete(nil)
}
