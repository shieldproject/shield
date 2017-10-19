package core

import (
	"fmt"
	"net/http"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/goutils/log"
	"github.com/starkandwayne/shield/db"
	"github.com/starkandwayne/shield/route"
)

const (
	//UnauthString is the error returned for a 401
	UnauthString = "Authentication required"
	//OopsString is a default error returned for a 500
	OopsString = "An unknown error occurred"
)

func IsValidTenantRole(role string) bool {
	return role == "admin" || role == "engineer" || role == "operator"
}

func IsValidSystemRole(role string) bool {
	return role == "admin" || role == "manager" || role == "engineer"
}

type authTenant struct {
	UUID uuid.UUID `json:"uuid"`
	Name string    `json:"name"`
	Role string    `json:"role"`
}
type authUser struct {
	Name    string `json:"name"`
	Account string `json:"account"`
	Backend string `json:"backend"`
	SysRole string `json:"sysrole"`
}
type authTenantGrant struct {
	Admin    bool `json:"admin"`
	Engineer bool `json:"engineer"`
	Operator bool `json:"operator"`
}
type authGrants struct {
	System struct {
		Admin    bool `json:"admin"`
		Manager  bool `json:"manager"`
		Engineer bool `json:"engineer"`
	} `json:"system"`
	Tenants map[string]authTenantGrant `json:"tenant"`
}
type authResponse struct {
	User    authUser     `json:"user"`
	Tenants []authTenant `json:"tenants"`
	//The currently selected tenant
	Tenant *authTenant `json:"tenant,omitempty"`

	Grants authGrants `json:"is"`
}

//Gets the session ID from the request. Returns "" if not given.
func getSessionID(req *http.Request) string {
	sessionID := req.Header.Get("X-Shield-Session")

	if sessionID == "" { //If not in header, check for cookie
		cookie, err := req.Cookie(SessionCookieName)
		if err == nil { //ErrNoCookie is the only error returned from Cookie()
			sessionID = cookie.Value
		}
	}

	return sessionID
}

func getAuthToken(req *http.Request) string {
	return req.Header.Get("X-Shield-Token")
}

func (core *Core) checkAuth(user *db.User) (*authResponse, error) {
	if user == nil {
		return nil, fmt.Errorf("No user was given")
	}

	answer := authResponse{
		User: authUser{
			Name:    user.Name,
			Account: user.Account,
			Backend: user.Backend,
			SysRole: user.SysRole,
		},
	}

	switch user.SysRole {
	case "admin":
		answer.Grants.System.Admin = true
		answer.Grants.System.Manager = true
		answer.Grants.System.Engineer = true
	case "manager":
		answer.Grants.System.Manager = true
		answer.Grants.System.Engineer = true
	case "engineer":
		answer.Grants.System.Engineer = true
	}

	memberships, err := core.DB.GetMembershipsForUser(user.UUID)
	if err != nil {
		log.Debugf("failed to retrieve tenant memberships for user %s@%s (uuid %s): %s",
			user.Account, user.Backend, user.UUID.String(), err)
		return nil, err
	}

	answer.Tenants = make([]authTenant, len(memberships))
	answer.Grants.Tenants = make(map[string]authTenantGrant)
	for i, membership := range memberships {
		answer.Tenants[i].UUID = membership.TenantUUID
		answer.Tenants[i].Name = membership.TenantName
		answer.Tenants[i].Role = membership.Role

		grant := authTenantGrant{}
		switch membership.Role {
		case "admin":
			grant.Admin = true
			grant.Engineer = true
			grant.Operator = true
		case "engineer":
			grant.Engineer = true
			grant.Operator = true
		case "operator":
			grant.Operator = true
		}
		answer.Grants.Tenants[membership.TenantUUID.String()] = grant
	}
	if len(answer.Tenants) > 0 {
		answer.Tenant = &answer.Tenants[0]
	}

	if answer.User.Backend == "local" {
		answer.User.Backend = "SHIELD"
	} else {
		if p, ok := core.auth[answer.User.Backend]; ok {
			answer.User.Backend = p.Configuration(false).Name
		}
	}

	return &answer, nil
}

//SetAuthHeaders sets the appropriate HTTP headers in the given request object
// to send back the session information in a login request
func SetAuthHeaders(r *route.Request, sessionID uuid.UUID) {
	r.SetCookie(SessionCookie(sessionID.String(), true))
	r.SetRespHeader("X-Shield-Session", sessionID.String())
}

