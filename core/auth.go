package core

import (
	"fmt"
	"net/http"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/goutils/log"
	"github.com/starkandwayne/shield/db"
	"github.com/starkandwayne/shield/route"
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
		return nil, nil
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
	user, err := core.AuthenticatedUser(r)
	if user == nil || err != nil {
		r.Fail(route.Unauthorized(err, "Authorization required"))
		return false
	}

	memberships, err := core.DB.GetMembershipsForUser(user.UUID)
	if err != nil {
		err = fmt.Errorf("failed to retrieve tenant memberships for user %s@%s (uuid %s): %s",
			user.Account, user.Backend, user.UUID.String(), err)
		r.Fail(route.Forbidden(err, "Access denied"))
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

	r.Fail(route.Forbidden(nil, "Access denied"))
	return false
}

func (core *Core) hasSystemRole(r *route.Request, roles ...string) bool {
	user, err := core.AuthenticatedUser(r)
	if user == nil || err != nil {
		r.Fail(route.Unauthorized(err, "Authorization required"))
		return false
	}

	for _, role := range roles {
		if user.SysRole == role {
			return true
		}
	}

	r.Fail(route.Forbidden(nil, "Access denied"))
	return false
}

func (core *Core) AuthenticatedUser(r *route.Request) (*db.User, error) {
	session := r.SessionID()
	user, err := core.DB.GetUserForSession(session)
	if err != nil || user == nil {
		return user, err
	}

	err = core.DB.PokeSession(&db.Session{
		UUID:      uuid.Parse(session),
		UserUUID:  user.UUID,
		IP:        r.Req.RemoteAddr,
		UserAgent: r.Req.UserAgent(),
	})
	if err != nil {
		log.Errorf("Failed to poke session %s with error %s", session, err.Error())
	}

	return user, nil
}

func (core *Core) IsNotAuthenticated(r *route.Request) bool {
	if user, err := core.AuthenticatedUser(r); user == nil || err != nil {
		r.Fail(route.Unauthorized(err, "Authorization required"))
		return true
	}
	return false
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
