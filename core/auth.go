package core

import (
	"fmt"
	"strings"

	"github.com/jhunt/go-log"
	"github.com/pborman/uuid"

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
	UUID    string `json:"uuid"`
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

func (core *Core) checkAuth(user *db.User) (*authResponse, error) {
	if user == nil {
		return nil, nil
	}

	answer := authResponse{
		User: authUser{
			UUID:    user.UUID.String(),
			Name:    user.Name,
			Account: user.Account,
			Backend: user.Backend,
			SysRole: user.SysRole,
		},
		Tenant: nil,
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

		if answer.Tenants[i].UUID.String() == user.DefaultTenant {
			answer.Tenant = &answer.Tenants[i]
		}

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
	if answer.Tenant == nil && len(answer.Tenants) > 0 {
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

func SetAuthHeaders(r *route.Request, sessionID uuid.UUID) {
	r.SetCookie(SessionCookie(sessionID.String(), true))
	r.SetRespHeader("X-Shield-Session", sessionID.String())
}

func (core *Core) hasRole(r *route.Request, tenant string, roles ...string) bool {
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

	for _, role := range roles {
		l := strings.SplitN(role, "/", 2)
		if len(l) != 2 {
			continue
		}

		if l[0] == "system" {
			if l[1] == "*" && user.SysRole != "" {
				return true
			}
			if l[1] == user.SysRole {
				return true
			}
			continue
		}

		for _, m := range memberships {
			if m.TenantUUID.String() == tenant {
				if l[1] == "*" {
					return true
				}
				if l[1] == m.Role {
					return true
				}
				break
			}
		}
	}

	r.Fail(route.Forbidden(nil, "Access denied"))
	return false
}

func (core *Core) hasTenant(r *route.Request, id string) bool {
	tenant, err := core.DB.GetTenant(id)
	if err != nil || tenant == nil {
		r.Fail(route.NotFound(err, "No such tenant"))
		return false
	}
	return true
}

func (core *Core) CanManageTenants(r *route.Request, tenant string) bool {
	user, err := core.AuthenticatedUser(r)
	if user == nil || err != nil {
		r.Fail(route.Unauthorized(err, "Authorization required"))
		return false
	}

	if user.SysRole == "admin" || user.SysRole == "manager" {
		return true
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
			if m.Role == "admin" {
				return true
			}
			break
		}
	}

	r.Fail(route.Forbidden(nil, "Access denied"))
	return false
}

func (core *Core) AuthenticatedUser(r *route.Request) (*db.User, error) {
	session, err := core.DB.GetSession(r.SessionID())
	if err != nil || session == nil {
		return nil, err
	}
	session.IP = r.Req.RemoteAddr
	session.UserAgent = r.Req.UserAgent()

	if session.Expired(core.sessionTimeout) {
		log.Infof("session %s expired; purging...", r.SessionID())
		core.DB.ClearSession(session.UUID.String())
		return nil, nil
	}
	user, err := core.DB.GetUserForSession(session.UUID.String())
	if err != nil || user == nil {
		return user, err
	}

	err = core.DB.PokeSession(session)
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
	return !core.hasRole(r, "", "system/admin")
}

func (core *Core) IsNotSystemManager(r *route.Request) bool {
	return !core.hasRole(r, "", "system/manager", "system/admin")
}

func (core *Core) IsNotSystemEngineer(r *route.Request) bool {
	return !core.hasRole(r, "", "system/engineer", "system/manager", "system/admin")
}

func (core *Core) IsNotTenantAdmin(r *route.Request, tenant string) bool {
	return !core.hasRole(r, tenant, "tenant/admin", "system/manager", "system/admin") ||
		!core.hasTenant(r, tenant)
}

func (core *Core) IsNotTenantEngineer(r *route.Request, tenant string) bool {
	return !core.hasRole(r, tenant, "tenant/engineer", "tenant/admin", "system/*") ||
		!core.hasTenant(r, tenant)
}

func (core *Core) IsNotTenantOperator(r *route.Request, tenant string) bool {
	return !core.hasRole(r, tenant, "tenant/*", "system/*") ||
		!core.hasTenant(r, tenant)
}
