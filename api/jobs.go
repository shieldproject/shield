package api

type Job struct {
	UUID           string `json:"uuid"`
	Name           string `json:"name"`
	Summary        string `json:"summary"`
	RetentionName  string `json:"retention_name"`
	RetentionUUID  string `json:"retention_uuid"`
	Expiry         int    `json:"expiry"`
	ScheduleName   string `json:"schedule_name"`
	ScheduleUUID   string `json:"schedule_uuid"`
	Schedule       string `json:"schedule"`
	Paused         bool   `json:"paused"`
	StoreUUID      string `json:"store_uuid"`
	StorePlugin    string `json:"store_plugin"`
	StoreEndpoint  string `json:"store_endpoint"`
	TargetUUID     string `json:"target_uuid"`
	TargetPlugin   string `json:"target_plugin"`
	TargetEndpoint string `json:"target_endpoint"`
	Agent          string `json:"agent"`
}

type JobFilter struct {
	Target    string
	Store     string
	Schedule  string
	Retention string
	Paused    YesNo
}

func FetchListJobs(target, store, schedule, retention, paused string) ([]Job, error) {
	return GetJobs(JobFilter{
		Target:    target,
		Store:     store,
		Schedule:  schedule,
		Retention: retention,
		Paused:    MaybeString(paused),
	})
}

func GetJobs(filter JobFilter) ([]Job, error) {
	uri := ShieldURI("/v1/jobs")
	uri.MaybeAddParameter("target", filter.Target)
	uri.MaybeAddParameter("store", filter.Store)
	uri.MaybeAddParameter("schedule", filter.Schedule)
	uri.MaybeAddParameter("retention", filter.Retention)
	uri.MaybeAddParameter("paused", filter.Paused)

	var data []Job
	return data, uri.Get(&data)
}

func GetJob(uuid string) (Job, error) {
	var data Job
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

func CreateJob(contentJSON string) (Job, error) {
	data := struct {
		Status string `json:"ok"`
		UUID   string `json:"uuid"`
	}{}
	err := ShieldURI("/v1/jobs").Post(&data, contentJSON)
	if err == nil {
		return GetJob(data.UUID)
	}
	return Job{}, err
}

func UpdateJob(uuid string, contentJSON string) (Job, error) {
	err := ShieldURI("/v1/job/%s", uuid).Put(nil, contentJSON)
	if err == nil {
		return GetJob(uuid)
	}
	return Job{}, err
}

func DeleteJob(uuid string) error {
	return ShieldURI("/v1/job/%s", uuid).Delete(nil)
}

func PauseJob(uuid string) error {
	return ShieldURI("/v1/job/%s/pause", uuid).Post(nil, "")
}

func UnpauseJob(uuid string) error {
	return ShieldURI("/v1/job/%s/unpause", uuid).Post(nil, "")
}

func RunJob(uuid string, ownerJSON string) error {
	return ShieldURI("/v1/job/%s/run", uuid).Post(nil, ownerJSON)
}
