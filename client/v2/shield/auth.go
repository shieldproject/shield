package shield

import (
	"fmt"
)

type AuthMethod interface {
	Authenticate(*Client) (bool, error)
}

type LocalAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (auth *LocalAuth) Authenticate(c *Client) (bool, error) {
	r := &Response{}
	err := c.post("/v2/auth/login", auth, r)
	return err == nil, err
}

type TokenAuth struct {
	Token string
}

func (auth *TokenAuth) Authenticate(c *Client) (bool, error) {
	c.Debugf("setting session id to '%s'", auth.Token)
	c.Session = auth.Token
	return true, nil
}

func (c *Client) Authenticate(auth AuthMethod) error {
	ok, err := auth.Authenticate(c)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("authentication failed")
	}

	id, err := c.AuthID()
	if err != nil {
		return err
	}
	if id.Unauthenticated {
		return fmt.Errorf("Authentication failed")
	}
	return nil
}

func (c *Client) Logout() error {
	return c.get("/v2/auth/logout", nil)
}

type AuthID struct {
	Unauthenticated bool `json:"unauthenticated,omitempty"`

	User struct {
		Name    string `json:"name"`
		Account string `json:"account"`
		Backend string `json:"backend"`
		SysRole string `json:"sysrole"`
	} `json:"user"`

	Tenants []struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
		Role string `json:"role"`
	} `json:"tenants"`

	Tenant struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
		Role string `json:"role"`
	} `json:"tenant"`

	Is struct {
		System struct {
			Admin    bool `json:"admin"`
			Manager  bool `json:"manager"`
			Engineer bool `json:"engineer"`
		} `json:"system"`
		Tenants map[string]struct {
			Admin    bool `json:"admin"`
			Engineer bool `json:"engineer"`
			Operator bool `json:"operator"`
		} `json:"tenant"`
	} `json:"is"`
}

func (c *Client) AuthID() (*AuthID, error) {
	out := &AuthID{}
	return out, c.get("/v2/auth/id", out)
}
