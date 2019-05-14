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
	LastStatus string `json:"status"`
	LastRun    int64  `json:"last_run"`
	FixedKey   bool   `json:"fixed_key"`

	TargetUUID string `json:"-"`
	Target     struct {
		UUID   string `json:"uuid"`
		Name   string `json:"name"`
		Agent  string `json:"agent"`
		Plugin string `json:"plugin"`

		Endpoint string                 `json:"endpoint,omitempty"`
		Config   map[string]interface{} `json:"config,omitempty"`
	} `json:"target"`

	StoreUUID string `json:"-"`
	Store     struct {
		UUID    string `json:"uuid"`
		Name    string `json:"name"`
		Agent   string `json:"agent"`
		Plugin  string `json:"plugin"`
		Summary string `json:"summary"`

		Endpoint string                 `json:"endpoint,omitempty"`
		Config   map[string]interface{} `json:"config,omitempty"`
	} `json:"store"`

	AgentHost string `json:"-"`
	AgentPort int    `json:"-"`
}

type JobFilter struct {
	UUID   string `qs:"uuid"`
	Fuzzy  bool   `qs:"exact:f:t"`
	Name   string `qs:"name"`
	Store  string `qs:"store"`
	Target string `qs:"target"`
	Paused *bool  `qs:"paused:t:f"`
}

func fixupJobResponse(p *Job) {
}

func (c *Client) ListJobs(parent *Tenant, filter *JobFilter) ([]*Job, error) {
	u := qs.Generate(filter).Encode()
	var out []*Job
	if err := c.get(fmt.Sprintf("/v2/tenants/%s/jobs?%s", parent.UUID, u), &out); err != nil {
		return nil, err
	}
	for _, p := range out {
		fixupJobResponse(p)
	}
	return out, nil
}

func (c *Client) FindJob(tenant *Tenant, q string, fuzzy bool) (*Job, error) {
	if uuid.Parse(q) != nil {
		return c.GetJob(tenant, q)
	}

	l, err := c.ListJobs(tenant, &JobFilter{
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

	return c.GetJob(tenant, l[0].UUID)
}

func (c *Client) GetJob(parent *Tenant, uuid string) (*Job, error) {
	if parent == nil {
		return nil, nil
	}

	var out *Job
	if err := c.get(fmt.Sprintf("/v2/tenants/%s/jobs/%s", parent.UUID, uuid), &out); err != nil {
		return nil, err
	}
	fixupJobResponse(out)
	return out, nil
}

func (c *Client) CreateJob(parent *Tenant, job *Job) (*Job, error) {
	var out *Job

	in := struct {
		Name     string `json:"name"`
		Summary  string `json:"summary"`
		Schedule string `json:"schedule"`
		Retain   string `json:"retain"`
		Paused   bool   `json:"paused"`
		Store    string `json:"store"`
		Target   string `json:"target"`
		FixedKey bool   `json:"fixed_key"`
	}{
		Name:     job.Name,
		Summary:  job.Summary,
		Schedule: job.Schedule,
		Retain:   job.Retain,
		Paused:   job.Paused,
		Target:   job.TargetUUID,
		Store:    job.StoreUUID,
		FixedKey: job.FixedKey,
	}
	if err := c.post(fmt.Sprintf("/v2/tenants/%s/jobs", parent.UUID), in, &out); err != nil {
		return nil, err
	}
	fixupJobResponse(out)
	return out, nil
}

func (c *Client) UpdateJob(parent *Tenant, job *Job) (*Job, error) {
	in := struct {
		Name     string `json:"name,omitempty"`
		Summary  string `json:"summary,omitempty"`
		Schedule string `json:"schedule,omitempty"`
		Retain   string `json:"retain,omitempty"`
		Store    string `json:"store,omitempty"`
		Target   string `json:"target,omitempty"`
		FixedKey bool   `json:"fixed_key"`
	}{
		Name:     job.Name,
		Summary:  job.Summary,
		Schedule: job.Schedule,
		Retain:   job.Retain,
		Target:   job.TargetUUID,
		Store:    job.StoreUUID,
		FixedKey: job.FixedKey,
	}
	if err := c.put(fmt.Sprintf("/v2/tenants/%s/jobs/%s", parent.UUID, job.UUID), in, nil); err != nil {
		return nil, err
	}
	return c.GetJob(parent, job.UUID)
}

func (c *Client) DeleteJob(parent *Tenant, in *Job) (Response, error) {
	var out Response
	return out, c.delete(fmt.Sprintf("/v2/tenants/%s/jobs/%s", parent.UUID, in.UUID), &out)
}

func (c *Client) PauseJob(parent *Tenant, job *Job) (Response, error) {
	var out Response
	return out, c.post(fmt.Sprintf("/v2/tenants/%s/jobs/%s/pause", parent.UUID, job.UUID), nil, &out)
}

func (c *Client) UnpauseJob(parent *Tenant, job *Job) (Response, error) {
	var out Response
	return out, c.post(fmt.Sprintf("/v2/tenants/%s/jobs/%s/unpause", parent.UUID, job.UUID), nil, &out)
}

func (c *Client) RunJob(parent *Tenant, job *Job) (Response, error) {
	var out Response
	return out, c.post(fmt.Sprintf("/v2/tenants/%s/jobs/%s/run", parent.UUID, job.UUID), nil, &out)
}

func (j Job) Status() string {
	if j.Paused {
		return "paused"
	}
	return j.LastStatus
}
