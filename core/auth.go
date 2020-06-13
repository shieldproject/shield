package core

import (
	"github.com/jhunt/go-log"

	"github.com/shieldproject/shield/db"
	"github.com/shieldproject/shield/route"
)

type authUser struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Account string `json:"account"`
	Backend string `json:"backend"`
	SysRole string `json:"sysrole"`
}
type authGrants struct {
	System struct {
		Admin    bool `json:"admin"`
		Manager  bool `json:"manager"`
		Engineer bool `json:"engineer"`
		Operator bool `json:"operator"`
	} `json:"system"`
}
type authResponse struct {
	User   authUser   `json:"user"`
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
	}

	switch user.SysRole {
	case "admin":
		answer.Grants.System.Admin = true
		answer.Grants.System.Manager = true
		answer.Grants.System.Engineer = true
		answer.Grants.System.Operator = true
	case "manager":
		answer.Grants.System.Manager = true
		answer.Grants.System.Engineer = true
		answer.Grants.System.Operator = true
	case "engineer":
		answer.Grants.System.Engineer = true
		answer.Grants.System.Operator = true
	case "operator":
		answer.Grants.System.Operator = true
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

func (c *Core) hasRole(fail bool, r *route.Request, roles ...string) bool {
	user, err := c.AuthenticatedUser(r)
	if user == nil || err != nil {
		r.Fail(route.Unauthorized(err, "Authorization required"))
		return false
	}

	for _, role := range roles {
		if role == "*" && user.SysRole != "" {
			return true
		}
		if role == user.SysRole {
			return true
		}
	}
	if fail {
		r.Fail(route.Forbidden(nil, "Access denied"))
	}
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

	if session.Expired(int(c.Config.API.Session.Timeout)) {
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
	return !c.hasRole(true, r, "admin")
}

func (c *Core) IsNotSystemManager(r *route.Request) bool {
	return !c.hasRole(true, r, "manager", "admin")
}

func (c *Core) IsNotSystemEngineer(r *route.Request) bool {
	return !c.hasRole(true, r, "engineer", "manager", "admin")
}

func (c *Core) IsNotSystemOperator(r *route.Request) bool {
	return !c.hasRole(true, r, "operator", "engineer", "manager", "admin")
}
