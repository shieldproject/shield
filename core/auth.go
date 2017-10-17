package core

import (
	"net/http"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/goutils/log"
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
	Tenant  *authTenant  `json:"tenant,omitempty"`

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

func (core *Core) checkAuth(sessionID string) (*authResponse, error) {
	log.Debugf("retrieving user account for session '%s'", sessionID)
	user, err := core.DB.GetUserForSession(sessionID)
	if err != nil {
		log.Debugf("failed to retrieve user account for session '%s': %s", sessionID, err)
		return nil, err
	}
	if user == nil {
		log.Debugf("failed to retrieve user account for session '%s': database did not throw an error, but returned a nil user", sessionID)
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
	r.SetHeader("X-Shield-Session", sessionID.String())
}
