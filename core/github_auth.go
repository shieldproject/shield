package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/pborman/uuid"

	"github.com/starkandwayne/shield/db"
	"github.com/starkandwayne/shield/lib/github"
	"github.com/starkandwayne/shield/util"
)

type GithubAuthProvider struct {
	AuthProviderBase

	ClientID       string `json:"client_id"`
	ClientSecret   string `json:"client_secret"`
	GithubEndpoint string `json:"github_endpoint"`
	Mapping        []struct {
		Github string `json:"github"`
		Tenant string `json:"tenant"`
		Rights []struct {
			Team string `json:"team"`
			Role string `json:"role"`
		} `json:"rights"`
	} `json:"mapping"`

	core *Core
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
	}
	p.GithubEndpoint = strings.TrimSuffix(p.GithubEndpoint, "/")
	p.properties = util.StringifyKeys(raw).(map[string]interface{})

	return nil
}

func (p *GithubAuthProvider) ReferencedTenants() []string {
	ll := make([]string, 0)
	for _, m := range p.Mapping {
		ll = append(ll, m.Tenant)
	}
	return ll
}

func (p *GithubAuthProvider) Initiate(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Location", p.authorizeURL("read:org"))
	w.WriteHeader(302)
}

func (p *GithubAuthProvider) HandleRedirect(req *http.Request) *db.User {
	var input = struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		Code         string `json:"code"`
	}{
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		Code:         req.URL.Query().Get("code"),
	}

	b, err := json.Marshal(input)
	if err != nil {
		p.Errorf("failed to marshal access token request: %s", err)
		return nil
	}

	uri := p.accessTokenURL()
	res, err := http.Post(uri, "application/json", bytes.NewBuffer(b))
	if err != nil {
		p.Errorf("failed to POST to Github access_token endpoint %s: %s", uri, err)
		return nil
	}
	b, err = ioutil.ReadAll(res.Body)
	if err != nil {
		p.Errorf("failed to read response from POST %s: %s", uri, err)
		return nil
	}
	u, err := url.Parse("?" + string(b))
	if err != nil {
		p.Errorf("failed to parse response '%s' from POST %s: %s", string(b), uri, err)
		return nil
	}
	token := u.Query().Get("access_token")
	if token == "" {
		p.Errorf("no access_token found in response '%s' from POST %s", string(b), u)
		return nil
	}

	client := github.NewClient(token)
	account, name, orgs, err := client.Lookup()
	if err != nil {
		p.Errorf("failed to perform lookup against Github: %s", err)
		return nil
	}

	//Check if the user that logged in via github already exists
	user, err := p.core.DB.GetUser(account, p.Identifier)
	if err != nil {
		p.Errorf("failed to retrieve user %s@%s from database: %s", account, p.Identifier, err)
		return nil
	}
	if user == nil {
		user = &db.User{
			UUID:    uuid.NewRandom(),
			Name:    name,
			Account: account,
			Backend: p.Identifier,
			SysRole: "",
		}
		p.core.DB.CreateUser(user)
	}

	p.ClearAssignments()
	for org, teams := range orgs {
		tenant, role, assigned := p.resolveOrgAndTeam(org, teams)
		if assigned && !p.Assign(user, tenant, role) {
			return nil
		}
	}
	if !p.SaveAssignments(p.core.DB, user) {
		return nil
	}

	return user
}

func (p GithubAuthProvider) resolveOrgAndTeam(org string, teams []string) (string, string, bool) {
	for _, candidate := range p.Mapping {
		if candidate.Github != org {
			continue
		}
		for _, match := range candidate.Rights {
			if match.Team == "" {
				return candidate.Tenant, match.Role, true
			}
			for _, team := range teams {
				if match.Team == team {
					return candidate.Tenant, match.Role, true
				}
			}
		}
	}
	return "", "", false /* not recognized; not allowed */
}

func (p GithubAuthProvider) accessTokenURL() string {
	return fmt.Sprintf("%s/login/oauth/access_token", p.GithubEndpoint)
}

func (p GithubAuthProvider) authorizeURL(scope string) string {
	return fmt.Sprintf("%s/login/oauth/authorize?scope=%s&client_id=%s", p.GithubEndpoint, scope, p.ClientID)
}
