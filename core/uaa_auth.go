package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/pborman/uuid"

	"github.com/starkandwayne/shield/db"
	"github.com/starkandwayne/shield/lib/uaa"
	"github.com/starkandwayne/shield/util"
)

/*
UAAAuthProvider contains all of the fields UAA needs to give us a token,
and contains a mapping of the uaa roles

client_id	String	Optional	A unique string representing the registration information provided by the client, the recipient of the token. Optional if it is passed as part of the Basic Authorization header.
grant_type	String	Required	the type of authentication being used to obtain the token, in this case client_credentials
client_secret	String	Optional	The secret passphrase configured for the OAuth client. Optional if it is passed as part of the Basic Authorization header.
response_type	String	Optional	The type of token that should be issued.
token_format	String	Optional	UAA 3.3.0 Can be set to 'opaque' to retrieve an opaque and revocable token.

*/
type UAAAuthProvider struct {
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
	UAAEndpoint   string `json:"uaa_endpoint"`
	SkipVerifyTLS bool   `json:"skip_verify_tls"`

	Mapping       []struct {
		Tenant string `json:"tenant"`
		Rights []struct {
			SCIM  string `json:"scim"`
			Role  string `json:"role"`
		} `json:"rights"`
	} `json:"mapping"`

	Name       string
	Identifier string
	core       *Core
	uaa        *uaa.Client
	http *http.Client
}

func (p *UAAAuthProvider) DisplayName() string {
	return p.Name
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

func (p *UAAAuthProvider) HandleRedirect(w http.ResponseWriter, req *http.Request) {
	code := req.URL.Query().Get("code")
	if code == "" {
		p.fail(w, fmt.Errorf("No code was supplied by the remote UAA"))
		return
	}

	token, err := p.uaa.GetAccessToken(code)
	if err != nil {
		p.fail(w, fmt.Errorf("Unable to fetch access token: %s", err))
		return
	}

	account, name, scims, err := p.uaa.Lookup(token)
	if err != nil {
		p.fail(w, fmt.Errorf("Unable to retrieve user information: %s", err))
		return
	}

	user, err := p.core.DB.GetUser(account, p.Identifier)
	if err != nil {
		p.fail(w, fmt.Errorf("failed to retrieve user %s@%s from database: %s", account, p.Identifier, err))
		return
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
	session, err := p.core.createSession(user)
	if err != nil {
		p.fail(w, fmt.Errorf("failed to create a session for user %s: %s", account, err))
		return
	}

	http.SetCookie(w, SessionCookie(session.UUID.String(), true))

	if err := p.core.DB.ClearMembershipsFor(user); err != nil {
		p.fail(w, fmt.Errorf("failed to clear memberships for user %s: %s", account, err))
		return
	}
	for tname, role := range p.resolveSCIM(scims) {
		p.log("ensuring that tenant '%s' exists", tname)
		tenant, err := p.core.DB.EnsureTenant(tname)
		if err != nil {
			p.fail(w, fmt.Errorf("failed to find/create tenant '%s': %s", tname, err))
			return
		}
		p.log("user = %v; tenant = %s", user, tname)
		p.log("assigning %s (user %s) to tenant '%s' as role '%s'", account, user.UUID, tenant.UUID, role)
		err = p.core.DB.AddUserToTenant(user.UUID.String(), tenant.UUID.String(), role)
		if err != nil {
			p.fail(w, fmt.Errorf("failed to assign %s to tenant '%s' as role '%s': %s", account, tname, role, err))
			return
		}
	}

	w.Header().Set("Location", "/")
	w.WriteHeader(302)
}

func (p UAAAuthProvider) resolveSCIM(scims []string) (map[string]string) {
	rights := make(map[string] string)

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

func (p UAAAuthProvider) log(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "uaa auth provider [%s]: %s\n", p.Identifier, fmt.Sprintf(msg, args...))
}

func (p UAAAuthProvider) fail(w http.ResponseWriter, err error) {
	p.log("%s", err)
	w.Header().Set("Location", "/fail/e500")
	w.WriteHeader(302)
}
