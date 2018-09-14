package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pborman/uuid"

	"github.com/starkandwayne/shield/db"
	"github.com/starkandwayne/shield/lib/uaa"
	"github.com/starkandwayne/shield/util"
)

type UAAAuthProvider struct {
	AuthProviderBase

	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
	UAAEndpoint   string `json:"uaa_endpoint"`
	SkipVerifyTLS bool   `json:"skip_verify_tls"`

	Mapping []struct {
		Tenant string `json:"tenant"`
		Rights []struct {
			SCIM string `json:"scim"`
			Role string `json:"role"`
		} `json:"rights"`
	} `json:"mapping"`

	core *Core
	uaa  *uaa.Client
}

func (p *UAAAuthProvider) Configure(raw map[interface{}]interface{}) error {
	b, err := json.Marshal(util.StringifyKeys(raw))
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, p)
	if err != nil {
		return err
	}

	if p.ClientID == "" {
		return fmt.Errorf("invalid configuration for UAA OAuth Provider: missing `client_id' value")
	}

	if p.ClientSecret == "" {
		return fmt.Errorf("invalid configuration for UAA OAuth Provider: missing `client_secret' value")
	}

	if p.UAAEndpoint == "" {
		return fmt.Errorf("invalid configuration for UAA OAuth Provider: missing 'uaa_endpoint' value")
	}

	p.UAAEndpoint = strings.TrimSuffix(p.UAAEndpoint, "/")
	p.properties = util.StringifyKeys(raw).(map[string]interface{})

	p.uaa = uaa.NewClient(uaa.Client{
		ID:       p.ClientID,
		Secret:   p.ClientSecret,
		Endpoint: p.UAAEndpoint,
	}, !p.SkipVerifyTLS)

	return nil
}

func (p *UAAAuthProvider) ReferencedTenants() []string {
	ll := make([]string, 0)
	for _, m := range p.Mapping {
		ll = append(ll, m.Tenant)
	}
	return ll
}

func (p *UAAAuthProvider) Initiate(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Location", p.uaa.AuthorizationURL(uaa.DefaultScopes))
	w.WriteHeader(302)
}

func (p *UAAAuthProvider) HandleRedirect(req *http.Request) *db.User {
	code := req.URL.Query().Get("code")
	if code == "" {
		p.Errorf("no code parameter was supplied by the remote UAA")
		return nil
	}

	token, err := p.uaa.GetAccessToken(code)
	if err != nil {
		p.Errorf("unable to fetch access token: %s", err)
		return nil
	}

	account, name, scims, err := p.uaa.Lookup(token)
	if err != nil {
		p.Errorf("unable to retrieve user information: %s", err)
		return nil
	}

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
	for tenant, role := range p.resolveSCIM(scims) {
		if !p.Assign(user, tenant, role) {
			return nil
		}
	}
	if !p.SaveAssignments(p.core.DB, user) {
		return nil
	}

	return user
}

func (p UAAAuthProvider) resolveSCIM(scims []string) map[string]string {
	rights := make(map[string]string)

	for _, mapping := range p.Mapping {
	Rights:
		for _, right := range mapping.Rights {
			for _, scim := range scims {
				if right.SCIM == "" || scim == right.SCIM {
					rights[mapping.Tenant] = right.Role
					break Rights
				}
			}
		}
	}

	return rights
}
