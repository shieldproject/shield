package core

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pborman/uuid"

	"github.com/starkandwayne/goutils/log"
	"github.com/starkandwayne/shield/db"
	"github.com/starkandwayne/shield/route"
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

	if len(p.Tokens) == 0 {
		return fmt.Errorf("No tokens were configured with token backend `%s'", p.Identifier)
	}

	return nil
}

func (p *TokenAuthProvider) Initiate(w http.ResponseWriter, req *http.Request) {
	r := route.NewRequest(w, req, false)

	token := req.Header.Get("X-Shield-Token")
	p.Debugf("X-Shield-Token is [%s]", token)

	assignments, ok := p.Tokens[token]
	if !ok {
		r.Fail(route.Errorf(
			401, //Unauthorized
			fmt.Errorf("authentication via token '%s' failed", token),
			"Refusing to authorize with given token"))
		return
	}

	var err error
	defer func() {
		if err != nil {
			r.Fail(route.Oops(nil, "An unknown error occurred"))
		}
	}()

	var user *db.User
	user, err = p.core.DB.GetUser(token, p.Identifier)
	if err != nil {
		p.Errorf("failed to retrieve user %s@%s from database: %s", token, p.Identifier, err)
		return
	}

	if user == nil {
		user = &db.User{
			UUID:    uuid.NewRandom(),
			Name:    token,
			Account: token,
			Backend: p.Identifier,
			SysRole: "",
		}
		_, err = p.core.DB.CreateUser(user)
		if err != nil {
			p.Errorf("failed to create new user in database %+v: %s", user, err)
			return
		}
	}

	if err = p.core.DB.ClearMembershipsFor(user); err != nil {
		p.Errorf("failed to clear memberships for user %s: %s\n", token, err)
		return
	}

	for _, assignment := range assignments {
		p.Infof("ensuring tenant '%s'\n", assignment.Tenant)
		var tenant *db.Tenant
		tenant, err = p.core.DB.EnsureTenant(assignment.Tenant)
		if err != nil {
			p.Errorf("failed to find/create tenant '%s': %s\n", assignment.Tenant, err)
			return
		}

		p.Infof("inviting %s [%s] to tenant '%s' [%s] as '%s'", token, user.UUID, tenant.Name, tenant.UUID, assignment.Role)
		err = p.core.DB.AddUserToTenant(user.UUID.String(), tenant.UUID.String(), assignment.Role)
		if err != nil {
			p.Errorf("failed to invite %s [%s] to tenant '%s' [%s] as %s: %s", token, user.UUID, tenant.Name, tenant.UUID, assignment.Role, err)
			return
		}
	}

	var session *db.Session
	session, err = p.core.createSession(user)
	if err != nil {
		log.Errorf("failed to create a session for user %s@%s: %s", user.Account, user.Backend, err)
		return
	}

	var id *authResponse
	id, err = p.core.checkAuth(session.UUID.String())
	if id == nil {
		p.Errorf("Failed to lookup session ID after login")
		return
	}

	SetAuthHeaders(r, session.UUID)
	r.OK(id)
}

func (p *TokenAuthProvider) HandleRedirect(req *http.Request) *db.User {
	p.Errorf("Can't handle redirect for token provider\n")
	return nil
}
