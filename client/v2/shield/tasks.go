package shield

import (
	"fmt"

	qs "github.com/jhunt/go-querytron"
)

type Task struct {
	UUID        string `json:"uuid,omitempty"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Owner       string `json:"owner"`
	StartedAt   int64  `json:"started_at"`
	StoppedAt   int64  `json:"stopped_at"`
	RequestedAt int64  `json:"requested_at"`
	Log         string `json:"log"`
	OK          bool   `json:"ok"`
	Notes       string `json:"notes"`
	Clear       string `json:"clear"`
	JobUUID     string `json:"job_uuid"`
	ArchiveUUID string `json:"archive_uuid"`
}

type TaskFilter struct {
	UUID   string `qs:"uuid"`
	Fuzzy  bool   `qs:"exact:f:t"`
	Status string `qs:"status"`
	Active *bool  `qs:"active:t:f"`
	Debug  *bool  `qs:"debug:t:f"`
	Limit  *int   `qs:"limit"`
	Target string `qs:"target"`
	Type   string `qs:"type"`
	Before int64  `qs:"before"`
}

func fixupTaskResponse(p *Task) {
}

func fixupTaskRequest(p *Task) {
}

func (c *Client) ListTasks(parent *Tenant, filter *TaskFilter) ([]*Task, error) {
	u := qs.Generate(filter).Encode()
	url := fmt.Sprintf("/v2/tasks?%s", u)
	if parent != nil {
		url = fmt.Sprintf("/v2/tenants/%s/tasks?%s", parent.UUID, u)
	}

	var out []*Task
	if err := c.get(url, &out); err != nil {
		return nil, err
	}
	for _, p := range out {
		fixupTaskResponse(p)
	}
	return out, nil
}

func (c *Client) GetTask(parent *Tenant, uuid string) (*Task, error) {
	url := fmt.Sprintf("/v2/tasks/%s", uuid)
	if parent != nil {
		url = fmt.Sprintf("/v2/tenants/%s/tasks/%s", parent.UUID, uuid)
	}

	var out *Task
	if err := c.get(url, &out); err != nil {
		return nil, err
	}
	fixupTaskResponse(out)
	return out, nil
}

func (c *Client) CreateTask(parent *Tenant, in *Task) (*Task, error) {
	fixupTaskRequest(in)
	var out *Task
	if err := c.post(fmt.Sprintf("/v2/tenants/%s/tasks", parent.UUID), in, &out); err != nil {
		return nil, err
	}
	fixupTaskResponse(out)
	return out, nil
}

func (c *Client) UpdateTask(parent *Tenant, in *Task) (*Task, error) {
	fixupTaskRequest(in)
	var out *Task
	if err := c.put(fmt.Sprintf("/v2/tenants/%s/tasks/%s", parent.UUID, in.UUID), in, &out); err != nil {
		return nil, err
	}
	fixupTaskResponse(out)
	return out, nil
}

func (c *Client) CancelTask(parent *Tenant, in *Task) (Response, error) {
	var out Response
	if err := c.delete(fmt.Sprintf("/v2/tenants/%s/tasks/%s", parent.UUID, in.UUID), &out); err != nil {
		return out, err
	}
	return out, nil
}
