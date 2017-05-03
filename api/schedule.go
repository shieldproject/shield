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
	Unused     YesNo
	Name       string
	ExactMatch YesNo
}

func GetSchedules(filter ScheduleFilter) ([]Schedule, error) {
	uri, err := ShieldURI("/v1/schedules")
	if err != nil {
		return []Schedule{}, err
	}
	uri.MaybeAddParameter("name", filter.Name)
	uri.MaybeAddParameter("unused", filter.Unused)
	uri.MaybeAddParameter("exact", filter.ExactMatch)

	var data []Schedule
	return data, uri.Get(&data)
}

func GetSchedule(id uuid.UUID) (Schedule, error) {
	var data Schedule
	uri, err := ShieldURI("/v1/schedule/%s", id)
	if err != nil {
		return Schedule{}, err
	}
	return data, uri.Get(&data)
}

func CreateSchedule(contentJSON string) (Schedule, error) {
	data := struct {
		UUID string `json:"uuid"`
	}{}
	uri, err := ShieldURI("/v1/schedules")
	if err != nil {
		return Schedule{}, err
	}
	if err := uri.Post(&data, contentJSON); err != nil {
		return Schedule{}, err
	}
	return GetSchedule(uuid.Parse(data.UUID))
}

func UpdateSchedule(id uuid.UUID, contentJSON string) (Schedule, error) {
	uri, err := ShieldURI("/v1/schedule/%s", id)
	if err != nil {
		return Schedule{}, err
	}
	if err := uri.Put(nil, contentJSON); err != nil {
		return Schedule{}, err
	}
	return GetSchedule(id)
}

func DeleteSchedule(id uuid.UUID) error {
	uri, err := ShieldURI("/v1/schedule/%s", id)
	if err != nil {
		return err
	}
	return uri.Delete(nil)
}
