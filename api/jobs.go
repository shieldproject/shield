package api

import (
	"github.com/pborman/uuid"
)

type Job struct {
	UUID           string `json:"uuid"`
	Name           string `json:"name"`
	Summary        string `json:"summary"`
	RetentionName  string `json:"retention_name"`
	RetentionUUID  string `json:"retention_uuid"`
	Expiry         int    `json:"expiry"`
	ScheduleName   string `json:"schedule_name"`
	ScheduleUUID   string `json:"schedule_uuid"`
	ScheduleWhen   string `json:"schedule_when"`
	Paused         bool   `json:"paused"`
	StoreUUID      string `json:"store_uuid"`
	StoreName      string `json:"store_name"`
	StorePlugin    string `json:"store_plugin"`
	StoreEndpoint  string `json:"store_endpoint"`
	TargetUUID     string `json:"target_uuid"`
	TargetName     string `json:"target_name"`
	TargetPlugin   string `json:"target_plugin"`
	TargetEndpoint string `json:"target_endpoint"`
	Agent          string `json:"agent"`
}

type JobFilter struct {
	Name      string
	Target    string
	Store     string
	Schedule  string
	Retention string
	Paused    YesNo
}

func GetJobs(filter JobFilter) ([]Job, error) {
	uri := ShieldURI("/v1/jobs")
	uri.MaybeAddParameter("name", filter.Name)
	uri.MaybeAddParameter("target", filter.Target)
	uri.MaybeAddParameter("store", filter.Store)
	uri.MaybeAddParameter("schedule", filter.Schedule)
	uri.MaybeAddParameter("retention", filter.Retention)
	uri.MaybeAddParameter("paused", filter.Paused)

	var data []Job
	return data, uri.Get(&data)
}

func GetJob(id uuid.UUID) (Job, error) {
	var data Job
	return data, ShieldURI("/v1/job/%s", id).Get(&data)
}

func IsPausedJob(id uuid.UUID) (bool, error) {
	data, err := GetJob(id)
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
		return GetJob(uuid.Parse(data.UUID))
	}
	return Job{}, err
}

func UpdateJob(id uuid.UUID, contentJSON string) (Job, error) {
	err := ShieldURI("/v1/job/%s", id).Put(nil, contentJSON)
	if err == nil {
		return GetJob(id)
	}
	return Job{}, err
}

func DeleteJob(id uuid.UUID) error {
	return ShieldURI("/v1/job/%s", id).Delete(nil)
}

func PauseJob(id uuid.UUID) error {
	return ShieldURI("/v1/job/%s/pause", id).Post(nil, "{}")
}

func UnpauseJob(id uuid.UUID) error {
	return ShieldURI("/v1/job/%s/unpause", id).Post(nil, "{}")
}

func RunJob(id uuid.UUID, ownerJSON string) error {
	return ShieldURI("/v1/job/%s/run", id).Post(nil, ownerJSON)
}
