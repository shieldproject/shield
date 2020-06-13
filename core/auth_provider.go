package core

import (
	"fmt"
	"net/http"

	"github.com/jhunt/go-log"

	"github.com/shieldproject/shield/db"
	"github.com/shieldproject/shield/route"
	"github.com/shieldproject/shield/util"
)

var (
	RoleTower map[string]int
)

func init() {
	RoleTower = map[string]int{
		"":         0,
		"operator": 1,
		"engineer": 2,
		"manager":  3,
		"admin":    4,
	}
}

type AuthProviderConfig struct {
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
	Type       string `json:"type"`

	WebEntry string `json:"web_entry"`
	CLIEntry string `json:"cli_entry"`
	Redirect string `json:"redirect"`

	Properties map[string]interface{} `json:"properties,omitempty"`
}

type AuthProvider interface {
	Configure(map[interface{}]interface{}) error
	Configuration(bool) AuthProviderConfig
	WireUpTo(core *Core)

	Initiate(*route.Request)
	HandleRedirect(*route.Request) *db.User
}

type AuthProviderBase struct {
	Name       string
	Identifier string
	Type       string

	core *Core

	properties map[string]interface{}

	assignment string
}

func (p AuthProviderBase) Configuration(private bool) AuthProviderConfig {
	cfg := AuthProviderConfig{
		Name:       p.Name,
		Identifier: p.Identifier,
		Type:       p.Type,

		WebEntry: fmt.Sprintf("/auth/%s/web", p.Identifier),
		CLIEntry: fmt.Sprintf("/auth/%s/cli", p.Identifier),
		Redirect: fmt.Sprintf("/auth/%s/redir", p.Identifier),
	}

	if private {
		cfg.Properties = util.StringifyKeys(p.properties).(map[string]interface{})
	}

	return cfg
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
	p.assignment = ""
}

func (p *AuthProviderBase) Assign(user *db.User, role string) bool {
	who := fmt.Sprintf("%s (%s@%s)", user.Name, user.Account, user.Backend)
	p.Infof("assigning system role %s to %s", role, who)
	if !IsValidSystemRole(role) {
		p.Errorf("unable to assign system role %s to %s: '%s' is not a valid system role", role, who, role)
		return false
	}

	if RoleTower[p.assignment] < RoleTower[role] {
		p.Infof("upgrading %s role assignment from %s -> %s", who, p.assignment, role)
		p.assignment = role
	}

	return true
}

func (p *AuthProviderBase) SaveAssignments(DB *db.DB, user *db.User) bool {
	who := fmt.Sprintf("%s (%s@%s)", user.Name, user.Account, user.Backend)

	user.SysRole = ""

	p.Infof("processing role assignment for %s", who)
	user.SysRole = p.assignment

	if err := DB.UpdateUser(user); err != nil {
		p.Errorf("unable to save %s system role assignment: %s", who, err)
		return false
	}
	return true
}
