package shield

type BacklogStatus struct {
	Priority int    `json:"priority"`
	Position int    `json:"position"`
	TaskUUID string `json:"task_uuid"`

	Op    string `json:"op"`
	Agent string `json:"agent"`

	Tenant *struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
	} `json:"tenant,omitempty"`

	Store *struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
	} `json:"store,omitempty"`

	System *struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
	} `json:"system,omitempty"`

	Job *struct {
		UUID     string `json:"uuid"`
		Name     string `json:"name"`
		Schedule string `json:"schedule"`
	} `json:"job,omitempty"`

	Archive *struct {
		UUID string `json:"uuid"`
		Size int64  `json:"size"`
	} `json:"archive,omitempty"`
}

type WorkerStatus struct {
	ID       int    `json:"id"`
	Idle     bool   `json:"idle"`
	TaskUUID string `json:"task_uuid"`

	Op     string `json:"op"`
	Status string `json:"status"`
	Agent  string `json:"agent"`

	Tenant *struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
	} `json:"tenant,omitempty"`

	Store *struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
	} `json:"store,omitempty"`

	System *struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
	} `json:"system,omitempty"`

	Job *struct {
		UUID     string `json:"uuid"`
		Name     string `json:"name"`
		Schedule string `json:"schedule"`
	} `json:"job,omitempty"`

	Archive *struct {
		UUID string `json:"uuid"`
		Size int64  `json:"size"`
	} `json:"archive,omitempty"`
}

type SchedulerStatus struct {
	Backlog []BacklogStatus `json:"backlog"`
	Workers []WorkerStatus  `json:"workers"`
}

func (c *Client) SchedulerStatus() (*SchedulerStatus, error) {
	var st *SchedulerStatus
	return st, c.get("/v2/scheduler/status", &st)
}
