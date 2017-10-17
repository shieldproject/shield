package core

import (
	"fmt"
	"net/http"

	"github.com/starkandwayne/goutils/log"

	"github.com/starkandwayne/shield/db"
)

const (
	SystemTenatnName = "SYSTEM"
)

var (
	RoleTower map[string]int
)

func init() {
	RoleTower = map[string]int{
		"operator": 1,
		"engineer": 2,
		"manager":  3,
		"admin":    4,
	}
}

type AuthProvider interface {
	DisplayName() string

	Configure(map[interface{}]interface{}) error
	Initiate(http.ResponseWriter, *http.Request)
	HandleRedirect(*http.Request) *db.User
}

type AuthProviderBase struct {
	Name       string
	Identifier string
	Type       string

	assignments map[string]string
}

func (p AuthProviderBase) DisplayName() string {
	return p.Name
}

func (p AuthProviderBase) Errorf(m string, args ...interface{}) {
	args = append([]interface{}{p.Identifier, p.Type}, args...)
	log.Errorf("auth provider %s (%s): "+m, args...)
}

func (p AuthProviderBase) Infof(m string, args ...interface{}) {
	args = append([]interface{}{p.Identifier, p.Type}, args...)
	log.Infof("auth provider %s (%s): "+m, args...)
}

func (p AuthProviderBase) Debugf(m string, args ...interface{}) {
	args = append([]interface{}{p.Identifier, p.Type}, args...)
	log.Debugf("auth provider %s (%s): "+m, args...)
}

func (p AuthProviderBase) Fail(w http.ResponseWriter) {
	w.Header().Set("Location", "/fail/e500")
	w.WriteHeader(302)
}

func (p *AuthProviderBase) ClearAssignments() {
	p.assignments = make(map[string]string)
}

func (p *AuthProviderBase) Assign(user *db.User, tenant, role string) bool {
	who := fmt.Sprintf("%s (%s@%s)", user.Name, user.Account, user.Backend)
	if tenant == "SYSTEM" {
		p.Infof("assigning system role %s to %s", role, who)
		if !IsValidSystemRole(role) {
			p.Errorf("unable to assign system role %s to %s: '%s' is not a valid system role", role, who, role)
			return false
		}

	} else {
		p.Infof("assigning tenant role %s on '%s' to %s", role, tenant, who)
		if !IsValidTenantRole(role) {
			p.Errorf("unable to assign tenant role %s on '%s' to %s: '%s' is not a valid tenant role", role, tenant, who, role)
			return false
		}
	}

	if existing, already := p.assignments[tenant]; already {
		if RoleTower[existing] < RoleTower[role] {
			if tenant == "SYSTEM" {
				p.Infof("upgrading %s system assignment from %s -> %s", who, existing, role)
				p.assignments[tenant] = role

			} else {
				p.Infof("ignoring system assignment of %s to %s: %s is already assigned the %s role (which is more powerful)",
					role, who, who, existing)
			}
		}
	} else {
		p.assignments[tenant] = role
	}

	return true
}

func (p *AuthProviderBase) SaveAssignments(DB *db.DB, user *db.User) bool {
	who := fmt.Sprintf("%s (%s@%s)", user.Name, user.Account, user.Backend)

	user.SysRole = ""

	p.Infof("processing %d role assignments for %s", len(p.assignments), who)
	p.Infof("clearing pre-existing tenant assignments for %s", who)
	if err := DB.ClearMembershipsFor(user); err != nil {
		p.Errorf("failed to clear pre-existing tenant assignments for %s: %s", who, err)
		return false
	}

	for on, role := range p.assignments {
		if on == "SYSTEM" {
			user.SysRole = role

		} else {
			tenant, err := DB.EnsureTenant(on)
			p.Infof("ensuring that we have a tenant named '%s'", on)
			if err != nil {
				p.Errorf("failed to find/create tenant '%s': %s", on, err)
				return false
			}
			p.Infof("saving assignment of tenant role %s on '%s' to %s", role, on, who)
			err = DB.AddUserToTenant(user.UUID.String(), tenant.UUID.String(), role)
			if err != nil {
				p.Errorf("failed to assign tenant role %s on '%s' to %s: %s", role, on, who, err)
				return false
			}
		}
	}

	if err := DB.UpdateUser(user); err != nil {
		p.Errorf("unable to save %s system role assignment: %s", who, err)
		return false
	}
	return true
}
