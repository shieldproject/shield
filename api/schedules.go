package api

import (
	"encoding/json"

	"github.com/pborman/uuid"
)

type Schedule struct {
	UUID    string `json:"uuid,omitempty"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	When    string `json:"when"`
}

//Create sends a request to the SHIELD core to create a new schedule object.
// Returns the UUID of the created schedule
func (s *Schedule) Create() (string, error) {
	data := struct {
		UUID string `json:"uuid"`
	}{}
	uri, err := ShieldURI("/v1/schedules")
	if err != nil {
		return "", err
	}

	contentJSON, err := json.Marshal(s)
	if err != nil {
		panic("Schedule given to Create was nil")
	}
	err = uri.Post(&data, string(contentJSON))
	return data.UUID, err
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
