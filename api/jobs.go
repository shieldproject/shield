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
	Schedule       string `json:"schedule"`
	Paused         bool   `json:"paused"`
	ScheduleName   string `json:"schedule_name,omitempty"`
	ScheduleUUID   string `json:"schedule_uuid,omitempty"`
	ScheduleWhen   string `json:"schedule_when,omitempty"`
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
	Name       string
	Target     string
	Store      string
	Retention  string
	Paused     YesNo
	ExactMatch YesNo
}

func GetJobs(filter JobFilter) ([]Job, error) {
	uri, err := ShieldURI("/v1/jobs")
	if err != nil {
		return []Job{}, err
	}
	uri.MaybeAddParameter("name", filter.Name)
	uri.MaybeAddParameter("target", filter.Target)
	uri.MaybeAddParameter("store", filter.Store)
	uri.MaybeAddParameter("retention", filter.Retention)
	uri.MaybeAddParameter("paused", filter.Paused)
	uri.MaybeAddParameter("exact", filter.ExactMatch)

	var data []Job
	return data, uri.Get(&data)
}

func GetJob(id uuid.UUID) (Job, error) {
	var data Job
	uri, err := ShieldURI("/v1/job/%s", id)
	if err != nil {
		return Job{}, err
	}
	return data, uri.Get(&data)
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
	uri, err := ShieldURI("/v1/jobs")
	if err != nil {
		return Job{}, err
	}
	if err := uri.Post(&data, contentJSON); err != nil {
		return Job{}, err
	}
	return GetJob(uuid.Parse(data.UUID))
}

func UpdateJob(id uuid.UUID, contentJSON string) (Job, error) {
	uri, err := ShieldURI("/v1/job/%s", id)
	if err != nil {
		return Job{}, err
	}
	if err := uri.Put(nil, contentJSON); err != nil {
		return Job{}, err
	}
	return GetJob(id)
}

func DeleteJob(id uuid.UUID) error {
	uri, err := ShieldURI("/v1/job/%s", id)
	if err != nil {
		return err
	}
	return uri.Delete(nil)
}

func PauseJob(id uuid.UUID) error {
	uri, err := ShieldURI("/v1/job/%s/pause", id)
	if err != nil {
		return err
	}
	return uri.Post(nil, "{}")
}

func UnpauseJob(id uuid.UUID) error {
	uri, err := ShieldURI("/v1/job/%s/unpause", id)
	if err != nil {
		return err
	}
	return uri.Post(nil, "{}")
}

//If the string returned is the empty string but the error returned is nil, then
//it is most likely that the deployed version of the backend does not support
//handing back the uuid for an adhoc task.
func RunJob(id uuid.UUID, ownerJSON string) (string, error) {
	respMap := make(map[string]string)
	uri, err := ShieldURI("/v1/job/%s/run", id)
	if err != nil {
		return "", err
	}
	if err := uri.Post(&respMap, ownerJSON); err != nil {
		return "", err
	}
	return respMap["task_uuid"], nil
}
