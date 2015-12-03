package api_agent

import (
	"github.com/starkandwayne/shield/db"
)

func FetchListJobs(target, store, schedule, retention, paused string) (*[]db.AnnotatedJob, error) {
	uri := ShieldURI("/v1/jobs")
	if target != "" {
		uri.AddParameter("target", target)
	}
	if store != "" {
		uri.AddParameter("store", store)
	}
	if schedule != "" {
		uri.AddParameter("schedule", schedule)
	}
	if retention != "" {
		uri.AddParameter("retention", retention)
	}
	if paused != "" {
		uri.AddParameter("paused", paused)
	}

	data := &[]db.AnnotatedJob{}
	return data, uri.Get(&data)
}

func GetJob(uuid string) (*db.AnnotatedJob, error) {
	data := &db.AnnotatedJob{}
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

func CreateJob(contentJSON string) (*db.AnnotatedJob, error) {
	data := struct {
		Status string `json:"ok"`
		UUID   string `json:"uuid"`
	}{}
	err := ShieldURI("/v1/jobs").Post(&data, contentJSON)
	if err == nil {
		return GetJob(data.UUID)
	}
	return nil, err
}

func UpdateJob(uuid string, contentJSON string) (*db.AnnotatedJob, error) {
	err := ShieldURI("/v1/job/%s", uuid).Put(nil, contentJSON)
	if err == nil {
		return GetJob(uuid)
	}
	return nil, err
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
