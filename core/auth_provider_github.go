package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pborman/uuid"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"

	"github.com/shieldproject/shield/db"
	"github.com/shieldproject/shield/lib/github"
	"github.com/shieldproject/shield/route"
	"github.com/shieldproject/shield/util"
)

type GithubAuthProvider struct {
	AuthProviderBase

	ClientID         string `json:"client_id"`
	ClientSecret     string `json:"client_secret"`
	GithubEndpoint   string `json:"github_endpoint"`
	GithubAPI        string `json:"github_api"`
	GithubEnterprise bool   `json:"github_enterprise"`
	Mapping          []struct {
		Github string `json:"github"`
		Tenant string `json:"tenant"`
		Rights []struct {
			Team string `json:"team"`
			Role string `json:"role"`
		} `json:"rights"`
	} `json:"mapping"`
}

func (p *GithubAuthProvider) Configure(raw map[interface{}]interface{}) error {
	b, err := json.Marshal(util.StringifyKeys(raw))
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, p)
	if err != nil {
		return err
	}

	if p.ClientID == "" {
		return fmt.Errorf("invalid configuration for Github OAuth Provider: missing `client_id' value")
	}

	if p.ClientSecret == "" {
		return fmt.Errorf("invalid configuration for Github OAuth Provider: missing `client_secret' value")
	}

	if p.GithubEndpoint == "" {
		p.GithubEndpoint = "https://github.com"
		p.GithubAPI = "https://api.github.com/"
	}

	p.GithubEndpoint = strings.TrimSuffix(p.GithubEndpoint, "/")
	if p.GithubAPI == "" {
		p.GithubAPI = p.GithubEndpoint + "/api/v3/"
	}

	p.properties = util.StringifyKeys(raw).(map[string]interface{})

	return nil
}

func (p *GithubAuthProvider) WireUpTo(c *Core) {
	p.core = c
}

func (p *GithubAuthProvider) ReferencedTenants() []string {
	ll := make([]string, 0)
	for _, m := range p.Mapping {
		ll = append(ll, m.Tenant)
	}
	return ll
}

func (p *GithubAuthProvider) Initiate(r *route.Request) {
	r.Redirect(302, p.authorizeURL("read:org"))
}

func (p *GithubAuthProvider) HandleRedirect(r *route.Request) *db.User {
	ctx := context.Background()
	code := r.Param("code", "")

	conf := &oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		Endpoint:     githuboauth.Endpoint,
	}
	if p.GithubEnterprise {
		conf.Endpoint = oauth2.Endpoint{
			AuthURL:  p.GithubEndpoint + "/login/oauth/authorize",
			TokenURL: p.GithubEndpoint + "/login/oauth/access_token",
		}
	}

	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		p.Errorf("failed to exchange OAuth2 code for token: %s", err)
		return nil
	}
	token := tok.AccessToken

	client, err := github.NewClient(p.GithubAPI, token)
	if err != nil {
		p.Errorf("failed to perform lookup against Github: %s", err)
		return nil
	}

	account, name, orgs, err := client.Lookup()
	if err != nil {
		p.Errorf("failed to perform lookup against Github: %s", err)
		return nil
	}

	// check if the user that logged in via github already exists
	if p.core.db == nil {
		p.Errorf("no handle for the core database found!")
		return nil
	}
	user, err := p.core.db.GetUser(account, p.Identifier)
	if err != nil {
		p.Errorf("failed to retrieve user %s@%s from database: %s", account, p.Identifier, err)
		return nil
	}
	if user == nil {
		user = &db.User{
			UUID:    uuid.NewRandom().String(),
			Name:    name,
			Account: account,
			Backend: p.Identifier,
			SysRole: "",
		}
		p.core.db.CreateUser(user)
	}

	p.ClearAssignments()
Mapping:
	for _, candidate := range p.Mapping {
		for org, teams := range orgs {
			if candidate.Github == org {
				for _, match := range candidate.Rights {
					if match.Team == "" {
						if !p.Assign(user, candidate.Tenant, match.Role) {
							return nil
						}
						continue Mapping
					}

					for _, team := range teams {
						if match.Team == team {
							if !p.Assign(user, candidate.Tenant, match.Role) {
								return nil
							}
							continue Mapping
						}
					}
				}
			}
		}
	}
	if !p.SaveAssignments(p.core.db, user) {
		return nil
	}

	return user
}

func (p GithubAuthProvider) authorizeURL(scope string) string {
	return fmt.Sprintf("%s/login/oauth/authorize?scope=%s&client_id=%s", p.GithubEndpoint, scope, p.ClientID)
}
