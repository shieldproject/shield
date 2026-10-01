package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v92/github"
)

type Client struct {
	gh *github.Client
}

func NewClient(api, token string) (*Client, error) {
	opts := []github.ClientOptionsFunc{github.WithAuthToken(token)}
	if api != "" {
		opts = append(opts, github.WithURLs(&api, nil))
	}

	gh, err := github.NewClient(opts...)
	if err != nil {
		return nil, err
	}
	return &Client{gh: gh}, nil
}

func (c *Client) Lookup() (string, string, map[string][]string, error) {
	ctx := context.Background()
	m := make(map[string][]string)

	user, _, err := c.gh.Users.Get(ctx, "")
	if err != nil {
		return "", "", nil, err
	}
	if user.Login == nil {
		return "", "", nil, fmt.Errorf("no login name found in Github profile...")
	}

	// not passing username below only works in github enterprise.
	orgs, _, err := c.gh.Organizations.List(ctx, *user.Login, nil)
	if err != nil {
		return "", "", nil, err
	}
	for _, org := range orgs {
		if org.Login == nil {
			continue
		}
		m[*org.Login] = make([]string, 0)
	}

	teams, _, err := c.gh.Teams.ListUserTeams(ctx, nil)
	if err != nil {
		return "", "", nil, err
	}

	for _, team := range teams {
		if team.Organization == nil || team.Organization.Login == nil || team.Name == nil {
			continue
		}
		m[*team.Organization.Login] = append(m[*team.Organization.Login], *team.Name)
	}

	if user.Name == nil {
		user.Name = user.Login
	}
	return *user.Login, *user.Name, m, nil
}
