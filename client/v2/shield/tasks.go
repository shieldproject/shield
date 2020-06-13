package shield

import (
	"fmt"

	qs "github.com/jhunt/go-querytron"
	"github.com/pborman/uuid"
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
	Store  string `qs:"store"`
	Type   string `qs:"type"`
	Before int64  `qs:"before"`
}

func fixupTaskResponse(p *Task) {
}

func fixupTaskRequest(p *Task) {
}

func (c *Client) ListTasks(filter *TaskFilter) ([]*Task, error) {
	u := qs.Generate(filter).Encode()
	url := fmt.Sprintf("/v2/tasks?%s", u)

	var out []*Task
	if err := c.get(url, &out); err != nil {
		return nil, err
	}
	for _, p := range out {
		fixupTaskResponse(p)
	}
	return out, nil
}

func (c *Client) FindTask(q string, fuzzy bool) (*Task, error) {
	if uuid.Parse(q) != nil {
		return c.GetTask(q)
	}

	l, err := c.ListTasks(&TaskFilter{
		UUID:  q,
		Fuzzy: fuzzy,
	})
	if err != nil {
		return nil, err
	}

	if len(l) == 0 {
		return nil, fmt.Errorf("no matching task found")
	}
	if len(l) > 1 {
		return nil, fmt.Errorf("multiple matching tasks found")
	}

	return c.GetTask(l[0].UUID)
}

func (c *Client) GetTask(uuid string) (*Task, error) {
	url := fmt.Sprintf("/v2/tasks/%s", uuid)

	var out *Task
	if err := c.get(url, &out); err != nil {
		return nil, err
	}
	fixupTaskResponse(out)
	return out, nil
}

func (c *Client) CreateTask(in *Task) (*Task, error) {
	fixupTaskRequest(in)
	var out *Task
	if err := c.post("/v2/tasks", in, &out); err != nil {
		return nil, err
	}
	fixupTaskResponse(out)
	return out, nil
}

func (c *Client) UpdateTask(in *Task) (*Task, error) {
	fixupTaskRequest(in)
	var out *Task
	if err := c.put(fmt.Sprintf("/v2/tasks/%s", in.UUID), in, &out); err != nil {
		return nil, err
	}
	fixupTaskResponse(out)
	return out, nil
}

func (c *Client) CancelTask(in *Task) (Response, error) {
	var out Response
	if err := c.delete(fmt.Sprintf("/v2/tasks/%s", in.UUID), &out); err != nil {
		return out, err
	}
	return out, nil
}
