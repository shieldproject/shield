package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/goutils/log"

	"github.com/starkandwayne/shield/db"
	"github.com/starkandwayne/shield/route"
	"github.com/starkandwayne/shield/util"
)

type v2AuthProvider struct {
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
	Type       string `json:"type"`
	WebEntry   string `json:"web_entry"`
	CLIEntry   string `json:"cli_entry"`
	Redirect   string `json:"redirect"`

	Properties map[string]interface{} `json:"properties,omitempty"`
}

type v2SystemArchive struct {
	UUID     uuid.UUID `json:"uuid"`
	Schedule string    `json:"schedule"`
	TakenAt  int64     `json:"taken_at"`
	Expiry   int       `json:"expiry"`
	Size     int       `json:"size"`
	OK       bool      `json:"ok"`
	Notes    string    `json:"notes"`
}
type v2SystemTask struct {
	UUID      uuid.UUID        `json:"uuid"`
	Type      string           `json:"type"`
	Status    string           `json:"status"`
	Owner     string           `json:"owner"`
	StartedAt int64            `json:"started_at"`
	OK        bool             `json:"ok"`
	Notes     string           `json:"notes"`
	Archive   *v2SystemArchive `json:"archive,omitempty"`
}
type v2SystemJob struct {
	UUID     uuid.UUID `json:"uuid"`
	Schedule string    `json:"schedule"`
	From     string    `json:"from"`
	To       string    `json:"to"`
	OK       bool      `json:"ok"`

	Store struct {
		UUID    uuid.UUID `json:"uuid"`
		Name    string    `json:"name"`
		Summary string    `json:"summary"`
		Plugin  string    `json:"plugin"`
	} `json:"store"`

	Keep struct {
		N    int `json:"n"`
		Days int `json:"days"`
	} `json:"keep"`

	Retention struct {
		UUID    uuid.UUID `json:"uuid"`
		Name    string    `json:"name"`
		Summary string    `json:"summary"`
		Days    int       `json:"days"`
	} `json:"retention"`
}
type v2System struct {
	UUID  uuid.UUID `json:"uuid"`
	Name  string    `json:"name"`
	Notes string    `json:"notes"`
	OK    bool      `json:"ok"`

	Jobs  []v2SystemJob  `json:"jobs"`
	Tasks []v2SystemTask `json:"tasks"`
}

type v2PatchAnnotation struct {
	Type        string `json:"type"`
	UUID        string `json:"uuid"`
	Disposition string `json:"disposition"`
	Notes       string `json:"notes"`
	Clear       string `json:"clear"`
}

type v2LocalTenant struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
	Role string `json:"role"`
}
type v2LocalUser struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Account string `json:"account"`
	SysRole string `json:"sysrole"`

	Tenants []v2LocalTenant `json:"tenants"`
}

