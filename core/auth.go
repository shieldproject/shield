package core

import (
	"fmt"
	"strings"

	"github.com/jhunt/go-log"

	"github.com/starkandwayne/shield/db"
	"github.com/starkandwayne/shield/route"
)

type authTenant struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
	Role string `json:"role"`
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

func (c *Core) checkAuth(user *db.User) (*authResponse, error) {
	if user == nil {
		return nil, nil
	}

	answer := authResponse{
		User: authUser{
			UUID:    user.UUID,
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

	memberships, err := c.db.GetMembershipsForUser(user.UUID)
	if err != nil {
		log.Debugf("failed to retrieve tenant memberships for user %s@%s (uuid %s): %s",
			user.Account, user.Backend, user.UUID, err)
		return nil, err
	}

	answer.Tenants = make([]authTenant, len(memberships))
	answer.Grants.Tenants = make(map[string]authTenantGrant)
	for i, membership := range memberships {
		answer.Tenants[i].UUID = membership.TenantUUID
		answer.Tenants[i].Name = membership.TenantName
		answer.Tenants[i].Role = membership.Role

		if answer.Tenants[i].UUID == user.DefaultTenant {
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
		answer.Grants.Tenants[membership.TenantUUID] = grant
	}
	if answer.Tenant == nil && len(answer.Tenants) > 0 {
		answer.Tenant = &answer.Tenants[0]
	}

	if answer.User.Backend == "local" {
		answer.User.Backend = "SHIELD"
	} else {
		if p, ok := c.providers[answer.User.Backend]; ok {
			answer.User.Backend = p.Configuration(false).Name
		}
	}

	return &answer, nil
}

func (c *Core) hasRole(fail bool, r *route.Request, tenant string, roles ...string) bool {
	user, err := c.AuthenticatedUser(r)
	if user == nil || err != nil {
		r.Fail(route.Unauthorized(err, "Authorization required"))
		return false
	}

	memberships, err := c.db.GetMembershipsForUser(user.UUID)
	if err != nil {
		err = fmt.Errorf("failed to retrieve tenant memberships for user %s@%s (uuid %s): %s",
			user.Account, user.Backend, user.UUID, err)
		if fail {
			r.Fail(route.Forbidden(err, "Access denied"))
		}
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
			if m.TenantUUID == tenant {
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
	if fail {
		r.Fail(route.Forbidden(nil, "Access denied"))
	}
	return false
}

func (c *Core) hasTenant(fail bool, r *route.Request, id string) bool {
	tenant, err := c.db.GetTenant(id)
	if err != nil || tenant == nil {
		if fail {
			r.Fail(route.NotFound(err, "No such tenant"))
		}
		return false
	}
	return true
}

func (c *Core) CanManageTenants(r *route.Request, tenant string) bool {
	user, err := c.AuthenticatedUser(r)
	if user == nil || err != nil {
		r.Fail(route.Unauthorized(err, "Authorization required"))
		return false
	}

	if user.SysRole == "admin" || user.SysRole == "manager" {
		return true
	}

	memberships, err := c.db.GetMembershipsForUser(user.UUID)
	if err != nil {
		err = fmt.Errorf("failed to retrieve tenant memberships for user %s@%s (uuid %s): %s",
			user.Account, user.Backend, user.UUID, err)
		r.Fail(route.Forbidden(err, "Access denied"))
		return false
	}

	for _, m := range memberships {
		if m.TenantUUID == tenant {
			if m.Role == "admin" {
				return true
			}
			break
		}
	}

	r.Fail(route.Forbidden(nil, "Access denied"))
	return false
}

func (c *Core) AuthenticatedUser(r *route.Request) (*db.User, error) {
	session, err := c.db.GetSession(r.SessionID())
	if err != nil {
		log.Errorf("failed to retrieve session [%s] from database: %s", r.SessionID(), err)
		return nil, err
	}
	if session == nil {
		log.Errorf("failed to retrieve session [%s] from database: (no such session)", r.SessionID())
		return nil, err
	}
	session.IP = r.RemoteIP()
	session.UserAgent = r.UserAgent()

	if session.Expired(c.Config.API.Session.Timeout) {
		log.Infof("session %s expired; purging...", r.SessionID())
		c.db.ClearSession(session.UUID)
		return nil, nil
	}
	user, err := c.db.GetUserForSession(session.UUID)
	if err != nil || user == nil {
		log.Errorf("failed to retrieve user belonging to session [%s] from database: %s", session.UUID, err)
		return user, err
	}

	err = c.db.PokeSession(session)
	if err != nil {
		log.Errorf("Failed to poke session %s with error %s", session, err.Error())
	}

	return user, nil
}

func (c *Core) IsNotAuthenticated(r *route.Request) bool {
	if user, err := c.AuthenticatedUser(r); user == nil || err != nil {
		r.Fail(route.Unauthorized(err, "Authorization required"))
		return true
	}
	return false
}

func (c *Core) IsNotSystemAdmin(r *route.Request) bool {
	return !c.hasRole(true, r, "", "system/admin")
}

func (c *Core) IsNotSystemManager(r *route.Request) bool {
	return !c.hasRole(true, r, "", "system/manager", "system/admin")
}

func (c *Core) IsNotSystemEngineer(r *route.Request) bool {
	return !c.hasRole(true, r, "", "system/engineer", "system/manager", "system/admin")
}

func (c *Core) IsNotTenantAdmin(r *route.Request, tenant string) bool {
	return !c.hasRole(true, r, tenant, "tenant/admin", "system/manager", "system/admin") ||
		!c.hasTenant(true, r, tenant)
}

func (c *Core) IsNotTenantEngineer(r *route.Request, tenant string) bool {
	return !c.hasRole(true, r, tenant, "tenant/engineer", "tenant/admin", "system/*") ||
		!c.hasTenant(true, r, tenant)
}

func (c *Core) IsNotTenantOperator(r *route.Request, tenant string) bool {
	return !c.hasRole(true, r, tenant, "tenant/*", "system/*") ||
		!c.hasTenant(true, r, tenant)
}

func (c *Core) CanSeeCredentials(r *route.Request, tenant string) bool {
	return c.hasRole(false, r, tenant, "tenant/engineer", "tenant/admin", "system/*") &&
		c.hasTenant(false, r, tenant)
}
func (c *Core) CanSeeGlobalCredentials(r *route.Request) bool {
	return c.hasRole(false, r, "", "system/*")
}
