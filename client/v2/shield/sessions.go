package shield

import (
	"fmt"

	qs "github.com/jhunt/go-querytron"
)

type Session struct {
	UUID           string `json:"uuid"`
	UserUUID       string `json:"user_uuid"`
	CreatedAt      int64  `json:"created_at"`
	LastSeen       int64  `json:"last_seen_at"`
	Token          string `json:"token_uuid"`
	Name           string `json:"name"`
	IP             string `json:"ip_addr"`
	UserAgent      string `json:"user_agent"`
	UserAccount    string `json:"user_account"`
	CurrentSession bool   `json:"current_session"`
}

type SessionFilter struct {
	Name       string `qs:"name"`
	UUID       string
	UserUUID   string `qs:"user_uuid"`
	IP         string `qs:"ip_addr"`
	Limit      int    `qs:"limit"`
	ExactMatch bool   `qs:"exact:f:t"`
	IsToken    bool   `qs:"is_token"`
}

func fixupSessionResponse(p *Session) {
}

func (c *Client) ListSessions(filter *SessionFilter) ([]*Session, error) {
	u := qs.Generate(filter).Encode()
	var out []*Session
	if err := c.get(fmt.Sprintf("/v2/auth/sessions?%s", u), &out); err != nil {
		return nil, err
	}
	for _, p := range out {
		fixupSessionResponse(p)
	}
	return out, nil
}

func (c *Client) GetSession(uuid string) (*Session, error) {
	var out *Session
	if err := c.get(fmt.Sprintf("/v2/auth/sessions/%s", uuid), &out); err != nil {
		return nil, err
	}
	fixupSessionResponse(out)
	return out, nil
}

func (c *Client) DeleteSession(in *Session) (Response, error) {
	var out Response
	return out, c.delete(fmt.Sprintf("/v2/auth/sessions/%s", in.UUID), &out)
}
