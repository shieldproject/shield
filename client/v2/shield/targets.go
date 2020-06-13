package shield

import (
	"fmt"

	qs "github.com/jhunt/go-querytron"
	"github.com/pborman/uuid"
)

type Target struct {
	UUID    string `json:"uuid,omitempty"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Plugin  string `json:"plugin"`
	Agent   string `json:"agent"`

	Config map[string]interface{} `json:"config"`
}

type TargetFilter struct {
	UUID   string `qs:"uuid"`
	Fuzzy  bool   `qs:"exact:f:t"`
	Name   string `qs:"name"`
	Plugin string `qs:"plugin"`
	Used   *bool  `qs:"unused:f:t"`
}

func fixupTargetResponse(p *Target) {
}

func fixupTargetRequest(p *Target) {
}

func (c *Client) ListTargets(filter *TargetFilter) ([]*Target, error) {
	u := qs.Generate(filter).Encode()

	var out []*Target
	if err := c.get(fmt.Sprintf("/v2/targets?%s", u), &out); err != nil {
		return nil, err
	}
	for _, p := range out {
		fixupTargetResponse(p)
	}
	return out, nil
}

func (c *Client) FindTarget(q string, fuzzy bool) (*Target, error) {
	if uuid.Parse(q) != nil {
		return c.GetTarget(q)
	}

	l, err := c.ListTargets(&TargetFilter{
		UUID:  q,
		Name:  q,
		Fuzzy: fuzzy,
	})
	if err != nil {
		return nil, err
	}

	if len(l) == 0 {
		return nil, fmt.Errorf("no matching target found")
	}
	if len(l) > 1 {
		return nil, fmt.Errorf("multiple matching targets found")
	}

	return c.GetTarget(l[0].UUID)
}

func (c *Client) GetTarget(uuid string) (*Target, error) {
	var out *Target
	if err := c.get(fmt.Sprintf("/v2/targets/%s", uuid), &out); err != nil {
		return nil, err
	}
	fixupTargetResponse(out)
	return out, nil
}

func (c *Client) CreateTarget(in *Target) (*Target, error) {
	fixupTargetRequest(in)
	var out *Target
	if err := c.post("/v2/targets", in, &out); err != nil {
		return nil, err
	}
	fixupTargetResponse(out)
	return out, nil
}

func (c *Client) UpdateTarget(in *Target) (*Target, error) {
	fixupTargetRequest(in)
	var out *Target
	if err := c.put(fmt.Sprintf("/v2/targets/%s", in.UUID), in, &out); err != nil {
		return nil, err
	}
	fixupTargetResponse(out)
	return out, nil
}

func (c *Client) DeleteTarget(in *Target) (Response, error) {
	var out Response
	return out, c.delete(fmt.Sprintf("/v2/targets/%s", in.UUID), &out)
}
