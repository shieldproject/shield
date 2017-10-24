package shield

import (
	"fmt"

	qs "github.com/jhunt/go-querytron"
	"github.com/pborman/uuid"
)

type Policy struct {
	UUID    string `json:"uuid,omitempty"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Expires int    `json:"expires"`
}

type PolicyFilter struct {
	Fuzzy bool   `qs:"exact:f:t"`
	Name  string `qs:"name"`
	Used  *bool  `qs:"unused:f:t"`
}

func fixupPolicyResponse(p *Policy) {
	if p.Expires >= 86400 {
		p.Expires /= 86400
	}
}

func fixupPolicyRequest(p *Policy) {
	if p.Expires < 86400 {
		p.Expires *= 86400
	}
}

func (c *Client) ListPolicies(parent *Tenant, filter *PolicyFilter) ([]*Policy, error) {
	u := qs.Generate(filter).Encode()
	var out []*Policy
	if err := c.get(fmt.Sprintf("/v2/tenants/%s/policies?%s", parent.UUID, u), &out); err != nil {
		return nil, err
	}
	for _, p := range out {
		fixupPolicyResponse(p)
	}
	return out, nil
}

func (c *Client) FindPolicy(tenant *Tenant, q string, fuzzy bool) (*Policy, error) {
	if uuid.Parse(q) != nil {
		return c.GetPolicy(tenant, q)
	}

	l, err := c.ListPolicies(tenant, &PolicyFilter{
		Name:  q,
		Fuzzy: fuzzy,
	})
	if err != nil {
		return nil, err
	}

	if len(l) == 0 {
		return nil, fmt.Errorf("no matching policy found")
	}
	if len(l) > 1 {
		return nil, fmt.Errorf("multiple matching policies found")
	}

	return c.GetPolicy(tenant, l[0].UUID)
}

func (c *Client) GetPolicy(parent *Tenant, uuid string) (*Policy, error) {
	var out *Policy
	if err := c.get(fmt.Sprintf("/v2/tenants/%s/policies/%s", parent.UUID, uuid), &out); err != nil {
		return nil, err
	}
	fixupPolicyResponse(out)
	return out, nil
}

func (c *Client) CreatePolicy(parent *Tenant, in *Policy) (*Policy, error) {
	fixupPolicyRequest(in)
	var out *Policy
	if err := c.post(fmt.Sprintf("/v2/tenants/%s/policies", parent.UUID), in, &out); err != nil {
		return nil, err
	}
	fixupPolicyResponse(out)
	return out, nil
}

func (c *Client) UpdatePolicy(parent *Tenant, in *Policy) (*Policy, error) {
	fixupPolicyRequest(in)
	var out *Policy
	if err := c.patch(fmt.Sprintf("/v2/tenants/%s/policies/%s", parent.UUID, in.UUID), in, &out); err != nil {
		return nil, err
	}
	fixupPolicyResponse(out)
	return out, nil
}

func (c *Client) DeletePolicy(parent *Tenant, in *Policy) (Response, error) {
	var out Response
	return out, c.delete(fmt.Sprintf("/v2/tenants/%s/policies/%s", parent.UUID, in.UUID), &out)
}

func (c *Client) ListPolicyTemplates(filter *PolicyFilter) ([]*Policy, error) {
	u := qs.Generate(filter).Encode()
	var out []*Policy
	if err := c.get(fmt.Sprintf("/v2/global/policies?%s", u), &out); err != nil {
		return nil, err
	}
	for _, p := range out {
		fixupPolicyResponse(p)
	}
	return out, nil
}

func (c *Client) FindPolicyTemplate(q string, fuzzy bool) (*Policy, error) {
	if uuid.Parse(q) != nil {
		return c.GetPolicyTemplate(q)
	}

	l, err := c.ListPolicyTemplates(&PolicyFilter{
		Name:  q,
		Fuzzy: fuzzy,
	})
	if err != nil {
		return nil, err
	}

	if len(l) == 0 {
		return nil, fmt.Errorf("no matching policy template found")
	}
	if len(l) > 1 {
		return nil, fmt.Errorf("multiple matching policy templates found")
	}

	return c.GetPolicyTemplate(l[0].UUID)
}

func (c *Client) GetPolicyTemplate(uuid string) (*Policy, error) {
	var out *Policy
	if err := c.get(fmt.Sprintf("/v2/global/policies/%s", uuid), &out); err != nil {
		return nil, err
	}
	fixupPolicyResponse(out)
	return out, nil
}

func (c *Client) CreatePolicyTemplate(in *Policy) (*Policy, error) {
	fixupPolicyRequest(in)
	var out *Policy
	if err := c.post(fmt.Sprintf("/v2/global/policies"), in, &out); err != nil {
		return nil, err
	}
	fixupPolicyResponse(out)
	return out, nil
}

func (c *Client) UpdatePolicyTemplate(in *Policy) (*Policy, error) {
	fixupPolicyRequest(in)
	var out *Policy
	if err := c.put(fmt.Sprintf("/v2/global/policies/%s", in.UUID), in, &out); err != nil {
		return nil, err
	}
	fixupPolicyResponse(out)
	return out, nil
}

func (c *Client) DeletePolicyTemplate(in *Policy) (Response, error) {
	var out Response
	return out, c.delete(fmt.Sprintf("/v2/global/policies/%s", in.UUID), &out)
}
