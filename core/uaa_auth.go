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
	http *http.Client
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

	p.uaa = uaa.NewClient(uaa.Client{
		ID:       p.ClientID,
		Secret:   p.ClientSecret,
		Endpoint: p.UAAEndpoint,
	}, !p.SkipVerifyTLS)

	return nil
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

	if err := p.core.DB.ClearMembershipsFor(user); err != nil {
		p.Errorf("failed to clear memberships for user %s: %s", account, err)
		return nil
	}
	for tname, role := range p.resolveSCIM(scims) {
		p.Infof("ensuring that tenant '%s' exists", tname)
		tenant, err := p.core.DB.EnsureTenant(tname)
		if err != nil {
			p.Errorf("failed to find/create tenant '%s': %s", tname, err)
			return nil
		}
		p.Infof("inviting %s [%s] to tenant '%s' [%s] as '%s'", account, user.UUID, tenant.Name, tenant.UUID, role)
		err = p.core.DB.AddUserToTenant(user.UUID.String(), tenant.UUID.String(), role)
		if err != nil {
			p.Errorf("failed to invite %s [%s] to tenant '%s' [%s] as %s: %s", account, user.UUID, tenant.Name, tenant.UUID, role, err)
			return nil
		}
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
