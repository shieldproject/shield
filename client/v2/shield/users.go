package shield

import (
	"fmt"

	qs "github.com/jhunt/go-querytron"
	"github.com/pborman/uuid"
)

type User struct {
	UUID     string `json:"uuid,omitempty"`
	Name     string `json:"name"`
	Account  string `json:"account"`
	SysRole  string `json:"sysrole"`
	Password string `json:"password,omitempty"`

	Tenants []struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
		Role string `json:"role"`
	} `json:"tenants"`
}

type UserFilter struct {
	UUID    string `qs:"uuid"`
	Fuzzy   bool   `qs:"exact:f:t"`
	Account string `qs:"account"`
	SysRole string `qs:"sysrole"`
}

func fixupUserResponse(p *User) {
}

func fixupUserRequest(p *User) {
}

func (c *Client) ListUsers(filter *UserFilter) ([]*User, error) {
	u := qs.Generate(filter).Encode()

	var out []*User
	if err := c.get(fmt.Sprintf("/v2/auth/local/users?%s", u), &out); err != nil {
		return nil, err
	}
	for _, p := range out {
		fixupUserResponse(p)
	}
	return out, nil
}

func (c *Client) FindUser(q string, fuzzy bool) (*User, error) {
	if uuid.Parse(q) != nil {
		return c.GetUser(q)
	}

	l, err := c.ListUsers(&UserFilter{
		Account: q,
		Fuzzy:   fuzzy,
	})
	if err != nil {
		return nil, err
	}

	if len(l) == 0 {
		return nil, fmt.Errorf("no matching user found")
	}
	if len(l) > 1 {
		return nil, fmt.Errorf("multiple matching users found")
	}

	return c.GetUser(l[0].UUID)
}

func (c *Client) GetUser(uuid string) (*User, error) {
	var out *User
	if err := c.get(fmt.Sprintf("/v2/auth/local/users/%s", uuid), &out); err != nil {
		return nil, err
	}
	fixupUserResponse(out)
	return out, nil
}

func (c *Client) CreateUser(in *User) (*User, error) {
	fixupUserRequest(in)
	var out *User
	if err := c.post("/v2/auth/local/users", in, &out); err != nil {
		return nil, err
	}
	fixupUserResponse(out)
	return out, nil
}

func (c *Client) UpdateUser(in *User) (*User, error) {
	fixupUserRequest(in)
	var out *User
	if err := c.patch(fmt.Sprintf("/v2/auth/local/users/%s", in.UUID), in, &out); err != nil {
		return nil, err
	}
	fixupUserResponse(out)
	return out, nil
}

func (c *Client) DeleteUser(in *User) (Response, error) {
	var out Response
	return out, c.delete(fmt.Sprintf("/v2/auth/local/users/%s", in.UUID), &out)
}
