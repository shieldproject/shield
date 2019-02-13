package github

import (
	"context"
	"fmt"
	"net/url"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type Client struct {
	gh *github.Client
}

func NewClient(api, token string) (*Client, error) {
	gh := github.NewClient(
		oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)),
	)

	if api != "" {
		u, err := url.Parse(api)
		if err != nil {
			return nil, err
		}
		gh.BaseURL = u
	}
	return &Client{gh: gh}, nil
}

func (c *Client) Lookup() (string, string, map[string][]string, error) {
	m := make(map[string][]string)

	user, _, err := c.gh.Users.Get("")
	if err != nil {
		return "", "", nil, err
	}
	if user.Login == nil {
		return "", "", nil, fmt.Errorf("no login name found in Github profile...")
	}

	orgs, _, err := c.gh.Organizations.List("", nil)
	if err != nil {
		return "", "", nil, err
	}
	for _, org := range orgs {
		m[*org.Login] = make([]string, 0)
	}

	teams, _, err := c.gh.Organizations.ListUserTeams(nil)
	if err != nil {
		return "", "", nil, err
	}

	for _, team := range teams {
		m[*team.Organization.Login] = append(m[*team.Organization.Login], *team.Name)
	}

	if user.Name == nil {
		user.Name = user.Login
	}
	return *user.Login, *user.Name, m, nil
}
