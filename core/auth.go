package core

import (
	"fmt"
	"net/http"

	"github.com/pborman/uuid"
)

type userTenancyInfo struct {
	UUID uuid.UUID `json:"uuid"`
	Name string    `json:"name"`
	Role string    `json:"role"`
}
type sessionedUserInfo struct {
	User struct {
		UUID    string `json:"uuid"`
		Name    string `json:"name"`
		Account string `json:"account"`
		Backend string `json:"backend"`
	} `json:"user"`

	Tenants []userTenancyInfo `json:"tenants"`
}

func (core *Core) getUserInfoFromRequest(req *http.Request) (*sessionedUserInfo, error) {
	cookie, err := req.Cookie("shield7")
	if err != nil {
		return nil, err
	}

	fmt.Printf("retrieving user for session '%s'\n", cookie.Value)
	user, err := core.DB.GetUserForSession(cookie.Value)
	if err != nil || user == nil {
		return nil, err
	}

	answer := &sessionedUserInfo{}
	answer.User.UUID = user.UUID.String()
	answer.User.Name = user.Name
	answer.User.Account = user.Account
	answer.User.Backend = user.Backend

	memberships, err := core.DB.GetMembershipsForUser(user.UUID)
	if err != nil {
		return nil, err
	}

	answer.Tenants = make([]userTenancyInfo, len(memberships))
	for i, membership := range memberships {
		answer.Tenants[i].UUID = membership.TenantUUID
		answer.Tenants[i].Name = membership.TenantName
		answer.Tenants[i].Role = membership.Role
	}
	if err != nil {
		return nil, err
	}

	return answer, nil
}

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
