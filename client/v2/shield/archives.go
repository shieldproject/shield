package shield

import (
	"fmt"

	qs "github.com/jhunt/go-querytron"
	"github.com/pborman/uuid"
)

type Archive struct {
	UUID   string `json:"uuid,omitempty"`
	Key    string `json:"key"`
	Status string `json:"status"`
	Notes  string `json:"notes"`

	Target *Target `json:"target,omitempty"`

	EncryptionType string `json:"encryption_type"`
	Size           int64  `json:"size"`
}

type ArchiveFilter struct {
	UUID   string `qs:"uuid"`
	Fuzzy  bool   `qs:"exact:f:t"`
	Target string `qs:"target"`
	Status string `qs:"status"`
	//Before string `qs:"before"`
	//After string `qs:"after"`
	Limit *int `qs:"limit"`
}

func fixupArchiveResponse(p *Archive) {
}

func fixupArchiveRequest(p *Archive) {
}

func (c *Client) ListArchives(filter *ArchiveFilter) ([]*Archive, error) {
	u := qs.Generate(filter).Encode()
	var out []*Archive
	if err := c.get(fmt.Sprintf("/v2/archives?%s", u), &out); err != nil {
		return nil, err
	}
	for _, p := range out {
		fixupArchiveResponse(p)
	}
	return out, nil
}

func (c *Client) FindArchive(q string, fuzzy bool) (*Archive, error) {
	if uuid.Parse(q) != nil {
		return c.GetArchive(q)
	}

	l, err := c.ListArchives(&ArchiveFilter{
		UUID:  q,
		Fuzzy: fuzzy,
	})
	if err != nil {
		return nil, err
	}

	if len(l) == 0 {
		return nil, fmt.Errorf("no matching archive found")
	}
	if len(l) > 1 {
		return nil, fmt.Errorf("multiple matching archives found")
	}

	return c.GetArchive(l[0].UUID)
}

func (c *Client) GetArchive(uuid string) (*Archive, error) {
	var out *Archive
	if err := c.get(fmt.Sprintf("/v2/archives/%s", uuid), &out); err != nil {
		return nil, err
	}
	fixupArchiveResponse(out)
	return out, nil
}

func (c *Client) CreateArchive(in *Archive) (*Archive, error) {
	fixupArchiveRequest(in)
	var out *Archive
	if err := c.post("/v2/archives", in, &out); err != nil {
		return nil, err
	}
	fixupArchiveResponse(out)
	return out, nil
}

func (c *Client) UpdateArchive(in *Archive) (*Archive, error) {
	fixupArchiveRequest(in)
	var out *Archive
	if err := c.put(fmt.Sprintf("/v2/archives/%s", in.UUID), in, &out); err != nil {
		return nil, err
	}
	fixupArchiveResponse(out)
	return out, nil
}

func (c *Client) DeleteArchive(in *Archive) (Response, error) {
	var out Response
	return out, c.delete(fmt.Sprintf("/v2/archives/%s", in.UUID), &out)
}

func (c *Client) RestoreArchive(a *Archive, t *Target) (*Task, error) {
	var out Task
	var filter struct {
		Target string `json:"target"`
	}

	if t != nil {
		filter.Target = t.UUID
	}

	return &out, c.post(fmt.Sprintf("/v2/archives/%s/restore",
		a.UUID), filter, &out)
}
