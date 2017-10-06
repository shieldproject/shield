package core

import (
	"fmt"
	"net/http"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/goutils/log"
)

func (core *Core) FindAuthProvider(identifier string) (AuthProvider, error) {
	var provider AuthProvider

	for _, auth := range core.auth {
		if auth.Identifier == identifier {
			switch auth.Backend {
			case "github":
				provider = &GithubAuthProvider{
					Name:       auth.Name,
					Identifier: identifier,
					core:       core,
				}
			case "uaa":
				provider = &UAAAuthProvider{
					Identifier: identifier,
					core:       core,
				}
			default:
				return nil, fmt.Errorf("unrecognized auth provider type '%s'", auth.Backend)
			}

			if err := provider.Configure(auth.Properties); err != nil {
				return nil, fmt.Errorf("failed to configure '%s' auth provider '%s': %s",
					auth.Backend, auth.Identifier, err)
			}
			return provider, nil
		}
	}

	return nil, fmt.Errorf("auth provider %s not defined", identifier)
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
type authResponse struct {
	User    authUser     `json:"user"`
	Tenants []authTenant `json:"tenants"`
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

func (core *Core) checkAuth(req *http.Request) (*authResponse, error) {
	sessionID := getSessionID(req)
	if sessionID == "" {
		log.Debugf("No session ID was found in request")
		return nil, nil
	}

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

	memberships, err := core.DB.GetMembershipsForUser(user.UUID)
	if err != nil {
		log.Debugf("failed to retrieve tenant memberships for user %s@%s (uuid %s): %s",
			user.Account, user.Backend, user.UUID.String(), err)
		return nil, err
	}

	answer.Tenants = make([]authTenant, len(memberships))
	for i, membership := range memberships {
		answer.Tenants[i].UUID = membership.TenantUUID
		answer.Tenants[i].Name = membership.TenantName
		answer.Tenants[i].Role = membership.Role
	}

	if answer.User.Backend == "local" {
		answer.User.Backend = "SHIELD"

	} else {
		log.Debugf("looking up auth provider configuration for '%s'", user.Backend)
		if p, err := core.FindAuthProvider(answer.User.Backend); err == nil {
			answer.User.Backend = p.DisplayName()
		}
	}

	return &answer, nil
}
