package core

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pborman/uuid"

	"github.com/starkandwayne/shield/db"
	"github.com/starkandwayne/shield/util"
)

type TokenAuthProvider struct {
	AuthProviderBase

	Tokens map[string][]struct {
		Tenant string `json:"tenant"`
		Role   string `json:"role"`
	} `json:"tokens"`

	core *Core
}

func (p *TokenAuthProvider) Configure(raw map[interface{}]interface{}) error {
	b, err := json.Marshal(util.StringifyKeys(raw))
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, p)
	if err != nil {
		return err
	}

	return nil
}

func (p *TokenAuthProvider) Initiate(w http.ResponseWriter, req *http.Request) {
	/* just go straight to the redirector */
	w.Header().Set("Location", fmt.Sprintf("/auth/%s/redir", p.Identifier))
	w.WriteHeader(302)
}

func (p *TokenAuthProvider) HandleRedirect(req *http.Request) *db.User {
	token := req.Header.Get("X-Shield-Token")
	p.Debugf("X-Shield-Token is [%s]", token)

	assignments, ok := p.Tokens[token]
	if !ok {
		p.Errorf("authentication via token '%s' failed", token)
		return nil
	}

	user, err := p.core.DB.GetUser(token, p.Identifier)
	if err != nil {
		p.Errorf("failed to retrieve user %s@%s from database: %s\n", token, p.Identifier, err)
		return nil
	}
	if user == nil {
		user = &db.User{
			UUID:    uuid.NewRandom(),
			Name:    token,
			Account: token,
			Backend: p.Identifier,
			SysRole: "",
		}
		p.core.DB.CreateUser(user)
	}

	if err := p.core.DB.ClearMembershipsFor(user); err != nil {
		p.Errorf("failed to clear memberships for user %s: %s\n", token, err)
		return nil
	}
	for _, assignment := range assignments {
		p.Infof("ensuring tenant '%s'\n", assignment.Tenant)
		tenant, err := p.core.DB.EnsureTenant(assignment.Tenant)
		if err != nil {
			p.Errorf("failed to find/create tenant '%s': %s\n", assignment.Tenant, err)
			return nil
		}
		p.Infof("inviting %s [%s] to tenant '%s' [%s] as '%s'", token, user.UUID, tenant.Name, tenant.UUID, assignment.Role)
		err = p.core.DB.AddUserToTenant(user.UUID.String(), tenant.UUID.String(), assignment.Role)
		if err != nil {
			p.Errorf("failed to invite %s [%s] to tenant '%s' [%s] as %s: %s", token, user.UUID, tenant.Name, tenant.UUID, assignment.Role, err)
			return nil
		}
	}

	return user
}
