package shield

import (
	"fmt"
	"strings"

	qs "github.com/jhunt/go-querytron"
	"github.com/pborman/uuid"
)

type Tenant struct {
	UUID string `json:"uuid,omitempty"`
	Name string `json:"name"`

	Members []struct {
		UUID    string `json:"uuid,omitempty"`
		Fuzzy   bool   `json:"exact:f:t"`
		Name    string `json:"name"`
		Account string `json:"account"`
		Backend string `json:"backend"`
		Role    string `json:"role"`
	} `json:"members"`
}

type TenantFilter struct {
	UUID  string `qs:"uuid"`
	Fuzzy bool   `qs:"exact:f:t"`
	Name  string `qs:"name"`
}

func (c *Client) ListTenants(filter *TenantFilter) ([]*Tenant, error) {
	u := qs.Generate(filter).Encode()
	var out []*Tenant
	return out, c.get(fmt.Sprintf("/v2/tenants?%s", u), &out)
}

func (c *Client) GetMyTenants() ([]*Tenant, error) {
	id, err := c.AuthID()
	if err != nil {
		return nil, err
	}

	if id.User.SysRole != "" {
		return c.ListTenants(nil)
	}

	l := make([]*Tenant, len(id.Tenants))
	for i := range id.Tenants {
		l[i] = &Tenant{
			UUID: id.Tenants[i].UUID,
			Name: id.Tenants[i].Name,
		}
	}

	return l, nil
}

func (c *Client) FindMyTenant(q string, fuzzy bool) (*Tenant, error) {
	tenants, err := c.GetMyTenants()
	if err != nil {
		return nil, err
	}

	q = strings.ToLower(q)
	l := make([]*Tenant, 0)
	for _, tenant := range tenants {
		if !fuzzy && (tenant.UUID == q || strings.ToLower(tenant.Name) == q) {
			l = append(l, tenant)
		} else if fuzzy && strings.Contains(strings.ToLower(tenant.Name), q) {
			l = append(l, tenant)
		} else if fuzzy && strings.HasPrefix(strings.ToLower(tenant.UUID), q) {
			l = append(l, tenant)
		}
	}

	if len(l) == 0 {
		if fuzzy {
			return nil, fmt.Errorf("no tenants matching '*%s*'", q)
		}
		return nil, fmt.Errorf("no tenant named '%s'", q)
	}

	if len(l) > 1 {
		if fuzzy {
			return nil, fmt.Errorf("multiple tenants matching '*%s*'", q)
		}
		return nil, fmt.Errorf("multiple tenants named '%s'", q)
	}

	return l[0], nil
}

func (c *Client) FindTenant(q string, fuzzy bool) (*Tenant, error) {
	if uuid.Parse(q) != nil {
		return c.GetTenant(q)
	}

	l, err := c.ListTenants(&TenantFilter{
		UUID:  q,
		Name:  q,
		Fuzzy: fuzzy,
	})
	if err != nil {
		return nil, err
	}

	if len(l) == 0 {
		return nil, fmt.Errorf("no matching tenant found")
	}
	if len(l) > 1 {
		return nil, fmt.Errorf("multiple matching tenants found")
	}

	return c.GetTenant(l[0].UUID)
}

func (c *Client) GetTenant(uuid string) (*Tenant, error) {
	var out *Tenant
	return out, c.get(fmt.Sprintf("/v2/tenants/%s", uuid), &out)
}

func (c *Client) CreateTenant(in *Tenant) (*Tenant, error) {
	var out *Tenant
	return out, c.post("/v2/tenants", in, &out)
}

func (c *Client) UpdateTenant(in *Tenant) (*Tenant, error) {
	var out *Tenant
	return out, c.patch(fmt.Sprintf("/v2/tenants/%s", in.UUID), in, &out)
}

func (c *Client) DeleteTenant(in *Tenant, recurse bool) (Response, error) {
	r := "f"
	if recurse {
		r = "t"
	}
	var out Response
	return out, c.delete(fmt.Sprintf("/v2/tenants/%s?recurse=%s", in.UUID, r), &out)
}
