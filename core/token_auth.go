package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/goutils/log"

	"github.com/starkandwayne/shield/db"
	"github.com/starkandwayne/shield/util"
)

type TokenAuthProvider struct {
	Tokens map[string][]struct {
		Tenant string `json:"tenant"`
		Role   string `json:"role"`
	} `json:"tokens"`

	Name       string
	Identifier string
	Usage      string
	core       *Core
}

func (t *TokenAuthProvider) DisplayName() string {
	return t.Name
}

func (t *TokenAuthProvider) Configure(raw map[interface{}]interface{}) error {
	b, err := json.Marshal(util.StringifyKeys(raw))
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, t)
	if err != nil {
		return err
	}

	return nil
}

func (t *TokenAuthProvider) Initiate(w http.ResponseWriter, req *http.Request) {
	token := req.Header.Get("X-Shield-Token")
	log.Debugf("X-Shield-Token is [%s]", token)

	assignments, ok := t.Tokens[token]
	if !ok {
		t.log("authentication via token '%s' failed", token)
		t.fail(w)
		return
	}

	user, err := t.core.DB.GetUser(token, t.Identifier)
	if err != nil {
		t.log("failed to retrieve user %s@%s from database: %s\n", token, t.Identifier, err)
		t.log("failed to retrieve user %s@%s from database: %s\n", token, t.Identifier, err)
		t.fail(w)
		return
	}
	if user == nil {
		user = &db.User{
			UUID:    uuid.NewRandom(),
			Name:    token,
			Account: token,
			Backend: t.Identifier,
			SysRole: "",
		}
		t.core.DB.CreateUser(user)
	}
	session, err := t.core.createSession(user)
	if err != nil {
		t.log("failed to create a session for user %s: %s\n", token, err)
		t.fail(w)
		return
	}

	http.SetCookie(w, SessionCookie(session.UUID.String(), true))

	if err := t.core.DB.ClearMembershipsFor(user); err != nil {
		t.log("failed to clear memberships for user %s: %s\n", token, err)
		t.fail(w)
		return
	}
	for _, assignment := range assignments {
		t.log("ensuring tenant '%s'\n", assignment.Tenant)
		tenant, err := t.core.DB.EnsureTenant(assignment.Tenant)
		if err != nil {
			t.log("failed to find/create tenant '%s': %s\n", assignment.Tenant, err)
			t.fail(w)
			return
		}
		t.log("user = %v; tenant = %v\n", user, tenant)
		t.log("assigning %s (user %s) to tenant '%s' as role '%s'\n", token, user.UUID, tenant.UUID, assignment.Role)
		err = t.core.DB.AddUserToTenant(user.UUID.String(), tenant.UUID.String(), assignment.Role)
		if err != nil {
			t.log("failed to assign %s to tenant '%s' as role '%s': %s\n", token, assignment.Tenant, assignment.Role, err)
			t.fail(w)
			return
		}
	}

	w.Header().Set("Location", "/")
	w.WriteHeader(302)
}

func (t *TokenAuthProvider) HandleRedirect(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(500)
	fmt.Fprintf(w, "token auth provider should never get this far\n")
}

func (t TokenAuthProvider) log(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "token-auth[%s]: ", t.Identifier)
	fmt.Fprintf(os.Stderr, msg, args...)
}

func (t TokenAuthProvider) fail(w http.ResponseWriter) {
	w.Header().Set("Location", "/fail/e500")
	w.WriteHeader(302)
}
