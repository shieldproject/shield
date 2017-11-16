package shield

import (
	"fmt"
)

type AuthToken struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	Session   string `json:"session"`
	CreatedAt int64  `json:"created_at"`
	LastSeen  int64  `json:"last_seen"`
}

func (c *Client) ListAuthTokens() ([]*AuthToken, error) {
	l := make([]*AuthToken, 0)
	err := c.get("/v2/auth/tokens", &l)
	return l, err
}

func (c *Client) CreateAuthToken(t *AuthToken) (*AuthToken, error) {
	return t, c.post("/v2/auth/tokens", t, t)
}

func (c *Client) RevokeAuthToken(t *AuthToken) error {
	return c.delete(fmt.Sprintf("/v2/auth/tokens/%s", t.UUID), nil)
}
