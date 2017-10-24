package shield

import (
	"fmt"
)

type AuthProvider struct {
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
	Type       string `json:"type"`

	WebEntry string `json:"web_entry"`
	CLIEntry string `json:"cli_entry"`
	Redirect string `json:"redirect"`

	Properties map[string]interface{} `json:"properties,omitempty"`
}

func (c *Client) AuthProviders() ([]*AuthProvider, error) {
	l := make([]*AuthProvider, 0)
	return l, c.get("/v2/auth/providers", &l)
}

func (c *Client) AuthProviderAnonymous(id string) (*AuthProvider, error) {
	l, err := c.AuthProviders()
	if err != nil {
		return nil, err
	}

	for _, provider := range l {
		if id == provider.Identifier {
			return provider, nil
		}
	}

	return nil, fmt.Errorf("no such authentication provider '%s'", id)
}

func (c *Client) AuthProvider(id string) (*AuthProvider, error) {
	p := AuthProvider{}
	return &p, c.get("/v2/auth/providers/"+id, &p)
}
