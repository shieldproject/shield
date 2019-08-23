package shield

import (
	"fmt"

	qs "github.com/jhunt/go-querytron"
	"github.com/pborman/uuid"
)

type Agent struct {
	UUID          string `json:"uuid"`
	Name          string `json:"name"`
	Address       string `json:"address"`
	Version       string `json:"version"`
	Status        string `json:"status"`
	Hidden        bool   `json:"hidden"`
	LastSeenAt    int64  `json:"last_seen_at"`
	LastCheckedAt int64  `json:"last_checked_at"`
	LastError     string `json:"last_error"`

	Metadata map[string]interface{} `json:"metadata"`

	Problems []string `json:"problems"`
}

type AgentFilter struct {
	UUID   string `qs:"uuid"`
	Fuzzy  bool   `qs:"exact:f:t"`
	Hidden *bool  `qs:"hidden:t:f"`
}

func (c *Client) ListAgents(filter *AgentFilter) ([]*Agent, error) {
	u := qs.Generate(filter).Encode()
	var out struct {
		Agents   []*Agent            `json:"agents"`
		Problems map[string][]string `json:"problems"`
	}
	if err := c.get(fmt.Sprintf("/v2/agents?%s", u), &out); err != nil {
		return nil, err
	}
	for _, agent := range out.Agents {
		if pp, ok := out.Problems[agent.UUID]; ok {
			agent.Problems = pp
		}
	}
	return out.Agents, nil
}

func (c *Client) FindAgent(q string, fuzzy bool) (*Agent, error) {
	if uuid.Parse(q) != nil {
		return c.GetAgent(q)
	}

	l, err := c.ListAgents(&AgentFilter{
		UUID:  q,
		Fuzzy: fuzzy,
	})
	if err != nil {
		return nil, err
	}

	if len(l) == 0 {
		return nil, fmt.Errorf("no matching agent found")
	}
	if len(l) > 1 {
		return nil, fmt.Errorf("multiple matching agents found")
	}

	return c.GetAgent(l[0].UUID)
}

func (c *Client) GetAgent(uuid string) (*Agent, error) {
	var out struct {
		Agent    *Agent                 `json:"agent"`
		Metadata map[string]interface{} `json:"metadata"`
		Problems []string               `json:"problems"`
	}
	if err := c.get(fmt.Sprintf("/v2/agents/%s", uuid), &out); err != nil {
		return nil, err
	}
	out.Agent.Problems = out.Problems
	out.Agent.Metadata = out.Metadata
	return out.Agent, nil
}

func (c *Client) HideAgent(in *Agent) (Response, error) {
	var out Response
	return out, c.post(fmt.Sprintf("/v2/agents/%s/hide", in.UUID), nil, &out)
}

func (c *Client) ShowAgent(in *Agent) (Response, error) {
	var out Response
	return out, c.post(fmt.Sprintf("/v2/agents/%s/show", in.UUID), nil, &out)
}

func (c *Client) DeleteAgent(in *Agent) (Response, error) {
	var out Response
	return out, c.delete(fmt.Sprintf("/v2/agents/%s", in.UUID), &out)
}
