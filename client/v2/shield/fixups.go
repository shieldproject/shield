package shield

import (
	"fmt"

	qs "github.com/jhunt/go-querytron"
)

type Fixup struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Summary   string `json:"summary"`
	CreatedAt int64  `json:"created_at"`
	AppliedAt int64  `json:"applied_at"`
}

type FixupFilter struct {
}

func (c *Client) ListFixups(filter *FixupFilter) ([]*Fixup, error) {
	u := qs.Generate(filter).Encode()
	var out []*Fixup
	return out, c.get(fmt.Sprintf("/v2/fixups?%s", u), &out)
}

func (c *Client) GetFixup(id string) (*Fixup, error) {
	var out *Fixup
	return out, c.get(fmt.Sprintf("/v2/fixups/%s", id), &out)
}

func (c *Client) ApplyFixup(id string) (Response, error) {
	var out Response
	return out, c.post(fmt.Sprintf("/v2/fixups/%s/apply", id), nil, &out)
}