func (core *Core) hasTenantRole(r *route.Request, tenant string, roles ...string) bool {
	session, _ := r.SessionID()
	if uuid.Parse(session) == nil {
		return false
	}

	user, err := core.DB.GetUserForSession(session)
	if err != nil {
		log.Debugf("failed to retrieve user account for session '%s': %s", session, err)
		return false
	}
	if user == nil {
		log.Debugf("failed to retrieve user account for session '%s': database did not throw an error, but returned a nil user", session)
		return false
	}

	memberships, err := core.DB.GetMembershipsForUser(user.UUID)
	if err != nil {
		log.Debugf("failed to retrieve tenant memberships for user %s@%s (uuid %s): %s",
			user.Account, user.Backend, user.UUID.String(), err)
		return false
	}

	for _, m := range memberships {
		if m.TenantUUID.String() == tenant {
			for _, role := range roles {
				if role == m.Role {
					return true
				}
			}
			break
		}
	}

	return false
}

func (core *Core) hasSystemRole(r *route.Request, roles ...string) bool {
	session, _ := r.SessionID()
	if uuid.Parse(session) == nil {
		return false
	}

	user, err := core.DB.GetUserForSession(session)
	if err != nil {
		log.Debugf("failed to retrieve user account for session '%s': %s", session, err)
		return false
	}
	if user == nil {
		log.Debugf("failed to retrieve user account for session '%s': database did not throw an error, but returned a nil user", session)
		return false
	}

	for _, role := range roles {
		if user.SysRole == role {
			return true
		}
	}
	return false
}

//AuthenticatedUser gets the session ID out of the Request object and retrieves
// the user which it authenticates. If the session ID is invalid or no such
// session exists, nil is returned. Nil will also be returned if the session ID
// was given as an auth token and the session associated with the ID is not
// an auth token. The fail handler is called on the given route.Request object
// if no user is returned
func (core *Core) AuthenticatedUser(r *route.Request) *db.User {
	session, isToken := r.SessionID()
	sessionUUID := uuid.Parse(session)
	if sessionUUID == nil {
		r.Fail(route.Unauthorized(fmt.Errorf("No valid session ID was given"), UnauthString))
		return nil
	}

	user, err := core.DB.GetUserForSession(session)
	if err != nil {
		r.Fail(route.Oops(
			fmt.Errorf("failed to retrieve user account for session '%s': %s", session, err),
			OopsString,
		))
		return nil
	}
	if user == nil {
		r.Fail(route.Unauthorized(
			fmt.Errorf("failed to retrieve user account for session '%s': database did not throw an error, but returned a nil user", session),
			UnauthString,
		))
		return nil
	}

	//Verify a token is actually a token and a normal session isn't a token
	token, err := db.TokenFilter{SessionUUID: &sessionUUID}.Get(core.DB)
	if err != nil {
		r.Fail(route.Oops(
			fmt.Errorf("failed to retrieve token for session '%s': %s", session, err),
			OopsString,
		))
		return nil
	}

	if isToken != (token != nil) {
		r.Fail(route.Unauthorized(nil, UnauthString))
		return nil
	}
	err = core.DB.UpdateSessionLastUsed(sessionUUID)
	if err != nil {
		r.Fail(route.Oops(err, OopsString))
		return nil
	}

	return user
}

//IsNotAuthenticated returns true if no authenticated user is associated with
// the session ID in the given Request object. See AuthenticatedUser for the
// semantics of how a user is determined to be associated with a session ID.
func (core *Core) IsNotAuthenticated(r *route.Request) bool {
	return core.AuthenticatedUser(r) == nil
}

func (core *Core) IsNotSystemAdmin(r *route.Request) bool {
	return !core.hasSystemRole(r, "admin")
}

func (core *Core) IsNotSystemManager(r *route.Request) bool {
	return !core.hasSystemRole(r, "manager", "admin")
}

func (core *Core) IsNotSystemEngineer(r *route.Request) bool {
	return !core.hasSystemRole(r, "engineer", "manager", "admin")
}

func (core *Core) IsNotTenantAdmin(r *route.Request, tenant string) bool {
	return !core.hasTenantRole(r, tenant, "admin")
}

func (core *Core) IsNotTenantEngineer(r *route.Request, tenant string) bool {
	return !core.hasTenantRole(r, tenant, "engineer", "admin")
}

func (core *Core) IsNotTenantOperator(r *route.Request, tenant string) bool {
	return !core.hasTenantRole(r, tenant, "operator", "engineer", "admin")
}
