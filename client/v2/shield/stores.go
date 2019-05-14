package shield

import (
	"fmt"

	qs "github.com/jhunt/go-querytron"
	"github.com/pborman/uuid"
)

type Store struct {
	UUID      string `json:"uuid,omitempty"`
	Name      string `json:"name"`
	Summary   string `json:"summary"`
	Plugin    string `json:"plugin"`
	Agent     string `json:"agent"`
	Healthy   bool   `json:"healthy"`
	Threshold int64  `json:"threshold"`

	Config map[string]interface{} `json:"config"`
}

type StoreFilter struct {
	UUID   string `qs:"uuid"`
	Fuzzy  bool   `qs:"exact:f:t"`
	Name   string `qs:"name"`
	Plugin string `qs:"plugin"`
	Used   *bool  `qs:"unused:f:t"`
}

func fixupStoreResponse(p *Store) {
}

func fixupStoreRequest(p *Store) {
}

func (c *Client) ListStores(parent *Tenant, filter *StoreFilter) ([]*Store, error) {
	u := qs.Generate(filter).Encode()
	var out []*Store
	if err := c.get(fmt.Sprintf("/v2/tenants/%s/stores?%s", parent.UUID, u), &out); err != nil {
		return nil, err
	}
	for _, p := range out {
		fixupStoreResponse(p)
	}
	return out, nil
}

func (c *Client) GetStore(parent *Tenant, uuid string) (*Store, error) {
	var out *Store
	if err := c.get(fmt.Sprintf("/v2/tenants/%s/stores/%s", parent.UUID, uuid), &out); err != nil {
		return nil, err
	}
	fixupStoreResponse(out)
	return out, nil
}

func (c *Client) FindStore(tenant *Tenant, q string, fuzzy bool) (*Store, error) {
	if uuid.Parse(q) != nil {
		return c.GetStore(tenant, q)
	}

	l, err := c.ListStores(tenant, &StoreFilter{
		UUID:  q,
		Name:  q,
		Fuzzy: fuzzy,
	})
	if err != nil {
		return nil, err
	}

	if len(l) == 0 {
		return nil, fmt.Errorf("no matching store found")
	}
	if len(l) > 1 {
		return nil, fmt.Errorf("multiple matching stores found")
	}

	return c.GetStore(tenant, l[0].UUID)
}

func (c *Client) CreateStore(parent *Tenant, in *Store) (*Store, error) {
	fixupStoreRequest(in)
	var out *Store
	if err := c.post(fmt.Sprintf("/v2/tenants/%s/stores", parent.UUID), in, &out); err != nil {
		return nil, err
	}
	fixupStoreResponse(out)
	return out, nil
}

func (c *Client) UpdateStore(parent *Tenant, in *Store) (*Store, error) {
	fixupStoreRequest(in)
	var out *Store
	if err := c.put(fmt.Sprintf("/v2/tenants/%s/stores/%s", parent.UUID, in.UUID), in, &out); err != nil {
		return nil, err
	}
	fixupStoreResponse(out)
	return out, nil
}

func (c *Client) DeleteStore(parent *Tenant, in *Store) (Response, error) {
	var out Response
	return out, c.delete(fmt.Sprintf("/v2/tenants/%s/stores/%s", parent.UUID, in.UUID), &out)
}

func (c *Client) ListGlobalStores(filter *StoreFilter) ([]*Store, error) {
	u := qs.Generate(filter).Encode()
	var out []*Store
	if err := c.get(fmt.Sprintf("/v2/global/stores?%s", u), &out); err != nil {
		return nil, err
	}
	for _, p := range out {
		fixupStoreResponse(p)
	}
	return out, nil
}

func (c *Client) GetGlobalStore(uuid string) (*Store, error) {
	var out *Store
	if err := c.get(fmt.Sprintf("/v2/global/stores/%s", uuid), &out); err != nil {
		return nil, err
	}
	fixupStoreResponse(out)
	return out, nil
}

func (c *Client) FindGlobalStore(q string, fuzzy bool) (*Store, error) {
	if uuid.Parse(q) != nil {
		return c.GetGlobalStore(q)
	}

	l, err := c.ListGlobalStores(&StoreFilter{
		UUID:  q,
		Name:  q,
		Fuzzy: fuzzy,
	})
	if err != nil {
		return nil, err
	}

	if len(l) == 0 {
		return nil, fmt.Errorf("no matching store found")
	}
	if len(l) > 1 {
		return nil, fmt.Errorf("multiple matching stores found")
	}

	return c.GetGlobalStore(l[0].UUID)
}

func (c *Client) CreateGlobalStore(in *Store) (*Store, error) {
	fixupStoreRequest(in)
	var out *Store
	if err := c.post(fmt.Sprintf("/v2/global/stores"), in, &out); err != nil {
		return nil, err
	}
	fixupStoreResponse(out)
	return out, nil
}

func (c *Client) UpdateGlobalStore(in *Store) (*Store, error) {
	fixupStoreRequest(in)
	var out *Store
	if err := c.put(fmt.Sprintf("/v2/global/stores/%s", in.UUID), in, &out); err != nil {
		return nil, err
	}
	fixupStoreResponse(out)
	return out, nil
}

func (c *Client) DeleteGlobalStore(in *Store) (Response, error) {
	var out Response
	return out, c.delete(fmt.Sprintf("/v2/global/stores/%s", in.UUID), &out)
}

func (c *Client) FindUsableStore(tenant *Tenant, q string, fuzzy bool) (*Store, error) {
	store, err := c.FindStore(tenant, q, fuzzy)
	if err == nil {
		return store, nil
	}
	if store, _ := c.FindGlobalStore(q, fuzzy); store != nil {
		return store, nil
	}
	return nil, err
}
