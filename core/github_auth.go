package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/pborman/uuid"

	"github.com/starkandwayne/shield/db"
	"github.com/starkandwayne/shield/lib/github"
	"github.com/starkandwayne/shield/util"
)

type GithubAuthProvider struct {
	ClientID       string `json:"client_id"`
	ClientSecret   string `json:"client_secret"`
	GithubEndpoint string `json:"github_endpoint"`
	Mapping        map[string]struct {
		Tenant string `json:"tenant"`
		Rights []struct {
			Team string `json:"team"`
			Role string `json:"role"`
		} `json:"rights"`
	} `json:"mapping"`

	Name       string
	Identifier string
	Usage      string
	core       *Core
}

func (gh *GithubAuthProvider) DisplayName() string {
	return gh.Name
}

func (gh *GithubAuthProvider) Configure(raw map[interface{}]interface{}) error {
	b, err := json.Marshal(util.StringifyKeys(raw))
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, gh)
	if err != nil {
		return err
	}

	if gh.ClientID == "" {
		return fmt.Errorf("invalid configuration for Github OAuth Provider: missing `client_id' value")
	}

	if gh.ClientSecret == "" {
		return fmt.Errorf("invalid configuration for Github OAuth Provider: missing `client_secret' value")
	}

	if gh.GithubEndpoint == "" {
		gh.GithubEndpoint = "https://github.com"
	}
	gh.GithubEndpoint = strings.TrimSuffix(gh.GithubEndpoint, "/")

	return nil
}

func (gh *GithubAuthProvider) Initiate(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Location", gh.authorizeURL("read:org"))
	w.WriteHeader(302)
}

func (gh *GithubAuthProvider) HandleRedirect(w http.ResponseWriter, req *http.Request) {
	var input = struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		Code         string `json:"code"`
	}{
		ClientID:     gh.ClientID,
		ClientSecret: gh.ClientSecret,
		Code:         req.URL.Query().Get("code"),
	}

	b, err := json.Marshal(input)
	if err != nil {
		gh.log("failed to marshal access token request: %s\n", err)
		gh.fail(w)
		return
	}

	res, err := http.Post(gh.accessTokenURL(), "application/json", bytes.NewBuffer(b))
	if err != nil {
		gh.log("failed to POST to Github access_token endpoint %s: %s\n", gh.accessTokenURL(), err)
		gh.fail(w)
		return
	}
	b, err = ioutil.ReadAll(res.Body)
	if err != nil {
		gh.log("failed to read response from POST %s: %s\n", gh.accessTokenURL(), err)
		gh.fail(w)
		return
	}
	u, err := url.Parse("?" + string(b))
	if err != nil {
		gh.log("failed to parse response '%s' from POST %s: %s\n", string(b), gh.accessTokenURL(), err)
		gh.fail(w)
		return
	}
	token := u.Query().Get("access_token")
	if token == "" {
		gh.log("no access_token found in response '%s'\n", string(b))
		gh.fail(w)
		return
	}

	client := github.NewClient(token)
	account, name, orgs, err := client.Lookup()
	if err != nil {
		gh.log("failed to perform lookup against Github: %s\n", err)
		gh.fail(w)
		return
	}

	//Check if the user that logged in via github already exists
	user, err := gh.core.DB.GetUser(account, gh.Identifier)
	if err != nil {
		gh.log("failed to retrieve user %s@%s from database: %s\n", account, gh.Identifier, err)
		gh.fail(w)
		return
	}
	if user == nil {
		user = &db.User{
			UUID:    uuid.NewRandom(),
			Name:    name,
			Account: account,
			Backend: gh.Identifier,
			SysRole: "",
		}
		gh.core.DB.CreateUser(user)
	}
	session, err := gh.core.createSession(user)
	if err != nil {
		gh.log("failed to create a session for user %s: %s\n", account, err)
		gh.fail(w)
		return
	}

	http.SetCookie(w, SessionCookie(session.UUID.String(), true))

	if err := gh.core.DB.ClearMembershipsFor(user); err != nil {
		gh.log("failed to clear memberships for user %s: %s\n", account, err)
		gh.fail(w)
		return
	}

	/* We must pre-determine who we're going to assign this Github user
		   to, and what role to give them, in case we have overalpping
		   mappings -- two orgs map to the same tenant, with different roles.

	       This way, we can silently 'ugrade' a role to a more powerful
		   one if we see a later assignment to the same tenant. */
	assign := make(map[string]string)
	for org, teams := range orgs {
		tname, role, assigned := gh.resolveOrgAndTeam(org, teams)
		if assigned {
			if existing, already := assign[tname]; already {
				if (existing == "operator" && existing != role) || (existing == "engineer" && role == "admin") {
					gh.log("upgrading %s (%s org) assignment on tenant '%s' to %s\n", account, org, tname, role)
					assign[tname] = role
				}
			} else {
				gh.log("assigning %s (%s org) to tenant '%s' as %s\n", account, org, tname, role)
				assign[tname] = role
			}
		}
	}
	for tname, role := range assign {
		gh.log("ensuring that tenant '%s' exists\n", tname)
		tenant, err := gh.core.DB.EnsureTenant(tname)
		if err != nil {
			gh.log("failed to find/create tenant '%s': %s\n", tname, err)
			gh.fail(w)
			return
		}
		gh.log("inviting %s [%s] to tenant '%s' [%s] as '%s'\n", account, user.UUID, tenant.Name, tenant.UUID, role)
		err = gh.core.DB.AddUserToTenant(user.UUID.String(), tenant.UUID.String(), role)
		if err != nil {
			gh.log("failed to invite %s [%s] to tenant '%s' [%s] as %s: %s\n", account, user.UUID, tenant.Name, tenant.UUID, role, err)
			gh.fail(w)
			return
		}
	}

	w.Header().Set("Location", "/")
	w.WriteHeader(302)
}

func (gh GithubAuthProvider) resolveOrgAndTeam(org string, teams []string) (string, string, bool) {
	if candidate, ok := gh.Mapping[org]; ok {
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

func (gh GithubAuthProvider) accessTokenURL() string {
	return fmt.Sprintf("%s/login/oauth/access_token", gh.GithubEndpoint)
}

func (gh GithubAuthProvider) authorizeURL(scope string) string {
	return fmt.Sprintf("%s/login/oauth/authorize?scope=%s&client_id=%s", gh.GithubEndpoint, scope, gh.ClientID)
}

func (gh GithubAuthProvider) log(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "github-auth[%s]: ", gh.Identifier)
	fmt.Fprintf(os.Stderr, msg, args...)
}

func (gh GithubAuthProvider) fail(w http.ResponseWriter) {
	w.Header().Set("Location", "/fail/e500")
	w.WriteHeader(302)
}
