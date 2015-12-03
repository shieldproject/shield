package api

type Schedule struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	When    string `json:"when"`
}

type ScheduleFilter struct {
	Unused YesNo
}

func FetchListSchedules(unused string) ([]Schedule, error) {
	return GetSchedules(ScheduleFilter{
		Unused: MaybeString(unused),
	})
}

func GetSchedules(filter ScheduleFilter) ([]Schedule, error) {
	uri := ShieldURI("/v1/schedules")
	uri.MaybeAddParameter("unused", filter.Unused)

	var data []Schedule
	return data, uri.Get(&data)
}

func GetSchedule(uuid string) (Schedule, error) {
	var data Schedule
	return data, ShieldURI("/v1/schedule/%s", uuid).Get(&data)
}

func CreateSchedule(contentJSON string) (Schedule, error) {
	data := struct {
		UUID string `json:"uuid"`
	}{}
	err := ShieldURI("/v1/schedules").Post(&data, contentJSON)
	if err == nil {
		return GetSchedule(data.UUID)
	}
	return Schedule{}, err
}

func UpdateSchedule(uuid string, contentJSON string) (Schedule, error) {
	err := ShieldURI("/v1/schedule/%s", uuid).Put(nil, contentJSON)
	if err == nil {
		return GetSchedule(uuid)
	}
	return Schedule{}, err
}

func DeleteSchedule(uuid string) error {
	return ShieldURI("/v1/schedule/%s", uuid).Delete(nil)
}
