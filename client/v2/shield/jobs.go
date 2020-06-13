package shield

import (
	"fmt"

	qs "github.com/jhunt/go-querytron"
	"github.com/pborman/uuid"
)

type Job struct {
	UUID       string `json:"uuid,omitempty"`
	Name       string `json:"name"`
	Summary    string `json:"summary"`
	Schedule   string `json:"schedule"`
	KeepDays   int    `json:"keep_days"`
	KeepN      int    `json:"keep_n"`
	Retain     string `json:"-"`
	Paused     bool   `json:"paused"`
	Agent      string `json:"agent"`
	LastStatus string `json:"last_task_status"`
	LastRun    int64  `json:"last_run"`

	TargetUUID string `json:"-"`
	Target     struct {
		UUID   string `json:"uuid"`
		Name   string `json:"name"`
		Agent  string `json:"agent"`
		Plugin string `json:"plugin"`

		Endpoint string                 `json:"endpoint,omitempty"`
		Config   map[string]interface{} `json:"config,omitempty"`
	} `json:"target"`

	Bucket string `json:"bucket"`

	AgentHost string `json:"-"`
	AgentPort int    `json:"-"`
}

type JobFilter struct {
	UUID   string `qs:"uuid"`
	Fuzzy  bool   `qs:"exact:f:t"`
	Name   string `qs:"name"`
	Bucket string `qs:"bucket"`
	Target string `qs:"target"`
	Paused *bool  `qs:"paused:t:f"`
}

func fixupJobResponse(p *Job) {
}

func (c *Client) ListJobs(filter *JobFilter) ([]*Job, error) {
	u := qs.Generate(filter).Encode()
	var out []*Job
	if err := c.get(fmt.Sprintf("/v2/jobs?%s", u), &out); err != nil {
		return nil, err
	}
	for _, p := range out {
		fixupJobResponse(p)
	}
	return out, nil
}

func (c *Client) FindJob(q string, fuzzy bool) (*Job, error) {
	if uuid.Parse(q) != nil {
		return c.GetJob(q)
	}

	l, err := c.ListJobs(&JobFilter{
		UUID:  q,
		Name:  q,
		Fuzzy: fuzzy,
	})
	if err != nil {
		return nil, err
	}

	if len(l) == 0 {
		return nil, fmt.Errorf("no matching job found")
	}
	if len(l) > 1 {
		return nil, fmt.Errorf("multiple matching jobs found")
	}

	return c.GetJob(l[0].UUID)
}

func (c *Client) GetJob(uuid string) (*Job, error) {
	var out *Job
	if err := c.get(fmt.Sprintf("/v2/jobs/%s", uuid), &out); err != nil {
		return nil, err
	}
	fixupJobResponse(out)
	return out, nil
}

func (c *Client) CreateJob(job *Job) (*Job, error) {
	var out *Job

	in := struct {
		Name     string `json:"name"`
		Summary  string `json:"summary"`
		Schedule string `json:"schedule"`
		Retain   string `json:"retain"`
		Paused   bool   `json:"paused"`
		Bucket   string `json:"bucket"`
		Target   string `json:"target"`
	}{
		Name:     job.Name,
		Summary:  job.Summary,
		Schedule: job.Schedule,
		Retain:   job.Retain,
		Paused:   job.Paused,
		Target:   job.TargetUUID,
		Bucket:   job.Bucket,
	}
	if err := c.post("/v2/jobs", in, &out); err != nil {
		return nil, err
	}
	fixupJobResponse(out)
	return out, nil
}

func (c *Client) UpdateJob(job *Job) (*Job, error) {
	in := struct {
		Name     string `json:"name,omitempty"`
		Summary  string `json:"summary,omitempty"`
		Schedule string `json:"schedule,omitempty"`
		Retain   string `json:"retain,omitempty"`
		Bucket   string `json:"bucket,omitempty"`
		Target   string `json:"target,omitempty"`
	}{
		Name:     job.Name,
		Summary:  job.Summary,
		Schedule: job.Schedule,
		Retain:   job.Retain,
		Target:   job.TargetUUID,
		Bucket:   job.Bucket,
	}
	if err := c.put(fmt.Sprintf("/v2/jobs/%s", job.UUID), in, nil); err != nil {
		return nil, err
	}
	return c.GetJob(job.UUID)
}

func (c *Client) DeleteJob(in *Job) (Response, error) {
	var out Response
	return out, c.delete(fmt.Sprintf("/v2/jobs/%s", in.UUID), &out)
}

func (c *Client) PauseJob(job *Job) (Response, error) {
	var out Response
	return out, c.post(fmt.Sprintf("/v2/jobs/%s/pause", job.UUID), nil, &out)
}

func (c *Client) UnpauseJob(job *Job) (Response, error) {
	var out Response
	return out, c.post(fmt.Sprintf("/v2/jobs/%s/unpause", job.UUID), nil, &out)
}

func (c *Client) RunJob(job *Job) (Response, error) {
	var out Response
	return out, c.post(fmt.Sprintf("/v2/jobs/%s/run", job.UUID), nil, &out)
}

func (j Job) Status() string {
	if j.Paused {
		return "paused"
	}
	switch j.LastStatus {
	case "done":
		return "healthy"
	case "":
		return "(never run)"
	default:
		return j.LastStatus
	}
}
