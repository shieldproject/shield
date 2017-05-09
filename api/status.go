package api

type Status struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func GetStatus() (Status, error) {
	uri, err := ShieldURI("/v1/status")
	if err != nil {
		return Status{}, err
	}

	var data Status
	return data, uri.Get(&data)
}

type JobsStatus map[string]JobHealth

type JobHealth struct {
	Name    string `json:"name"`
	LastRun int64  `json:"last_run"`
	NextRun int64  `json:"next_run"`
	Paused  bool   `json:"paused"`
	Status  string `json:"status"`
}

func GetJobsStatus() (JobsStatus, error) {
	uri, err := ShieldURI("/v1/status/jobs")
	if err != nil {
		return JobsStatus{}, err
	}

	var data JobsStatus
	return data, uri.Get(&data)
}