func (core *Core) v2API() *route.Router {
	r := &route.Router{}

	r.Dispatch("GET /v2/health", func(r *route.Request) { // {{{
		health, err := core.checkHealth()
		if err != nil {
			r.Fail(route.Oops(err, "Unable to check SHIELD health"))
			return
		}
		r.OK(health)
	})
	// }}}

	r.Dispatch("POST /v2/init", func(r *route.Request) { // {{{
		var in struct {
			Master string `json:"master"`
		}
		if !r.Payload(&in) {
			return
		}

		if r.Missing("master", in.Master) {
			return
		}

		log.Infof("%s: initializing the SHIELD Core...", r)
		init, err := core.Initialize(in.Master)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to initialize the SHIELD Core"))
			return
		}
		if init {
			r.Fail(route.Bad(nil, "this SHIELD Core has already been initialized"))
			return
		}

		r.Success("Successfully initialized the SHIELD Core")
	})
	// }}}
	r.Dispatch("POST /v2/unlock", func(r *route.Request) { // {{{
		var in struct {
			Master string `json:"master"`
		}
		if !r.Payload(&in) {
			return
		}

		if r.Missing("master", in.Master) {
			return
		}

		init, err := core.Unlock(in.Master)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to unlock the SHIELD Core"))
			return
		}
		if !init {
			r.Fail(route.Bad(nil, "this SHIELD Core has not yet been initialized"))
			return
		}

		r.Success("Successfully unlocked the SHIELD Core")
	})
	// }}}
	r.Dispatch("POST /v2/rekey", func(r *route.Request) { // {{{
		var in struct {
			Current string `json:"current"`
			New     string `json:"new"`
		}
		if !r.Payload(&in) {
			return
		}

		if r.Missing("current", in.Current, "new", in.New) {
			return
		}

		err := core.Rekey(in.Current, in.New)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to rekey the SHIELD Core"))
			return
		}

		r.Success("Successfully rekeyed the SHIELD Core")
	})
	// }}}

	r.Dispatch("POST /v2/ui/users", func(r *route.Request) { // {{{
		var in struct {
			Search string `json:"search"`
		}
		if !r.Payload(&in) {
			return
		}
		if len(in.Search) < 3 {
			r.OK([]string{})
			return
		}

		users, err := core.DB.GetAllUsers(&db.UserFilter{
			Backend:    "local",
			Account:    in.Search,
			ExactMatch: false,
		})
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve users from the database."))
			return
		}
		r.OK(users)
	})
	// }}}

	r.Dispatch("GET /v2/auth/providers", func(r *route.Request) { // {{{
		l := make([]v2AuthProvider, 0)

		typ := r.Param("for", "cli")
		for _, auth := range core.auth {
			if auth.Backend == "token" && typ != "cli" {
				continue
			}
			l = append(l, v2AuthProvider{
				Name:       auth.Name,
				Identifier: auth.Identifier,
				Type:       auth.Backend,
				WebEntry:   fmt.Sprintf("/auth/%s/web", auth.Identifier),
				CLIEntry:   fmt.Sprintf("/auth/%s/cli", auth.Identifier),
				Redirect:   fmt.Sprintf("/auth/%s/redir", auth.Identifier),
			})
		}
		r.OK(l)
	})
	// }}}
	r.Dispatch("GET /v2/auth/providers/:name", func(r *route.Request) { // {{{
		for _, a := range core.auth {
			if a.Identifier == r.Args[1] {
				r.OK(&v2AuthProvider{
					Name:       a.Name,
					Identifier: a.Identifier,
					Type:       a.Backend,
					WebEntry:   fmt.Sprintf("/auth/%s/web", a.Identifier),
					CLIEntry:   fmt.Sprintf("/auth/%s/cli", a.Identifier),
					Redirect:   fmt.Sprintf("/auth/%s/redir", a.Identifier),

					Properties: util.StringifyKeys(a.Properties).(map[string]interface{}),
				})
				return
			}
		}
		r.Fail(route.NotFound(nil, "No such authentication provider: '%s'", r.Args[1]))
	})
	// }}}

	r.Dispatch("GET /v2/auth/local/users", func(r *route.Request) { // {{{
		limit := paramValue(r.Req, "limit", "")
		if invalidlimit(limit) {
			r.Fail(route.Bad(nil, "Invalid limit supplied: '%d'", limit))
			return
		}

		l, err := core.DB.GetAllUsers(&db.UserFilter{
			UUID:       paramValue(r.Req, "uuid", ""),
			Account:    paramValue(r.Req, "account", ""),
			SysRole:    paramValue(r.Req, "sysrole", ""),
			ExactMatch: paramEquals(r.Req, "exact", "t"),
			Backend:    "local",
			Limit:      limit,
		})
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve local users information"))
			return
		}

		users := make([]v2LocalUser, len(l))
		for i, user := range l {
			memberships, err := core.DB.GetMembershipsForUser(user.UUID)
			if err != nil {
				log.Errorf("failed to retrieve tenant memberships for user %s@%s (uuid %s): %s",
					user.Account, user.Backend, user.UUID.String(), err)
				r.Fail(route.Oops(err, "Unable to retrieve local users information"))
				return
			}

			users[i] = v2LocalUser{
				UUID:    user.UUID.String(),
				Name:    user.Name,
				Account: user.Account,
				SysRole: user.SysRole,
				Tenants: make([]v2LocalTenant, len(memberships)),
			}
			for j, membership := range memberships {
				users[i].Tenants[j].UUID = membership.TenantUUID.String()
				users[i].Tenants[j].Name = membership.TenantName
				users[i].Tenants[j].Role = membership.Role
			}
		}

		r.OK(users)
	})
	// }}}
	r.Dispatch("GET /v2/auth/local/users/:uuid", func(r *route.Request) { // {{{
		user, err := core.DB.GetUserByID(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve local user information"))
			return
		}

		if user == nil {
			r.Fail(route.NotFound(nil, "user '%s' not found (for local auth provider)", r.Args[1]))
			return
		}

		memberships, err := core.DB.GetMembershipsForUser(user.UUID)
		if err != nil {
			log.Errorf("failed to retrieve tenant memberships for user %s@%s (uuid %s): %s",
				user.Account, user.Backend, user.UUID.String(), err)
			r.Fail(route.Oops(err, "Unable to retrieve local users information"))
			return
		}

		local_user := v2LocalUser{
			UUID:    user.UUID.String(),
			Name:    user.Name,
			Account: user.Account,
			SysRole: user.SysRole,
			Tenants: make([]v2LocalTenant, len(memberships)),
		}

		for j, membership := range memberships {
			local_user.Tenants[j].UUID = membership.TenantUUID.String()
			local_user.Tenants[j].Name = membership.TenantName
			local_user.Tenants[j].Role = membership.Role
		}

		r.OK(local_user)
	})
	// }}}
	r.Dispatch("POST /v2/auth/local/users", func(r *route.Request) { // {{{
		var in struct {
			UUID     string `json:"uuid"`
			Name     string `json:"name"`
			Account  string `json:"account"`
			Password string `json:"password"`
			SysRole  string `json:"sysrole"`
		}
		if !r.Payload(&in) {
			return
		}

		if r.Missing("name", in.Name, "account", in.Account, "password", in.Password) {
			return
		}

		var id uuid.UUID
		if in.UUID != "" {
			id = uuid.Parse(in.UUID)
		}

		if in.SysRole != "" {
			switch in.SysRole {
			case
				"admin",
				"manager",
				"technician":
			default:
				r.Fail(route.Bad(nil, "System Role '%s' is invalid", in.SysRole))
				return
			}
		}

		u := &db.User{
			UUID:    id,
			Name:    in.Name,
			Account: in.Account,
			Backend: "local",
			SysRole: in.SysRole,
		}
		u.SetPassword(in.Password)

		exists, err := core.DB.GetUser(u.Account, "local")
		if err != nil {
			r.Fail(route.Oops(err, "Unable to create local user '%s' (local auth provider)", in.Name))
			return
		}

		if exists != nil {
			r.Fail(route.Bad(nil, "user '%s' already exists (for local auth provider)", u.Account))
			return
		}

		new_user, err := core.DB.CreateUser(u)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to create local user '%s' (local auth provider)", in.Name))
			return
		}
		r.OK(new_user)
	})
	// }}}
	r.Dispatch("PATCH /v2/auth/local/users/:uuid", func(r *route.Request) { // {{{
		/* FIXME rules for updating accounts:
		   1. you can update your own account (except for sysrole)
		   2. managers can update technicians and ''
		   3. admins can update managers, technicians and ''
		*/
		var in struct {
			Name     string `json:"name"`
			Password string `json:"password"`
			SysRole  string `json:"sysrole"`
		}
		if !r.Payload(&in) {
			return
		}

		user, err := core.DB.GetUserByID(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to update local user information"))
			return
		}
		if user == nil || user.Backend != "local" {
			r.Fail(route.NotFound(nil, "Local User '%s' not found", r.Args[1]))
			return
		}
		if in.Name != "" {
			user.Name = in.Name
		}

		if in.SysRole != "" {
			switch in.SysRole {
			case
				"admin",
				"manager",
				"technician":
				user.SysRole = in.SysRole
			default:
				r.Fail(route.Bad(nil, "System Role '%s' is invalid", in.SysRole))
				return
			}
		}

		if in.Password != "" {
			user.SetPassword(in.Password)
		}

		err = core.DB.UpdateUser(user)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to update local user '%s' (local auth provider)", in.Name))
			return
		}

		r.Success("Updated")
	})
	// }}}
	r.Dispatch("DELETE /v2/auth/local/users/:uuid", func(r *route.Request) { // {{{
		/* FIXME rules for deleting accounts:
		   1. you cannot delete your own account
		   2. managers can delete technicians and ''
		   3. admins can delete managers, technicians and ''
		*/
		user, err := core.DB.GetUserByID(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve local user information"))
			return
		}
		if user == nil || user.Backend != "local" {
			r.Fail(route.NotFound(nil, "Local User '%s' not found", r.Args[1]))
			return
		}

		err = core.DB.DeleteUser(user)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to delete local user '%s' (%s)", r.Args[1], user.Account))
			return
		}
		r.Success("Successfully deleted user '%s' (%s@local)", r.Args[1], user.Account)
	})
	// }}}

	r.Dispatch("GET /v2/systems", func(r *route.Request) { // {{{
		targets, err := core.DB.GetAllTargets(
			&db.TargetFilter{
				SkipUsed:   r.ParamIs("unused", "t"),
				SkipUnused: r.ParamIs("unused", "f"),
				SearchName: r.Param("name", ""),
				ForPlugin:  r.Param("plugin", ""),
				ExactMatch: r.ParamIs("exact", "t"),
			},
		)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve systems information"))
			return
		}

		systems := make([]v2System, len(targets))
		for i, target := range targets {
			err := core.v2copyTarget(&systems[i], target)
			if err != nil {
				r.Fail(route.Oops(err, "Unable to retrieve systems information"))
				return
			}
		}

		r.OK(systems)
	})
	// }}}
	r.Dispatch("GET /v2/systems/:uuid", func(r *route.Request) { // {{{
		log.Debugf("%s: got args [%v]", r, r.Args)
		target, err := core.DB.GetTarget(uuid.Parse(r.Args[1]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve system information"))
			return
		}

		if target == nil {
			r.Fail(route.NotFound(err, "system %s not found", r.Args[1]))
			return
		}

		var system v2System
		err = core.v2copyTarget(&system, target)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve system information"))
			return
		}

		// keep track of our archives, indexed by task UUID
		archives := make(map[string]*db.Archive)
		aa, err := core.DB.GetAllArchives(
			&db.ArchiveFilter{
				ForTarget:  target.UUID.String(),
				WithStatus: []string{"valid"},
			},
		)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve system information"))
			return
		}
		for _, archive := range aa {
			archives[archive.UUID.String()] = archive
		}

		tasks, err := core.DB.GetAllTasks(
			&db.TaskFilter{
				ForTarget:    target.UUID.String(),
				OnlyRelevant: true,
			},
		)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve system information"))
			return
		}
		system.Tasks = make([]v2SystemTask, len(tasks))
		for i, task := range tasks {
			system.Tasks[i].UUID = task.UUID
			system.Tasks[i].Type = task.Op
			system.Tasks[i].Status = task.Status
			system.Tasks[i].Owner = task.Owner
			system.Tasks[i].OK = task.OK
			system.Tasks[i].Notes = task.Notes

			if t := task.StartedAt.Time(); t.IsZero() {
				system.Tasks[i].StartedAt = 0
			} else {
				system.Tasks[i].StartedAt = t.Unix()
			}

			if archive, ok := archives[task.ArchiveUUID.String()]; ok {
				system.Tasks[i].Archive = &v2SystemArchive{
					UUID:     archive.UUID,
					Schedule: archive.Job,
					Expiry:   (int)((archive.ExpiresAt.Time().Unix() - archive.TakenAt.Time().Unix()) / 86400),
					Notes:    archive.Notes,
					Size:     -1, // FIXME
				}
			}
		}

		r.OK(system)
	})
	// }}}
	r.Dispatch("POST /v2/systems", func(r *route.Request) { // {{{
		/* FIXME */
		r.Fail(route.Errorf(501, nil, "%s: not implemented", r))
	})
	// }}}
	r.Dispatch("PUT /v2/systems/:uuid", func(r *route.Request) { // {{{
		/* FIXME */
		r.Fail(route.Errorf(501, nil, "%s: not implemented", r))
	})
	// }}}
	r.Dispatch("PATCH /v2/systems/:uuid", func(r *route.Request) { // {{{
		var in struct {
			Annotations []v2PatchAnnotation `json:"annotations"`
		}
		if !r.Payload(&in) {
			return
		}

		target, err := core.DB.GetTarget(uuid.Parse(r.Args[1]))
		if err != nil {
			r.Fail(route.Bad(err, "invalid or malformed target UUID: '%s'", r.Args[1]))
			return
		}

		for _, ann := range in.Annotations {
			switch ann.Type {
			case "task":
				err = core.DB.AnnotateTargetTask(
					target.UUID,
					ann.UUID,
					&db.TaskAnnotation{
						Disposition: ann.Disposition,
						Notes:       ann.Notes,
						Clear:       ann.Clear,
					},
				)
				if err != nil {
					r.Fail(route.Oops(err, "Unable to annotate task %s", ann.UUID))
					return
				}

			case "archive":
				err = core.DB.AnnotateTargetArchive(
					target.UUID,
					ann.UUID,
					ann.Notes,
				)
				if err != nil {
					r.Fail(route.Oops(err, "Unable to annotate archive %s", ann.UUID))
					return
				}

			default:
				r.Fail(route.Bad(nil, "unrecognized system annotation type '%s'", ann.Type))
				return
			}
		}

		_ = core.DB.MarkTasksIrrelevant()
		r.Success("annotated successfully")
	})
	// }}}
	r.Dispatch("DELETE /v2/systems/:uuid", func(r *route.Request) { // {{{
		/* FIXME */
		r.Fail(route.Errorf(501, nil, "%s: not implemented", r))
	})
	// }}}

	r.Dispatch("GET /v2/agents", func(r *route.Request) { // {{{
		agents, err := core.DB.GetAllAgents(nil)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve agent information"))
			return
		}

		resp := struct {
			Agents   []*db.Agent         `json:"agents"`
			Problems map[string][]string `json:"problems"`
		}{
			Agents:   agents,
			Problems: make(map[string][]string),
		}

		for _, agent := range agents {
			id := agent.UUID.String()
			pp := make([]string, 0)

			if agent.Version == "" {
				pp = append(pp, Problems["legacy-shield-agent-version"])
			}
			if agent.Version == "dev" {
				pp = append(pp, Problems["dev-shield-agent-version"])
			}

			resp.Problems[id] = pp
		}
		r.OK(resp)
	})
	// }}}
	r.Dispatch("GET /v2/agents/:uuid", func(r *route.Request) { // {{{
		agentID := uuid.Parse(r.Args[1])
		if agentID == nil {
			r.Fail(route.Bad(nil, "Invalid Agent UUID"))
			return
		}

		agent, err := core.DB.GetAgent(agentID)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve agent information"))
			return
		}
		if agent == nil {
			r.Fail(route.NotFound(nil, "No such agent"))
			return
		}

		var raw map[string]interface{}
		if err = json.Unmarshal([]byte(agent.Metadata), &raw); err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve agent information"))
			return
		}

		resp := struct {
			Agent    db.Agent               `json:"agent"`
			Metadata map[string]interface{} `json:"metadata"`
			Problems []string               `json:"problems"`
		}{
			Agent:    *agent,
			Metadata: raw,
			Problems: make([]string, 0),
		}

		if agent.Version == "" {
			resp.Problems = append(resp.Problems, Problems["legacy-shield-agent-version"])
		}
		if agent.Version == "dev" {
			resp.Problems = append(resp.Problems, Problems["dev-shield-agent-version"])
		}

		r.OK(resp)
	})
	// }}}
	r.Dispatch("POST /v2/agents", func(r *route.Request) { // {{{
		var in struct {
			Name string `json:"name"`
			Port int    `json:"port"`
		}
		if !r.Payload(&in) {
			return
		}

		peer := regexp.MustCompile(`:\d+$`).ReplaceAllString(r.Req.RemoteAddr, "")
		if peer == "" {
			r.Fail(route.Oops(nil, "Unable to determine remote peer address from '%s'", r.Req.RemoteAddr))
			return
		}

		if in.Name == "" {
			r.Fail(route.Bad(nil, "No `name' provided with pre-registration request"))
			return
		}
		if in.Port == 0 {
			r.Fail(route.Bad(nil, "No `port' provided with pre-registration request"))
			return
		}

		err := core.DB.PreRegisterAgent(peer, in.Name, in.Port)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to pre-register agent %s at %s:%i", in.Name, peer, in.Port))
			return
		}
		r.Success("pre-registered agent %s at %s:%i", in.Name, peer, in.Port)
	})
	// }}}

	r.Dispatch("GET /v2/tenants", func(r *route.Request) { // {{{
		tenants, err := core.DB.GetAllTenants()
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve tenants information"))
			return
		}
		r.OK(tenants)
	})
	// }}}
	r.Dispatch("POST /v2/tenants", func(r *route.Request) { // {{{
		var in struct {
			UUID string `json:"uuid"`
			Name string `json:"name"`

			Users []struct {
				UUID    string `json:"uuid"`
				Account string `json:"account"`
				Role    string `json:"role"`
			} `json:"users"`
		}
		if !r.Payload(&in) {
			return
		}

		if r.Missing("name", in.Name) {
			return
		}

		if strings.ToLower(in.Name) == "system" {
			r.Fail(route.Bad(nil, "tenant name 'system' is reserved"))
			return
		}

		t, err := core.DB.CreateTenant(in.UUID, in.Name)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to create new tenant '%s'", in.Name))
			return
		}

		for _, u := range in.Users {
			user, err := core.DB.GetUserByID(u.UUID)
			if err != nil {
				r.Fail(route.Oops(err, "Unrecognized user account '%s'", user))
				return
			}

			if user == nil {
				r.Fail(route.Oops(err, "Unrecognized user account '%s'", user))
				return
			}

			if user.Backend != "local" {
				r.Fail(route.Oops(nil, "Unable to invite '%s@%s' to tenant '%s' - only local users can be invited.", user.Account, user.Backend, t.Name))
				return
			}

			err = core.DB.AddUserToTenant(u.UUID, t.UUID.String(), u.Role)
			if err != nil {
				r.Fail(route.Oops(err, "Unable to invite '%s' to tenant '%s'", user.Account, t.Name))
				return
			}
		}

		r.OK(t)
	})
	// }}}
	r.Dispatch("GET /v2/tenants/:uuid", func(r *route.Request) { // {{{
		tenant, err := core.DB.GetTenant(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve tenant information"))
			return
		}
		if tenant == nil {
			r.Fail(route.NotFound(nil, "No such tenant"))
			return
		}

		tenant.Members, err = core.DB.GetUsersForTenant(tenant.UUID)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve tenant memberships information"))
			return
		}

		r.OK(tenant)
	})
	// }}}
	r.Dispatch("PUT /v2/tenants/:uuid", func(r *route.Request) { // {{{
		var in struct {
			Name string `json:"name"`
		}
		if !r.Payload(&in) {
			return
		}

		if r.Missing("name", in.Name) {
			return
		}

		t, err := core.DB.UpdateTenant(r.Args[1], in.Name)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to update tenant '%s'", in.Name))
			return
		}
		r.OK(t)
	})
	// }}}
	r.Dispatch("DELETE /v2/tenants/:uuid", func(r *route.Request) { // {{{
		/* FIXME */
		r.Fail(route.Errorf(501, nil, "%s: not implemented", r))
	})
	// }}}
	r.Dispatch("POST /v2/tenants/:uuid/invite", func(r *route.Request) { // {{{
		tenant, err := core.DB.GetTenant(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve tenant information"))
			return
		}
		if tenant == nil {
			r.Fail(route.NotFound(nil, "No such tenant"))
			return
		}

		var in struct {
			Users []struct {
				UUID    string `json:"uuid"`
				Account string `json:"account"`
				Role    string `json:"role"`
			} `json:"users"`
		}
		if !r.Payload(&in) {
			return
		}

		for _, u := range in.Users {
			user, err := core.DB.GetUserByID(u.UUID)
			if err != nil {
				r.Fail(route.Oops(err, "Unrecognized user account '%s'", user))
				return
			}

			if user == nil {
				r.Fail(route.Oops(err, "Unrecognized user account '%s'", user))
				return
			}

			if user.Backend != "local" {
				r.Fail(route.Oops(nil, "Unable to invite '%s@%s' to tenant '%s' - only local users can be invited.", user.Account, user.Backend, tenant.Name))
				return
			}

			err = core.DB.AddUserToTenant(u.UUID, tenant.UUID.String(), u.Role)
			if err != nil {
				r.Fail(route.Oops(err, "Unable to invite '%s' to tenant '%s'", user.Account, tenant.Name))
				return
			}
		}

		r.Success("Invitations sent")
	})
	// }}}
	r.Dispatch("POST /v2/tenants/:uuid/banish", func(r *route.Request) { // {{{
		tenant, err := core.DB.GetTenant(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve tenant information"))
			return
		}
		if tenant == nil {
			r.Fail(route.NotFound(nil, "No such tenant"))
			return
		}

		var in struct {
			Users []struct {
				UUID    string `json:"uuid"`
				Account string `json:"account"`
			} `json:"users"`
		}
		if !r.Payload(&in) {
			return
		}

		for _, u := range in.Users {
			user, err := core.DB.GetUserByID(u.UUID)
			if err != nil {
				r.Fail(route.Oops(err, "Unrecognized user account '%s'", user))
				return
			}

			if user == nil {
				r.Fail(route.Oops(err, "Unrecognized user account '%s'", user))
				return
			}

			if user.Backend != "local" {
				r.Fail(route.Oops(nil, "Unable to banish '%s@%s' from tenant '%s' - only local users can be banished.", user.Account, user.Backend, tenant.Name))
				return
			}

			err = core.DB.RemoveUserFromTenant(u.UUID, tenant.UUID.String())
			if err != nil {
				r.Fail(route.Oops(err, "Unable to banish '%s' from tenant '%s'", user.Account, tenant.Name))
				return
			}
		}

		r.Success("Banishments served")
	})
	// }}}

	r.Dispatch("GET /v2/tenants/:uuid/targets", func(r *route.Request) { // {{{
		targets, err := core.DB.GetAllTargets(
			&db.TargetFilter{
				ForTenant:  r.Args[1],
				SkipUsed:   r.ParamIs("unused", "t"),
				SkipUnused: r.ParamIs("unused", "f"),
				SearchName: r.Param("name", ""),
				ForPlugin:  r.Param("plugin", ""),
				ExactMatch: r.ParamIs("exact", "t"),
			},
		)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve targets information"))
			return
		}

		r.OK(targets)
	})
	// }}}
	r.Dispatch("POST /v2/tenants/:uuid/targets", func(r *route.Request) { // {{{
		tenant, err := core.DB.GetTenant(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve tenant information"))
			return
		}
		if tenant == nil {
			r.Fail(route.NotFound(nil, "No such tenant"))
			return
		}

		var in struct {
			Name     string `json:"name"`
			Summary  string `json:"summary"`
			Plugin   string `json:"plugin"`
			Endpoint string `json:"endpoint"`
			Agent    string `json:"agent"`
		}

		if !r.Payload(&in) {
			return
		}
		if r.Missing("name", in.Name, "plugin", in.Plugin, "endpoint", in.Endpoint, "agent", in.Agent) {
			return
		}

		target, err := core.DB.CreateTarget(&db.Target{
			Name:     in.Name,
			Summary:  in.Summary,
			Plugin:   in.Plugin,
			Endpoint: in.Endpoint,
			Agent:    in.Agent,
		})
		if err != nil {
			r.Fail(route.Oops(err, "Unable to create new data target"))
			return
		}

		r.OK(target)
	})
	// }}}
	r.Dispatch("GET /v2/tenants/:uuid/targets/:uuid", func(r *route.Request) { // {{{
		target, err := core.DB.GetTarget(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve target information"))
			return
		}

		if target == nil || target.TenantUUID.String() != r.Args[1] {
			r.Fail(route.NotFound(nil, "No such target"))
			return
		}

		r.OK(target)
	})
	// }}}
	r.Dispatch("PUT /v2/tenants/:uuid/targets/:uuid", func(r *route.Request) { // {{{
		target, err := core.DB.GetTarget(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve target information"))
			return
		}

		if target == nil || target.TenantUUID.String() != r.Args[1] {
			r.Fail(route.NotFound(nil, "No such target"))
			return
		}

		var in struct {
			Name     string `json:"name"`
			Summary  string `json:"summary"`
			Plugin   string `json:"plugin"`
			Endpoint string `json:"endpoint"`
			Agent    string `json:"agent"`
		}
		if !r.Payload(&in) {
			return
		}

		if in.Name != "" {
			target.Name = in.Name
		}
		if in.Summary != "" {
			target.Summary = in.Summary
		}
		if in.Plugin != "" {
			target.Plugin = in.Plugin
		}
		if in.Endpoint != "" {
			target.Endpoint = in.Endpoint
		}
		if in.Agent != "" {
			target.Agent = in.Agent
		}

		if err := core.DB.UpdateTarget(target); err != nil {
			r.Fail(route.Oops(err, "Unable to update target"))
			return
		}

		r.Success("Updated target successfully")
	})
	// }}}
	r.Dispatch("DELETE /v2/tenants/:uuid/targets/:uuid", func(r *route.Request) { // {{{
		target, err := core.DB.GetTarget(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve target information"))
			return
		}

		if target == nil || target.TenantUUID.String() != r.Args[1] {
			r.Fail(route.NotFound(nil, "No such target"))
			return
		}

		deleted, err := core.DB.DeleteTarget(target.UUID)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to delete target"))
			return
		}
		if !deleted {
			r.Fail(route.Forbidden(nil, "The target cannot be deleted at this time"))
			return
		}

		r.Success("Target deleted successfully")
	})
	// }}}

	r.Dispatch("GET /v2/tenants/:uuid/policies", func(r *route.Request) { // {{{
		policies, err := core.DB.GetAllRetentionPolicies(
			&db.RetentionFilter{
				ForTenant:  r.Args[1],
				SkipUsed:   r.ParamIs("unused", "t"),
				SkipUnused: r.ParamIs("unused", "f"),
				SearchName: r.Param("name", ""),
				ExactMatch: r.ParamIs("exact", "t"),
			},
		)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve retention policies information"))
			return
		}

		r.OK(policies)
	})
	// }}}
	r.Dispatch("POST /v2/tenants/:uuid/policies", func(r *route.Request) { // {{{
		var in struct {
			Name    string `json:"name"`
			Summary string `json:"summary"`
			Expires uint   `json:"expires"`
		}
		if !r.Payload(&in) {
			return
		}

		if r.Missing("name", in.Name) {
			return
		}

		/* FIXME: for v2, flip expires over to days, not seconds */
		if in.Expires < 86400 {
			r.Fail(route.Bad(nil, "Retention policy expiry must be greater than 1 day"))
			return
		}

		policy, err := core.DB.CreateRetentionPolicy(&db.RetentionPolicy{
			Name:    in.Name,
			Summary: in.Summary,
			Expires: in.Expires,
		})
		if err != nil {
			r.Fail(route.Oops(err, "Unable to create retention policy"))
			return
		}

		r.OK(policy)
	})
	// }}}
	r.Dispatch("GET /v2/tenants/:uuid/policies/:uuid", func(r *route.Request) { // {{{
		policy, err := core.DB.GetRetentionPolicy(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve retention policy information"))
			return
		}

		if policy == nil || policy.TenantUUID.String() != r.Args[1] {
			r.Fail(route.NotFound(nil, "No such retention policy"))
			return
		}

		r.OK(policy)
	})
	// }}}
	r.Dispatch("PUT /v2/tenants/:uuid/policies/:uuid", func(r *route.Request) { // {{{
		var in struct {
			Name    string `json:"name"`
			Summary string `json:"summary"`
			Expires uint   `json:"expires"`
		}
		if !r.Payload(&in) {
			return
		}

		policy, err := core.DB.GetRetentionPolicy(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve retention policy information"))
			return
		}

		if policy == nil || policy.TenantUUID.String() != r.Args[1] {
			r.Fail(route.NotFound(nil, "No such retention policy"))
			return
		}

		if in.Name != "" {
			policy.Name = in.Name
		}
		if in.Summary != "" {
			policy.Summary = in.Name
		}
		if in.Expires != 0 {
			policy.Expires = in.Expires
		}

		if err := core.DB.UpdateRetentionPolicy(policy); err != nil {
			r.Fail(route.Oops(err, "Unable to update retention policy"))
			return
		}

		r.Success("Updated retention policy successfully")
	})
	// }}}
	r.Dispatch("DELETE /v2/tenants/:uuid/policies/:uuid", func(r *route.Request) { // {{{
		policy, err := core.DB.GetRetentionPolicy(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve retention policy information"))
			return
		}

		if policy == nil || policy.TenantUUID.String() != r.Args[1] {
			r.Fail(route.NotFound(nil, "No such retention policy"))
			return
		}

		deleted, err := core.DB.DeleteRetentionPolicy(policy.UUID)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to delete retention policy"))
			return
		}
		if !deleted {
			r.Fail(route.Forbidden(nil, "The retention policy cannot be deleted at this time"))
			return
		}

		r.Success("Retention policy deleted successfully")
	})
	// }}}

	r.Dispatch("GET /v2/tenants/:uuid/stores", func(r *route.Request) { // {{{
		stores, err := core.DB.GetAllStores(
			&db.StoreFilter{
				SkipUsed:   r.ParamIs("unused", "t"),
				SkipUnused: r.ParamIs("unused", "f"),
				SearchName: r.Param("name", ""),
				ForPlugin:  r.Param("plugin", ""),
				ExactMatch: r.ParamIs("exact", "t"),
				ForTenant:  r.Args[1],
			},
		)
		if err != nil {
			r.Fail(route.Oops(err, "FIXME need an error message"))
			return
		}

		r.OK(stores)
	})
	// }}}
	r.Dispatch("GET /v2/tenants/:uuid/stores/:uuid", func(r *route.Request) { // {{{
		tennant_id := uuid.Parse(r.Args[1])
		store_id := uuid.Parse(r.Args[2])
		if store_id == nil || tennant_id == nil {
			r.Fail(route.Bad(nil, "Invalid UUID speficied"))
			return
		}
		store, err := core.DB.GetStore(store_id)
		if err != nil {
			r.Fail(route.Oops(err, "FIXME need an error message"))
			return
		}

		if store == nil || strings.Compare(store.TenantUUID.String(), tennant_id.String()) != 0 {
			r.Fail(route.NotFound(nil, "Store Not Found"))
			return
		}

		r.OK(store)
	})
	// }}}""
	r.Dispatch("POST /v2/tenants/:uuid/stores", func(r *route.Request) { // {{{
		if r.Req.Body == nil {
			r.Fail(route.Bad(nil, "FIXME need an error message"))
			return
		}
		tennant_id := uuid.Parse(r.Args[1])
		if tennant_id == nil {
			r.Fail(route.Bad(nil, "Invalid UUID speficied"))
			return
		}

		var params struct {
			Name     string `json:"name"`
			Summary  string `json:"summary"`
			Plugin   string `json:"plugin"`
			Endpoint string `json:"endpoint"`
		}
		if err := json.NewDecoder(r.Req.Body).Decode(&params); err != nil && err != io.EOF {
			r.Fail(route.Oops(err, "FIXME need an error message"))
			return
		}

		e := MissingParameters()
		e.Check("name", params.Name)
		e.Check("plugin", params.Plugin)
		e.Check("endpoint", params.Endpoint)
		if e.IsValid() {
			r.Fail(route.Oops(errors.New(e.Error()), e.Error()))
			return
		}

		id, err := core.DB.CreateStore(&db.Store{
			Name:       params.Name,
			Plugin:     params.Plugin,
			Endpoint:   params.Endpoint,
			Summary:    params.Summary,
			TenantUUID: tennant_id,
		})
		if err != nil {
			r.Fail(route.Oops(err, "FIXME need an error message"))
			return
		}

		r.OK(fmt.Sprintf("\"created\", \"uuid\":\"%s\"", id.String()))
	})
	// }}}
	r.Dispatch("PUT /v2/tenants/:uuid/stores/:uuid", func(r *route.Request) { // {{{
		if r.Req.Body == nil {
			r.Fail(route.Bad(nil, "FIXME need an error message"))
			return
		}

		tennant_id := uuid.Parse(r.Args[1])
		store_id := uuid.Parse(r.Args[2])
		if store_id == nil || tennant_id == nil {
			r.Fail(route.Bad(nil, "Invalid UUID speficied"))
			return
		}

		var params struct {
			Name     string `json:"name"`
			Summary  string `json:"summary"`
			Plugin   string `json:"plugin"`
			Endpoint string `json:"endpoint"`
		}
		if err := json.NewDecoder(r.Req.Body).Decode(&params); err != nil && err != io.EOF {
			r.Fail(route.Oops(err, "FIXME need an error message"))
			return
		}

		store, err := core.DB.GetStore(store_id)
		if err != nil {
			r.Fail(route.Oops(err, "FIXME need an error message"))
			return
		}
		if store == nil || strings.Compare(store.TenantUUID.String(), tennant_id.String()) != 0 {
			r.Fail(route.NotFound(nil, "Store Not Found"))
			return
		}

		if params.Name != "" {
			store.Name = params.Name
		}

		if params.Summary != "" {
			store.Summary = params.Summary
		}

		if params.Plugin != "" {
			store.Plugin = params.Plugin
		}

		if params.Endpoint != "" {
			store.Endpoint = params.Endpoint
		}

		if err := core.DB.UpdateStore(store); err != nil {
			r.Fail(route.Oops(err, "FIXME need an error message"))
			return
		}

		r.OK(fmt.Sprintf("updated"))
	})
	// }}}
	r.Dispatch("DELETE /v2/tenants/:uuid/stores/:uuid", func(r *route.Request) { // {{{
		tennant_id := uuid.Parse(r.Args[1])
		store_id := uuid.Parse(r.Args[2])
		if store_id == nil || tennant_id == nil {
			r.Fail(route.Bad(nil, "Invalid UUID speficied"))
			return
		}

		store, err := core.DB.GetStore(store_id)
		if err != nil {
			r.Fail(route.Oops(err, "FIXME need an error message"))
			return
		}
		if store == nil || strings.Compare(store.TenantUUID.String(), tennant_id.String()) != 0 {
			r.Fail(route.NotFound(nil, "Store Not Found"))
			return
		}

		deleted, err := core.DB.DeleteStore(store.UUID)

		if err != nil {
			r.Fail(route.Oops(err, "FIXME need an error message"))
			return
		}
		if !deleted {
			r.Fail(route.Bad(err, "FIXME need an error message"))
			return
		}

		r.OK("deleted")
	})
	// }}}

	r.Dispatch("GET /v2/tenants/:uuid/jobs", func(r *route.Request) { // {{{
		jobs, err := core.DB.GetAllJobs(
			&db.JobFilter{
				ForTenant:    r.Args[1],
				SkipPaused:   r.ParamIs("paused", "f"),
				SkipUnpaused: r.ParamIs("paused", "t"),

				SearchName: r.Param("name", ""),

				ForTarget:    r.Param("target", ""),
				ForStore:     r.Param("store", ""),
				ForRetention: r.Param("retention", ""),
				ExactMatch:   r.ParamIs("exact", "t"),
			},
		)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve tenant job information."))
			return
		}

		r.OK(jobs)
	})
	// }}}
	r.Dispatch("POST /v2/tenants/:uuid/jobs", func(r *route.Request) { // {{{
		var in struct {
			Name     string `json:"name"`
			Summary  string `json:"summary"`
			Schedule string `json:"schedule"`
			Paused   bool   `json:"paused"`

			StoreUUID     string `json:"store"`
			TargetUUID    string `json:"target"`
			RetentionUUID string `json:"retention"`
		}
		if !r.Payload(&in) {
			return
		}

		if r.Missing("name", in.Name, "store", in.StoreUUID, "target", in.TargetUUID, "schedule", in.Schedule, "retention", in.RetentionUUID) {
			return
		}

		job, err := core.DB.CreateJob(&db.Job{
			TenantUUID:    uuid.Parse(r.Args[1]),
			Name:          in.Name,
			Summary:       in.Summary,
			Schedule:      in.Schedule,
			Paused:        in.Paused,
			StoreUUID:     uuid.Parse(in.StoreUUID),
			TargetUUID:    uuid.Parse(in.TargetUUID),
			RetentionUUID: uuid.Parse(in.RetentionUUID),
		})
		if err != nil {
			r.Fail(route.Oops(err, "Unable to create new job"))
			return
		}

		r.OK(job)
	})
	// }}}
	r.Dispatch("GET /v2/tenants/:uuid/jobs/:uuid", func(r *route.Request) { // {{{
		job, err := core.DB.GetJob(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve job information"))
			return
		}

		if job == nil || job.TenantUUID.String() != r.Args[1] {
			r.Fail(route.NotFound(nil, "No such job"))
			return
		}

		r.OK(job)
	})
	// }}}
	r.Dispatch("PUT /v2/tenants/:uuid/jobs/:uuid", func(r *route.Request) { // {{{
		var in struct {
			Name     string `json:"name"`
			Summary  string `json:"summary"`
			Schedule string `json:"schedule"`

			StoreUUID     string `json:"store"`
			TargetUUID    string `json:"target"`
			RetentionUUID string `json:"retention"`
		}
		if !r.Payload(&in) {
			return
		}

		job, err := core.DB.GetJob(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve job information"))
			return
		}
		if job == nil || job.TenantUUID.String() != r.Args[1] {
			r.Fail(route.NotFound(nil, "No such job"))
			return
		}

		if in.Name != "" {
			job.Name = in.Name
		}
		if in.Summary != "" {
			job.Summary = in.Summary
		}
		if in.Schedule != "" {
			job.Schedule = in.Schedule
		}
		if in.TargetUUID != "" {
			job.TargetUUID = uuid.Parse(in.TargetUUID)
		}
		if in.StoreUUID != "" {
			job.StoreUUID = uuid.Parse(in.StoreUUID)
		}
		if in.RetentionUUID != "" {
			job.RetentionUUID = uuid.Parse(in.RetentionUUID)
		}

		if err := core.DB.UpdateJob(job); err != nil {
			r.Fail(route.Oops(err, "Unable to update job"))
			return
		}

		r.Success("Updated job successfully")
	})
	// }}}
	r.Dispatch("DELETE /v2/tenants/:uuid/jobs/:uuid", func(r *route.Request) { // {{{
		job, err := core.DB.GetJob(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve job information"))
			return
		}

		if job == nil || job.TenantUUID.String() != r.Args[1] {
			r.Fail(route.NotFound(nil, "No such job"))
			return
		}

		deleted, err := core.DB.DeleteJob(job.UUID)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to delete job"))
			return
		}
		if !deleted {
			r.Fail(route.Forbidden(nil, "The job cannot be deleted at this time"))
			return
		}

		r.Success("Job deleted successfully")
	})
	// }}}
	r.Dispatch("POST /v2/tenants/:uuid/jobs/:uuid/run", func(r *route.Request) { // {{{
		job, err := core.DB.GetJob(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve job information"))
			return
		}

		if job == nil || job.TenantUUID.String() != r.Args[1] {
			r.Fail(route.NotFound(nil, "No such job"))
			return
		}

		var in struct {
			Owner string `json:"owner"`
		}
		if !r.Payload(&in) {
			return
		}

		if in.Owner == "" {
			in.Owner = "anon"
		}

		task, err := core.DB.CreateBackupTask(in.Owner, job)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to schedule ad hoc backup job run"))
			return
		}

		var out struct {
			OK       string    `json:"ok"`
			TaskUUID uuid.UUID `json:"task_uuid"`
		}

		out.OK = "Scheduled ad hoc backup job run"
		out.TaskUUID = task.UUID
		r.OK(out)
	})
	// }}}
	r.Dispatch("POST /v2/tenants/:uuid/jobs/:uuid/pause", func(r *route.Request) { // {{{
		job, err := core.DB.GetJob(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve job information"))
			return
		}

		if job == nil || job.TenantUUID.String() != r.Args[1] {
			r.Fail(route.NotFound(nil, "No such job"))
			return
		}

		if _, err = core.DB.PauseJob(job.UUID); err != nil {
			r.Fail(route.Oops(err, "Unable to pause job"))
			return
		}
		r.Success("Paused job successfully")
	})
	// }}}
	r.Dispatch("POST /v2/tenants/:uuid/jobs/:uuid/unpause", func(r *route.Request) { // {{{
		job, err := core.DB.GetJob(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve job information"))
			return
		}

		if job == nil || job.TenantUUID.String() != r.Args[1] {
			r.Fail(route.NotFound(nil, "No such job"))
			return
		}

		if _, err = core.DB.UnpauseJob(job.UUID); err != nil {
			r.Fail(route.Oops(err, "Unable to unpause job"))
			return
		}
		r.Success("Unpaused job successfully")
	})
	// }}}

	r.Dispatch("GET /v2/tenants/:uuid/tasks", func(r *route.Request) { // {{{
		limit, err := strconv.Atoi(r.Param("limit", "0"))
		if err != nil || limit <= 0 {
			r.Fail(route.Bad(err, "Invalid limit parameter given"))
			return
		}

		tasks, err := core.DB.GetAllTasks(
			&db.TaskFilter{
				SkipActive:   r.ParamIs("active", "f"),
				SkipInactive: r.ParamIs("active", "t"),
				ForStatus:    r.Param("status", ""),
				ForTenant:    r.Args[1],
				Limit:        limit,
			},
		)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve task information"))
			return
		}

		r.OK(tasks)
	})
	// }}}
	r.Dispatch("GET /v2/tenants/:uuid/tasks/:uuid", func(r *route.Request) { // {{{
		task, err := core.DB.GetTask(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve task information"))
			return
		}
		if task == nil || task.TenantUUID.String() != r.Args[1] {
			r.Fail(route.NotFound(err, "No such task"))
			return
		}

		r.OK(task)
	})
	// }}}
	r.Dispatch("DELETE /v2/tenants/:uuid/tasks/:uuid", func(r *route.Request) { // {{{
		task, err := core.DB.GetTask(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve task information"))
			return
		}
		if task == nil || task.TenantUUID.String() != r.Args[1] {
			r.Fail(route.NotFound(err, "No such task"))
			return
		}

		if err = core.DB.CancelTask(task.UUID, time.Now()); err != nil {
			r.Fail(route.Oops(err, "Unable to cancel task"))
			return
		}

		r.Success("Canceled task successfully")
	})
	// }}}

	r.Dispatch("GET /v2/tenants/:uuid/archives", func(r *route.Request) { // {{{
		status := []string{}
		if s := r.Param("status", ""); s != "" {
			status = append(status, s)
		}

		limit := r.Param("limit", "")
		if invalidlimit(limit) {
			r.Fail(route.Bad(nil, "FIXME need an error message"))
			return
		}

		archives, err := core.DB.GetAllArchives(
			&db.ArchiveFilter{
				ForTenant:  r.Args[1],
				ForTarget:  r.Param("target", ""),
				ForStore:   r.Param("store", ""),
				Before:     paramDate(r.Req, "before"),
				After:      paramDate(r.Req, "after"),
				WithStatus: status,
				Limit:      limit,
			},
		)

		if err != nil {
			r.Fail(route.Oops(err, "FIXME need an error message"))
			return
		}

		r.OK(archives)
	})
	// }}}
	r.Dispatch("GET /v2/tenants/:uuid/archives/:uuid", func(r *route.Request) { // {{{
		tennant_id := uuid.Parse(r.Args[1])
		archive_id := uuid.Parse(r.Args[2])
		if archive_id == nil || tennant_id == nil {
			r.Fail(route.Bad(nil, "Invalid UUID speficied"))
			return
		}

		archive, err := core.DB.GetArchive(archive_id)
		if err != nil {
			r.Fail(route.Oops(err, "FIXME need an error message"))
			return
		}

		if archive == nil || strings.Compare(archive.TenantUUID.String(), tennant_id.String()) != 0 {
			r.Fail(route.NotFound(nil, "Archive Not Found"))
			return
		}

		r.OK(archive)
	})
	// }}}
	r.Dispatch("PUT /v2/tenants/:uuid/archives/:uuid", func(r *route.Request) { // {{{
		if r.Req.Body == nil {
			r.Fail(route.Bad(nil, "FIXME need an error message"))
			return
		}
		tennant_id := uuid.Parse(r.Args[1])
		archive_id := uuid.Parse(r.Args[2])
		if archive_id == nil || tennant_id == nil {
			r.Fail(route.Bad(nil, "Invalid UUID speficied"))
			return
		}

		archive, err := core.DB.GetArchive(archive_id)
		if err != nil {
			r.Fail(route.Oops(err, "FIXME need an error message"))
			return
		}

		if archive == nil || strings.Compare(archive.TenantUUID.String(), tennant_id.String()) != 0 {
			r.Fail(route.NotFound(nil, "Archive Not Found"))
			return
		}

		var params struct {
			Notes string `json:"notes"`
		}
		if err := json.NewDecoder(r.Req.Body).Decode(&params); err != nil && err != io.EOF {
			r.Fail(route.Oops(err, "FIXME need an error message"))
			return
		}

		if params.Notes == "" {
			r.Fail(route.Bad(nil, "No notes were supplied"))
		}
		archive.Notes = params.Notes
		if err := core.DB.UpdateArchive(archive); err != nil {
			r.Fail(route.Oops(err, "FIXME need an error message"))
			return
		}
		r.OK("updated")
	})
	// }}}
	r.Dispatch("DELETE /v2/tenants/:uuid/archives/:uuid", func(r *route.Request) { // {{{
		tennant_id := uuid.Parse(r.Args[1])
		archive_id := uuid.Parse(r.Args[2])
		if archive_id == nil || tennant_id == nil {
			r.Fail(route.Bad(nil, "Invalid UUID speficied"))
			return
		}

		archive, err := core.DB.GetArchive(archive_id)
		if err != nil {
			r.Fail(route.Oops(err, "FIXME need an error message"))
			return
		}

		if archive == nil || strings.Compare(archive.TenantUUID.String(), tennant_id.String()) != 0 {
			r.Fail(route.NotFound(nil, "Archive Not Found"))
			return
		}

		if deleted, err := core.DB.DeleteArchive(archive.UUID); err != nil {
			r.Fail(route.Oops(err, "FIXME need an error message"))
			return
		} else if !deleted {
			r.Fail(route.Bad(err, "FIXME need an error message"))
		} else {
			r.OK("deleted")
		}
	})
	// }}}
	r.Dispatch("POST /v2/tenants/:uuid/archives/:uuid/restore", func(r *route.Request) { // {{{
		if r.Req.Body == nil {
			r.Fail(route.Bad(nil, "FIXME need an error message"))
			return
		}

		tennant_id := uuid.Parse(r.Args[1])
		archive_id := uuid.Parse(r.Args[2])
		if archive_id == nil || tennant_id == nil {
			r.Fail(route.Bad(nil, "Invalid UUID speficied"))
			return
		}

		var params struct {
			Target string `json:"target"`
			Owner  string `json:"owner"`
		}
		if err := json.NewDecoder(r.Req.Body).Decode(&params); err != nil && err != io.EOF {
			r.Fail(route.Oops(err, "FIXME need an error message"))
			return
		}

		if params.Owner == "" {
			params.Owner = "anon"
		}

		// find the archive
		archive, err := core.DB.GetArchive(archive_id)
		if err != nil {
			r.Fail(route.Oops(err, "FIXME need an error message"))
			return
		}
		if archive == nil || strings.Compare(archive.TenantUUID.String(), tennant_id.String()) != 0 {
			r.Fail(route.NotFound(nil, "Archive Not Found"))
			return
		}

		var target *db.Target
		if params.Target == "" {
			target, err = core.DB.GetTarget(archive.TargetUUID)
		} else {
			target, err = core.DB.GetTarget(uuid.Parse(params.Target))
		}
		if err != nil {
			r.Fail(route.Oops(err, "FIXME need an error message"))
			return
		}

		if target == nil || strings.Compare(target.TenantUUID.String(), tennant_id.String()) != 0 {
			r.Fail(route.NotFound(nil, "Archive Not Found"))
			return
		}

		task, err := core.DB.CreateRestoreTask(params.Owner, archive, target)
		if err != nil {
			r.Fail(route.Oops(err, "FIXME need an error message"))
			return
		}
		r.OK(fmt.Sprintf(`{"ok":"scheduled","task_uuid":"%s"}`, task.UUID))
	})
	// }}}

	r.Dispatch("POST /v2/auth/login", func(r *route.Request) { // {{{
		auth := struct {
			Username string
			Password string
		}{}

		if !r.Payload(&auth) { //Payload reports its own errors
			return
		}

		if auth.Username == "" {
			r.Fail(route.Errorf(403, nil, "no username given"))
			return
		}

		if auth.Password == "" {
			r.Fail(route.Errorf(403, nil, "no password given"))
		}

		user, err := core.DB.GetUser(auth.Username, "local")
		if err != nil {
			r.Fail(route.Oops(err, "An unknown error occurred when authenticating local user '%s'", auth.Username))
			return
		}

		if user == nil || !user.Authenticate(auth.Password) {
			r.Fail(route.Errorf(403, nil, "Incorrect username or password"))
			return
		}

		session, err := core.createSession(user)
		if err != nil {
			r.Fail(route.Oops(err, "An unknown error occurred when authenticating local user '%s'", auth.Username))
			return
		}

		r.SetCookie(SessionCookie(session.UUID.String(), true))
		r.SetHeader("X-Shield-Session", session.UUID.String())

		id, _ := core.checkAuth(session.UUID.String())
		if id == nil {
			r.Fail(route.Oops(fmt.Errorf("Failed to lookup session ID after login"), "An unknown error occurred"))
		}

		r.OK(id)
	})
	// }}}
	r.Dispatch("GET /v2/auth/logout", func(r *route.Request) { // {{{
		sessionID := getSessionID(r.Req)
		if sessionID == "" {
			//I guess we're okay with not getting a session to logout?...
			r.Success("No user to logout")
		}

		id := uuid.Parse(sessionID)
		if id == nil {
			r.Fail(route.Bad(fmt.Errorf("Invalid session ID received"), "Unable to log out"))
		}
		err := core.DB.ClearSession(id)
		if err != nil {
			r.Fail(route.Oops(err, "An unknown error occurred"))
			return
		}

		// unset the session cookie
		r.SetCookie(SessionCookie("-", false))
		r.Success("Successfully logged out")
	})
	// }}}
	r.Dispatch("GET /v2/auth/id", func(r *route.Request) { // {{{
		sessionID := getSessionID(r.Req)
		if sessionID == "" {
			r.Fail(route.Bad(fmt.Errorf("Request contained invalid session ID"), "Unable to get user information"))
		}
		id, _ := core.checkAuth(sessionID)
		if id == nil {
			r.OK(struct {
				Unauthenticated bool `json:"unauthenticated"`
			}{
				Unauthenticated: true,
			})
			return
		}

		r.OK(id)
	})
	// }}}

	return r
}
