package core

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jhunt/go-log"
	"github.com/pborman/uuid"

	"github.com/starkandwayne/shield/db"
	"github.com/starkandwayne/shield/route"
	"github.com/starkandwayne/shield/timespec"
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
	UUID        uuid.UUID        `json:"uuid"`
	Type        string           `json:"type"`
	Status      string           `json:"status"`
	Owner       string           `json:"owner"`
	RequestedAt int64            `json:"requested_at"`
	StartedAt   int64            `json:"started_at"`
	OK          bool             `json:"ok"`
	Notes       string           `json:"notes"`
	Archive     *v2SystemArchive `json:"archive,omitempty"`
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
	r := &route.Router{
		Debug: core.debug,
	}

	r.Dispatch("GET /v2/info", func(r *route.Request) { // {{{
		in := struct {
			Version string `json:"version,omitempty"`
			Env     string `json:"env"`
			API     int    `json:"api"`
		}{
			Env: core.env,
			API: 2,
		}

		/* only show sensitive things like version numbers
		   to authenticated sessions. */
		if u, _ := core.AuthenticatedUser(r); u != nil {
			in.Version = Version
		}

		r.OK(in)
	})
	// }}}
	r.Dispatch("GET /v2/health", func(r *route.Request) { // {{{
		//you must be logged into shield to access shield health
		if core.IsNotAuthenticated(r) {
			return
		}
		health, err := core.checkHealth()
		if err != nil {
			r.Fail(route.Oops(err, "Unable to check SHIELD health"))
			return
		}
		r.OK(health)
	})
	// }}}

	r.Dispatch("GET /v2/tenants/:uuid/health", func(r *route.Request) { // {{{
		//you must at least be on the tenant (which would make you at least an operator)
		//to acces the health of said tenant
		if core.IsNotTenantOperator(r, r.Args[1]) {
			return
		}
		health, err := core.checkTenantHealth(r.Args[1])
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
			Search:  in.Search,
			Backend: "local",
		})
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve users from the database."))
			return
		}
		r.OK(users)
	})
	// }}}
	r.Dispatch("POST /v2/ui/check/timespec", func(r *route.Request) { // {{{
		var in struct {
			Timespec string `json:"timespec"`
		}
		if !r.Payload(&in) {
			return
		}

		spec, err := timespec.Parse(in.Timespec)
		if err != nil {
			r.Fail(route.Bad(err, fmt.Sprintf("%s", err)))
			return
		}

		r.Success("%s", spec)
	})
	// }}}

	r.Dispatch("GET /v2/auth/providers", func(r *route.Request) { // {{{
		l := make([]AuthProviderConfig, 0)

		for _, auth := range core.auth {
			cfg := auth.Configuration(false)
			l = append(l, cfg)
		}
		r.OK(l)
	})
	// }}}
	r.Dispatch("GET /v2/auth/providers/:name", func(r *route.Request) { // {{{
		if core.IsNotSystemAdmin(r) {
			return
		}

		a, ok := core.auth[r.Args[1]]
		if !ok {
			r.Fail(route.NotFound(nil, "No such authentication provider"))
			return
		}
		r.OK(a.Configuration(true))
	})
	// }}}

	r.Dispatch("GET /v2/auth/local/users", func(r *route.Request) { // {{{
		if core.IsNotSystemManager(r) {
			return
		}

		limit, err := strconv.Atoi(r.Param("limit", "0"))
		if err != nil || limit < 0 {
			r.Fail(route.Bad(err, "Invalid limit parameter given"))
			return
		}

		l, err := core.DB.GetAllUsers(&db.UserFilter{
			UUID:       r.Param("uuid", ""),
			Account:    r.Param("account", ""),
			SysRole:    r.Param("sysrole", ""),
			ExactMatch: r.ParamIs("exact", "t"),
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
		if core.IsNotSystemManager(r) {
			return
		}

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
			r.Fail(route.Oops(err, "Unable to retrieve local user information"))
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
		if core.IsNotSystemManager(r) {
			return
		}

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
				"engineer":
			default:
				r.Fail(route.Bad(nil, "System role '%s' is invalid", in.SysRole))
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
			r.Fail(route.Oops(err, "Unable to create local user '%s'", in.Account))
			return
		}

		if exists != nil {
			r.Fail(route.Bad(nil, "user '%s' already exists", u.Account))
			return
		}

		u, err = core.DB.CreateUser(u)
		if u == nil || err != nil {
			r.Fail(route.Oops(err, "Unable to create local user '%s'", in.Account))
			return
		}
		r.OK(u)
	})
	// }}}
	r.Dispatch("PATCH /v2/auth/local/users/:uuid", func(r *route.Request) { // {{{
		if core.IsNotSystemManager(r) {
			return
		}

		/* FIXME rules for updating accounts:
		   1. you can update your own account (except for sysrole)
		   2. managers can update engineers and ''
		   3. admins can update managers, engineers and ''
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
			r.Fail(route.Oops(err, "Unable to update local user '%s'", user.Account))
			return
		}
		if user == nil || user.Backend != "local" {
			r.Fail(route.NotFound(nil, "No such local user"))
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
				"engineer":
				user.SysRole = in.SysRole
			default:
				r.Fail(route.Bad(nil, "System role '%s' is invalid", in.SysRole))
				return
			}
		}

		if in.Password != "" {
			user.SetPassword(in.Password)
		}

		err = core.DB.UpdateUser(user)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to update local user '%s'", user.Account))
			return
		}

		r.Success("Updated")
	})
	// }}}
	r.Dispatch("DELETE /v2/auth/local/users/:uuid", func(r *route.Request) { // {{{
		if core.IsNotSystemManager(r) {
			return
		}

		/* FIXME rules for deleting accounts:
		   1. you cannot delete your own account
		   2. managers can delete engineers and ''
		   3. admins can delete managers, engineers and ''
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
		r.Success("Successfully deleted local user")
	})
	// }}}

	r.Dispatch("GET /v2/auth/tokens", func(r *route.Request) { // {{{
		if core.IsNotAuthenticated(r) {
			return
		}

		user, _ := core.AuthenticatedUser(r)
		tokens, err := core.DB.GetAllAuthTokens(&db.AuthTokenFilter{
			User: user,
		})
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve tokens information"))
			return
		}

		for i := range tokens {
			tokens[i].Session = nil
		}

		r.OK(tokens)
	})
	// }}}
	r.Dispatch("POST /v2/auth/tokens", func(r *route.Request) { // {{{
		if core.IsNotAuthenticated(r) {
			return
		}
		user, _ := core.AuthenticatedUser(r)

		var in struct {
			Name string `json:"name"`
		}
		if !r.Payload(&in) {
			return
		}
		if r.Missing("name", in.Name) {
			return
		}

		existing, err := core.DB.GetAllAuthTokens(&db.AuthTokenFilter{
			Name: in.Name,
			User: user,
		})
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve tokens information"))
			return
		}
		if len(existing) != 0 {
			r.Fail(route.Bad(err, "A token with this name already exists"))
			return
		}

		token, id, err := core.DB.GenerateAuthToken(in.Name, user)
		if id == "" || err != nil {
			r.Fail(route.Oops(err, "Unable to generate new token"))
			return
		}

		r.OK(token)
	})
	// }}}
	r.Dispatch("DELETE /v2/auth/tokens/:token", func(r *route.Request) { // {{{
		if core.IsNotAuthenticated(r) {
			return
		}

		user, _ := core.AuthenticatedUser(r)
		if err := core.DB.DeleteAuthToken(r.Args[1], user); err != nil {
			r.Fail(route.Oops(err, "Unable to revoke auth token"))
			return
		}

		r.Success("Token revoked")
	})
	// }}}
	r.Dispatch("GET /v2/auth/sessions", func(r *route.Request) { // {{{
		if core.IsNotSystemAdmin(r) {
			return
		}

		limit, err := strconv.Atoi(r.Param("limit", "0"))
		if err != nil || limit < 0 {
			r.Fail(route.Bad(err, "Invalid limit parameter given"))
			return
		}

		sessions, err := core.DB.GetAllSessions(
			&db.SessionFilter{
				UUID:       r.Param("uuid", ""),
				UserUUID:   r.Param("user_uuid", ""),
				Name:       r.Param("name", ""),
				IP:         r.Param("ip_addr", ""),
				ExactMatch: r.ParamIs("exact", "t"),
				IsToken:    r.ParamIs("is_token", "t"),
				Limit:      limit,
			},
		)
		for _, session := range sessions {
			if session.UUID.String() == r.SessionID() {
				session.CurrentSession = true
				break
			}
		}

		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve session information"))
			return
		}

		r.OK(sessions)
	})
	// }}}
	r.Dispatch("GET /v2/auth/sessions/:uuid", func(r *route.Request) { // {{{
		if core.IsNotSystemAdmin(r) {
			return
		}

		limit, err := strconv.Atoi(r.Param("limit", "0"))
		if err != nil || limit < 0 {
			r.Fail(route.Bad(err, "Invalid limit parameter given"))
			return
		}

		session, err := core.DB.GetSession(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve session information"))
			return
		}
		if session.UUID.String() == r.SessionID() {
			session.CurrentSession = true
		}

		r.OK(session)
	})
	// }}}
	r.Dispatch("DELETE /v2/auth/sessions/:uuid", func(r *route.Request) { // {{{
		if core.IsNotSystemAdmin(r) {
			return
		}
		session, err := core.DB.GetSession(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve session information"))
			return
		}
		if session == nil {
			r.Fail(route.NotFound(nil, "Session not found"))
			return
		}

		if err := core.DB.ClearSession(session.UUID.String()); err != nil {
			r.Fail(route.Oops(err, "Unable to clear session '%s' (%s)", r.Args[1], session.IP))
			return
		}
		r.Success("Successfully cleared session '%s' (%s)", r.Args[1], session.IP)
	})
	// }}}
	r.Dispatch("GET /v2/tenants/:uuid/systems", func(r *route.Request) { // {{{
		targets, err := core.DB.GetAllTargets(
			&db.TargetFilter{
				SkipUsed:   r.ParamIs("unused", "t"),
				SkipUnused: r.ParamIs("unused", "f"),
				SearchName: r.Param("name", ""),
				ForPlugin:  r.Param("plugin", ""),
				ExactMatch: r.ParamIs("exact", "t"),
				ForTenant:  r.Args[1],
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
	r.Dispatch("GET /v2/tenants/:uuid/systems/:uuid", func(r *route.Request) { // {{{
		target, err := core.DB.GetTarget(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve system information"))
			return
		}

		if target == nil || target.TenantUUID.String() != r.Args[1] {
			r.Fail(route.NotFound(err, "No such system"))
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
			system.Tasks[i].RequestedAt = task.RequestedAt.Time().Unix()

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
	r.Dispatch("POST /v2/tenants/:uuid/systems", func(r *route.Request) { // {{{
		/* FIXME */
		r.Fail(route.Errorf(501, nil, "%s: not implemented", r))
	})
	// }}}
	r.Dispatch("PUT /v2/tenants/:uuid/systems/:uuid", func(r *route.Request) { // {{{
		/* FIXME */
		r.Fail(route.Errorf(501, nil, "%s: not implemented", r))
	})
	// }}}
	r.Dispatch("PATCH /v2/tenants/:uuid/systems/:uuid", func(r *route.Request) { // {{{
		var in struct {
			Annotations []struct {
				Type        string `json:"type"`
				UUID        string `json:"uuid"`
				Disposition string `json:"disposition"`
				Notes       string `json:"notes"`
				Clear       string `json:"clear"`
			} `json:"annotations"`
		}
		if !r.Payload(&in) {
			return
		}

		target, err := core.DB.GetTarget(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve system information"))
			return
		}

		if target == nil || target.TenantUUID.String() != r.Args[1] {
			r.Fail(route.NotFound(nil, "No such system"))
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
	r.Dispatch("DELETE /v2/tenants/:uuid/systems/:uuid", func(r *route.Request) { // {{{
		/* FIXME */
		r.Fail(route.Errorf(501, nil, "%s: not implemented", r))
	})
	// }}}

	r.Dispatch("GET /v2/agents", func(r *route.Request) { // {{{
		if core.IsNotSystemAdmin(r) {
			return
		}

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
		if core.IsNotSystemAdmin(r) {
			return
		}

		agent, err := core.DB.GetAgent(uuid.Parse(r.Args[1]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve agent information"))
			return
		}
		if agent == nil {
			r.Fail(route.NotFound(nil, "No such agent"))
			return
		}

		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(agent.Metadata), &raw); err != nil {
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
	r.Dispatch("POST /v2/agents/:uuid/(show|hide)", func(r *route.Request) { // {{{
		if core.IsNotSystemAdmin(r) {
			return
		}

		agent, err := core.DB.GetAgent(uuid.Parse(r.Args[1]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve agent information"))
			return
		}
		if agent == nil {
			r.Fail(route.NotFound(nil, "No such agent"))
			return
		}

		agent.Hidden = (r.Args[2] == "hide")
		if err := core.DB.UpdateAgent(agent); err != nil {
			r.Fail(route.Oops(err, "Unable to set agent visibility"))
			return
		}

		if agent.Hidden {
			r.Success("Agent is now visible only to SHIELD site engineers")
		} else {
			r.Success("Agent is now visible to everyone")
		}
	})
	// }}}

	r.Dispatch("GET /v2/tenants", func(r *route.Request) { // {{{
		if core.IsNotSystemManager(r) {
			return
		}

		limit, err := strconv.Atoi(r.Param("limit", "0"))
		if err != nil || limit < 0 {
			r.Fail(route.Bad(err, "Invalid limit parameter given"))
			return
		}

		tenants, err := core.DB.GetAllTenants(&db.TenantFilter{
			UUID:       r.Param("uuid", ""),
			Name:       r.Param("name", ""),
			ExactMatch: r.ParamIs("exact", "t"),
			Limit:      limit,
		})

		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve tenants information"))
			return
		}
		r.OK(tenants)
	})
	// }}}
	r.Dispatch("GET /v2/tenants/:uuid", func(r *route.Request) { // {{{
		if !core.CanManageTenants(r, r.Args[1]) {
			return
		}

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
	r.Dispatch("POST /v2/tenants", func(r *route.Request) { // {{{
		if core.IsNotSystemManager(r) {
			return
		}

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
		if t == nil || err != nil {
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

		err = core.DB.InheritRetentionTemplates(t)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to install template retention policies into new tenant"))
			return
		}

		r.OK(t)
	})
	// }}}
	r.Dispatch("POST /v2/tenants/:uuid/invite", func(r *route.Request) { // {{{
		if !core.CanManageTenants(r, r.Args[1]) {
			return
		}

		tenant, err := core.DB.GetTenant(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to update tenant memberships information"))
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
		if !core.CanManageTenants(r, r.Args[1]) {
			return
		}

		tenant, err := core.DB.GetTenant(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to update tenant memberships information"))
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

		r.Success("Banishments served.")
	})
	// }}}
	r.Dispatch("GET /v2/tenants/:uuid", func(r *route.Request) { // {{{
		if core.IsNotSystemManager(r) {
			return
		}

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
	r.Dispatch("PATCH /v2/tenants/:uuid", func(r *route.Request) { // {{{
		if core.IsNotSystemManager(r) {
			return
		}

		var in struct {
			Name string `json:"name"`
		}
		if !r.Payload(&in) {
			return
		}

		if r.Missing("name", in.Name) {
			return
		}

		tenant, err := core.DB.GetTenant(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve tenant information"))
			return
		}
		if tenant == nil {
			r.Fail(route.Oops(err, "No such tenant"))
			return
		}

		if in.Name != "" {
			tenant.Name = in.Name
		}

		t, err := core.DB.UpdateTenant(tenant)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to update tenant '%s'", in.Name))
			return
		}
		r.OK(t)
	})
	// }}}
	r.Dispatch("DELETE /v2/tenants/:uuid", func(r *route.Request) { // {{{
		if core.IsNotSystemManager(r) {
			return
		}

		tenant, err := core.DB.GetTenant(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve tenant information"))
			return
		}

		if tenant == nil {
			r.Fail(route.NotFound(nil, "Tenant not found"))
			return
		}

		if err := core.DB.DeleteTenant(tenant); err != nil {
			r.Fail(route.Oops(err, "Unable to delete tenant '%s' (%s)", r.Args[1], tenant.Name))
			return
		}

		r.Success("Successfully deleted tenant '%s' (%s)", r.Args[1], tenant.Name)
	})
	// }}}

	r.Dispatch("GET /v2/tenants/:uuid/agents", func(r *route.Request) { // {{{
		if core.IsNotTenantOperator(r, r.Args[1]) {
			return
		}

		agents, err := core.DB.GetAllAgents(&db.AgentFilter{
			SkipHidden: true,
		})
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve agent information"))
			return
		}

		r.OK(agents)
	})
	// }}}
	r.Dispatch("GET /v2/tenants/:uuid/agents/:uuid", func(r *route.Request) { // {{{
		if core.IsNotTenantOperator(r, r.Args[1]) {
			return
		}

		agent, err := core.DB.GetAgent(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve agent information"))
			return
		}
		if agent == nil || agent.Hidden {
			r.Fail(route.NotFound(nil, "No such agent"))
			return
		}

		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(agent.Metadata), &raw); err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve agent information"))
			return
		}

		resp := struct {
			Agent    db.Agent               `json:"agent"`
			Metadata map[string]interface{} `json:"metadata"`
		}{
			Agent:    *agent,
			Metadata: raw,
		}

		r.OK(resp)
	})
	// }}}

	r.Dispatch("GET /v2/tenants/:uuid/targets", func(r *route.Request) { // {{{
		if core.IsNotTenantOperator(r, r.Args[1]) {
			return
		}

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
		if core.IsNotTenantEngineer(r, r.Args[1]) {
			return
		}

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
			Name    string `json:"name"`
			Summary string `json:"summary"`
			Plugin  string `json:"plugin"`
			Agent   string `json:"agent"`

			Config   map[string]interface{} `json:"config"`
			endpoint string
		}

		if !r.Payload(&in) {
			return
		}
		if in.Config != nil {
			b, err := json.Marshal(in.Config)
			if err != nil {
				r.Fail(route.Oops(err, "Unable to create target"))
			}
			in.endpoint = string(b)
		}
		if r.Missing("name", in.Name, "plugin", in.Plugin, "endpoint", in.endpoint, "agent", in.Agent) {
			return
		}

		if r.ParamIs("test", "t") {
			r.Success("validation suceeded (request made in ?test=t mode)")
			return
		}

		target, err := core.DB.CreateTarget(&db.Target{
			TenantUUID: uuid.Parse(r.Args[1]),
			Name:       in.Name,
			Summary:    in.Summary,
			Plugin:     in.Plugin,
			Config:     in.Config,
			Agent:      in.Agent,
		})
		if target == nil || err != nil {
			r.Fail(route.Oops(err, "Unable to create new data target"))
			return
		}

		r.OK(target)
	})
	// }}}
	r.Dispatch("GET /v2/tenants/:uuid/targets/:uuid", func(r *route.Request) { // {{{
		if core.IsNotTenantOperator(r, r.Args[1]) {
			return
		}

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
		if core.IsNotTenantEngineer(r, r.Args[1]) {
			return
		}

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

			Config map[string]interface{} `json:"config"`
		}
		if !r.Payload(&in) {
			return
		}
		if in.Endpoint == "" && in.Config != nil {
			b, err := json.Marshal(in.Config)
			if err != nil {
				r.Fail(route.Oops(err, "Unable to create target"))
			}
			in.Endpoint = string(b)
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
		if in.Config != nil {
			target.Config = in.Config
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
		if core.IsNotTenantEngineer(r, r.Args[1]) {
			return
		}

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
		if core.IsNotTenantOperator(r, r.Args[1]) {
			return
		}

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
		if core.IsNotTenantEngineer(r, r.Args[1]) {
			return
		}

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
		if in.Expires%86400 != 0 {
			r.Fail(route.Bad(nil, "Retention policy expiry must be a multiple of 1 day"))
			return
		}

		if r.ParamIs("test", "t") {
			r.Success("validation suceeded (request made in ?test=t mode)")
			return
		}

		policy, err := core.DB.CreateRetentionPolicy(&db.RetentionPolicy{
			TenantUUID: uuid.Parse(r.Args[1]),
			Name:       in.Name,
			Summary:    in.Summary,
			Expires:    in.Expires,
		})
		if policy == nil || err != nil {
			r.Fail(route.Oops(err, "Unable to create retention policy"))
			return
		}

		r.OK(policy)
	})
	// }}}
	r.Dispatch("GET /v2/tenants/:uuid/policies/:uuid", func(r *route.Request) { // {{{
		if core.IsNotTenantEngineer(r, r.Args[1]) {
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

		r.OK(policy)
	})
	// }}}
	r.Dispatch("PATCH /v2/tenants/:uuid/policies/:uuid", func(r *route.Request) { // {{{
		if core.IsNotTenantEngineer(r, r.Args[1]) {
			return
		}

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
			policy.Summary = in.Summary
		}
		if in.Expires != 0 {
			/* FIXME: for v2, flip expires over to days, not seconds */
			if in.Expires < 86400 {
				r.Fail(route.Bad(nil, "Retention policy expiry must be greater than 1 day"))
				return
			}
			if in.Expires%86400 != 0 {
				r.Fail(route.Bad(nil, "Retention policy expiry must be a multiple of 1 day"))
				return
			}
			policy.Expires = in.Expires
		}

		if err := core.DB.UpdateRetentionPolicy(policy); err != nil {
			r.Fail(route.Oops(err, "Unable to update retention policy"))
			return
		}

		r.OK(policy)
	})
	// }}}
	r.Dispatch("DELETE /v2/tenants/:uuid/policies/:uuid", func(r *route.Request) { // {{{
		if core.IsNotTenantEngineer(r, r.Args[1]) {
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
		if core.IsNotTenantOperator(r, r.Args[1]) {
			return
		}

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
			r.Fail(route.Oops(err, "Unable to retrieve storage systems information"))
			return
		}

		r.OK(stores)
	})
	// }}}
	r.Dispatch("GET /v2/tenants/:uuid/stores/:uuid", func(r *route.Request) { // {{{
		if core.IsNotTenantOperator(r, r.Args[1]) {
			return
		}

		store, err := core.DB.GetStore(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve storage system information"))
			return
		}

		if store == nil || store.TenantUUID.String() != r.Args[1] {
			r.Fail(route.NotFound(nil, "No such storage system"))
			return
		}

		/* FIXME: we also have to handle public, for operators */
		if err := store.DisplayPublic(); err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve storage systems information"))
			return
		}

		r.OK(store)
	})
	// }}}""
	r.Dispatch("POST /v2/tenants/:uuid/stores", func(r *route.Request) { // {{{
		if core.IsNotTenantEngineer(r, r.Args[1]) {
			return
		}

		var in struct {
			Name      string `json:"name"`
			Summary   string `json:"summary"`
			Agent     string `json:"agent"`
			Plugin    string `json:"plugin"`
			Threshold int64  `json:"threshold"`

			Config map[string]interface{} `json:"config"`
		}

		if !r.Payload(&in) {
			return
		}

		if r.Missing("name", in.Name, "agent", in.Agent, "plugin", in.Plugin, "threshold", fmt.Sprint(in.Threshold)) {
			return
		}

		tenant, err := core.DB.GetTenant(r.Args[1])
		if tenant == nil || err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve storage system information"))
			return
		}

		if r.ParamIs("test", "t") {
			r.Success("validation suceeded (request made in ?test=t mode)")
			return
		}

		store, err := core.DB.CreateStore(&db.Store{
			TenantUUID: tenant.UUID,
			Name:       in.Name,
			Summary:    in.Summary,
			Agent:      in.Agent,
			Plugin:     in.Plugin,
			Config:     in.Config,
			Threshold:  in.Threshold,
		})
		if store == nil || err != nil {
			r.Fail(route.Oops(err, "Unable to create new storage system"))
			return
		}

		r.OK(store)
	})
	// }}}
	r.Dispatch("PUT /v2/tenants/:uuid/stores/:uuid", func(r *route.Request) { // {{{
		if core.IsNotTenantEngineer(r, r.Args[1]) {
			return
		}

		var in struct {
			Name      string `json:"name"`
			Summary   string `json:"summary"`
			Agent     string `json:"agent"`
			Plugin    string `json:"plugin"`
			Threshold int64  `json:"threshold"`

			Config map[string]interface{} `json:"config"`
		}
		if !r.Payload(&in) {
			r.Fail(route.Bad(nil, "Unable to update storage system"))
			return
		}

		store, err := core.DB.GetStore(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve storage system information"))
			return
		}
		if store == nil || store.TenantUUID.String() != r.Args[1] {
			r.Fail(route.Oops(err, "No such storage system"))
			return
		}

		if in.Name != "" {
			store.Name = in.Name
		}
		if in.Summary != "" {
			store.Summary = in.Summary
		}
		if in.Agent != "" {
			store.Agent = in.Agent
		}
		if in.Plugin != "" {
			store.Plugin = in.Plugin
		}
		if in.Threshold != 0 {
			store.Threshold = in.Threshold
		}

		if in.Config != nil {
			store.Config = in.Config
		}

		if err := core.DB.UpdateStore(store); err != nil {
			r.Fail(route.Oops(err, "Unable to update storage system"))
			return
		}

		store, err = core.DB.GetStore(store.UUID)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve storage system information"))
			return
		}

		r.OK(store)
	})
	// }}}
	r.Dispatch("DELETE /v2/tenants/:uuid/stores/:uuid", func(r *route.Request) { // {{{
		if core.IsNotTenantEngineer(r, r.Args[1]) {
			return
		}

		store, err := core.DB.GetStore(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve storage system information"))
			return
		}
		if store == nil || store.TenantUUID.String() != r.Args[1] {
			r.Fail(route.Oops(err, "No such storage system"))
			return
		}

		deleted, err := core.DB.DeleteStore(store.UUID)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to delete storage system"))
			return
		}
		if !deleted {
			r.Fail(route.Bad(nil, "The storage system cannot be deleted at this time"))
			return
		}

		r.Success("Storage system deleted successfully")
	})
	// }}}

	r.Dispatch("GET /v2/tenants/:uuid/jobs", func(r *route.Request) { // {{{
		if core.IsNotTenantOperator(r, r.Args[1]) {
			return
		}

		jobs, err := core.DB.GetAllJobs(
			&db.JobFilter{
				ForTenant:    r.Args[1],
				SkipPaused:   r.ParamIs("paused", "f"),
				SkipUnpaused: r.ParamIs("paused", "t"),

				SearchName: r.Param("name", ""),

				ForTarget:  r.Param("target", ""),
				ForStore:   r.Param("store", ""),
				ForPolicy:  r.Param("policy", ""),
				ExactMatch: r.ParamIs("exact", "t"),
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
		if core.IsNotTenantEngineer(r, r.Args[1]) {
			return
		}

		var in struct {
			Name     string `json:"name"`
			Summary  string `json:"summary"`
			Schedule string `json:"schedule"`
			Paused   bool   `json:"paused"`
			Store    string `json:"store"`
			Target   string `json:"target"`
			Policy   string `json:"policy"`
		}
		if !r.Payload(&in) {
			return
		}

		if r.Missing("name", in.Name, "store", in.Store, "target", in.Target, "schedule", in.Schedule, "policy", in.Policy) {
			return
		}

		job, err := core.DB.CreateJob(&db.Job{
			TenantUUID: uuid.Parse(r.Args[1]),
			Name:       in.Name,
			Summary:    in.Summary,
			Schedule:   in.Schedule,
			Paused:     in.Paused,
			StoreUUID:  uuid.Parse(in.Store),
			TargetUUID: uuid.Parse(in.Target),
			PolicyUUID: uuid.Parse(in.Policy),
		})
		if job == nil || err != nil {
			r.Fail(route.Oops(err, "Unable to create new job"))
			return
		}

		r.OK(job)
	})
	// }}}
	r.Dispatch("GET /v2/tenants/:uuid/jobs/:uuid", func(r *route.Request) { // {{{
		if core.IsNotTenantOperator(r, r.Args[1]) {
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

		r.OK(job)
	})
	// }}}
	r.Dispatch("PUT /v2/tenants/:uuid/jobs/:uuid", func(r *route.Request) { // {{{
		if core.IsNotTenantEngineer(r, r.Args[1]) {
			return
		}

		var in struct {
			Name     string `json:"name"`
			Summary  string `json:"summary"`
			Schedule string `json:"schedule"`

			StoreUUID  string `json:"store"`
			TargetUUID string `json:"target"`
			PolicyUUID string `json:"policy"`
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
		job.TargetUUID = job.Target.UUID
		if in.TargetUUID != "" {
			job.TargetUUID = uuid.Parse(in.TargetUUID)
		}
		job.StoreUUID = job.Store.UUID
		if in.StoreUUID != "" {
			job.StoreUUID = uuid.Parse(in.StoreUUID)
		}
		job.PolicyUUID = job.Policy.UUID
		if in.PolicyUUID != "" {
			job.PolicyUUID = uuid.Parse(in.PolicyUUID)
		}

		if err := core.DB.UpdateJob(job); err != nil {
			r.Fail(route.Oops(err, "Unable to update job"))
			return
		}

		r.Success("Updated job successfully")
	})
	// }}}
	r.Dispatch("DELETE /v2/tenants/:uuid/jobs/:uuid", func(r *route.Request) { // {{{
		if core.IsNotTenantEngineer(r, r.Args[1]) {
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
		if core.IsNotTenantOperator(r, r.Args[1]) {
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

		user, _ := core.AuthenticatedUser(r)
		task, err := core.DB.CreateBackupTask(fmt.Sprintf("%s@%s", user.Account, user.Backend), job)
		if task == nil || err != nil {
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
		if core.IsNotTenantOperator(r, r.Args[1]) {
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

		if _, err = core.DB.PauseJob(job.UUID); err != nil {
			r.Fail(route.Oops(err, "Unable to pause job"))
			return
		}
		r.Success("Paused job successfully")
	})
	// }}}
	r.Dispatch("POST /v2/tenants/:uuid/jobs/:uuid/unpause", func(r *route.Request) { // {{{
		if core.IsNotTenantOperator(r, r.Args[1]) {
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

		if _, err = core.DB.UnpauseJob(job.UUID); err != nil {
			r.Fail(route.Oops(err, "Unable to unpause job"))
			return
		}
		r.Success("Unpaused job successfully")
	})
	// }}}

	r.Dispatch("GET /v2/tenants/:uuid/tasks", func(r *route.Request) { // {{{
		if core.IsNotTenantOperator(r, r.Args[1]) {
			return
		}

		limit, err := strconv.Atoi(r.Param("limit", "0"))
		if err != nil || limit < 0 {
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
		if core.IsNotTenantOperator(r, r.Args[1]) {
			return
		}

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
		if core.IsNotTenantOperator(r, r.Args[1]) {
			return
		}

		task, err := core.DB.GetTask(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve task information"))
			return
		}
		if task == nil || task.TenantUUID.String() != r.Args[1] {
			r.Fail(route.NotFound(err, "No such task"))
			return
		}

		if err := core.DB.CancelTask(task.UUID, time.Now()); err != nil {
			r.Fail(route.Oops(err, "Unable to cancel task"))
			return
		}

		r.Success("Canceled task successfully")
	})
	// }}}

	r.Dispatch("GET /v2/tenants/:uuid/archives", func(r *route.Request) { // {{{
		if core.IsNotTenantOperator(r, r.Args[1]) {
			return
		}

		limit, err := strconv.Atoi(r.Param("limit", "0"))
		if err != nil || limit < 0 {
			r.Fail(route.Bad(err, "Invalid limit parameter given"))
			return
		}

		status := []string{}
		if s := r.Param("status", ""); s != "" {
			status = append(status, s)
		}

		archives, err := core.DB.GetAllArchives(
			&db.ArchiveFilter{
				ForTenant:  r.Args[1],
				ForTarget:  r.Param("target", ""),
				ForStore:   r.Param("store", ""),
				Before:     r.ParamDate("before"),
				After:      r.ParamDate("after"),
				WithStatus: status,
				Limit:      limit,
			},
		)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve backup archives information"))
			return
		}

		r.OK(archives)
	})
	// }}}
	r.Dispatch("GET /v2/tenants/:uuid/archives/:uuid", func(r *route.Request) { // {{{
		if core.IsNotTenantOperator(r, r.Args[1]) {
			return
		}

		archive, err := core.DB.GetArchive(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve backup archive information"))
			return
		}

		if archive == nil || archive.TenantUUID.String() != r.Args[1] {
			r.Fail(route.NotFound(nil, "Archive Not Found"))
			return
		}

		r.OK(archive)
	})
	// }}}
	r.Dispatch("PUT /v2/tenants/:uuid/archives/:uuid", func(r *route.Request) { // {{{
		if core.IsNotTenantOperator(r, r.Args[1]) {
			return
		}

		var in struct {
			Notes string `json:"notes"`
		}
		if !r.Payload(&in) {
			return
		}

		archive, err := core.DB.GetArchive(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve backup archive information"))
			return
		}

		if archive == nil || archive.TenantUUID.String() != r.Args[1] {
			r.Fail(route.NotFound(nil, "No such backup archive"))
			return
		}

		if r.Missing("notes", in.Notes) {
			return
		}

		archive.Notes = in.Notes
		if err := core.DB.UpdateArchive(archive); err != nil {
			r.Fail(route.Oops(err, "Unable to update backup archive"))
			return
		}

		r.OK(archive)
	})
	// }}}
	r.Dispatch("DELETE /v2/tenants/:uuid/archives/:uuid", func(r *route.Request) { // {{{
		if core.IsNotTenantOperator(r, r.Args[1]) {
			return
		}

		archive, err := core.DB.GetArchive(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve backup archive information"))
			return
		}

		if archive == nil || archive.TenantUUID.String() != r.Args[1] {
			r.Fail(route.NotFound(nil, "No such backup archive"))
			return
		}

		if archive.Status != "valid" {
			r.Fail(route.Bad(err, "The backup archive could not be deleted at this time. Archive is already %s", archive.Status))
		}

		err = core.DB.ExpireArchive(archive.UUID)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to delete backup archive"))
			return
		}

		r.OK("Archive deleted successfully")
	})
	// }}}
	r.Dispatch("POST /v2/tenants/:uuid/archives/:uuid/restore", func(r *route.Request) { // {{{
		if core.IsNotTenantOperator(r, r.Args[1]) {
			return
		}

		var in struct {
			Target string `json:"target"`
		}
		if !r.Payload(&in) {
			return
		}

		archive, err := core.DB.GetArchive(uuid.Parse(r.Args[2]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve backup archive information"))
			return
		}
		if archive == nil || archive.TenantUUID.String() != r.Args[1] {
			r.Fail(route.NotFound(nil, "No such backup archive"))
			return
		}

		if in.Target == "" {
			in.Target = archive.TargetUUID.String()
		}

		target, err := core.DB.GetTarget(uuid.Parse(in.Target))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve backup archive information"))
			return
		}

		if target == nil || archive.TenantUUID.String() != r.Args[1] {
			r.Fail(route.NotFound(nil, "No such backup archive"))
			return
		}

		user, _ := core.AuthenticatedUser(r)
		task, err := core.DB.CreateRestoreTask(fmt.Sprintf("%s@%s", user.Account, user.Backend), archive, target)
		if task == nil || err != nil {
			r.Fail(route.Oops(err, "Unable to schedule a restore task"))
			return
		}

		r.OK(task)
	})
	// }}}

	r.Dispatch("POST /v2/auth/login", func(r *route.Request) { // {{{
		var in struct {
			Username string
			Password string
		}
		if !r.Payload(&in) {
			return
		}

		if r.Missing("username", in.Username, "password", in.Password) {
			return
		}

		user, err := core.DB.GetUser(in.Username, "local")
		if err != nil {
			r.Fail(route.Oops(err, "Unable to log you in"))
			return
		}

		if user == nil || !user.Authenticate(in.Password) {
			r.Fail(route.Errorf(401, nil, "Incorrect username or password"))
			return
		}

		session, err := core.createSession(&db.Session{
			UserUUID:  user.UUID,
			IP:        r.Req.RemoteAddr,
			UserAgent: r.Req.UserAgent(),
		})
		if err != nil {
			r.Fail(route.Oops(err, "Unable to log you in"))
			return
		}

		id, err := core.checkAuth(user)
		if err != nil || id == nil {
			r.Fail(route.Oops(err, "Unable to log you in"))
		}

		SetAuthHeaders(r, session.UUID)
		r.OK(id)
	})
	// }}}
	r.Dispatch("GET /v2/auth/logout", func(r *route.Request) { // {{{
		if err := core.DB.ClearSession(r.SessionID()); err != nil {
			r.Fail(route.Oops(err, "Unable to log you out"))
			return
		}

		r.SetCookie(SessionCookie("-", false))
		r.Success("Successfully logged out")
	})
	// }}}
	r.Dispatch("GET /v2/auth/id", func(r *route.Request) { // {{{
		user, _ := core.AuthenticatedUser(r)
		if id, _ := core.checkAuth(user); id != nil {
			r.OK(id)
			return
		}

		r.OK(struct {
			Unauthenticated bool `json:"unauthenticated"`
		}{true})
	})
	// }}}
	r.Dispatch("POST /v2/auth/passwd", func(r *route.Request) { // {{{
		if core.IsNotAuthenticated(r) {
			return
		}

		var in struct {
			OldPassword string `json:"old_password"`
			NewPassword string `json:"new_password"`
		}

		if !r.Payload(&in) {
			return
		}

		user, _ := core.AuthenticatedUser(r)
		if !user.Authenticate(in.OldPassword) {
			r.Fail(route.Forbidden(nil, "Incorrect password"))
			return
		}

		user.SetPassword(in.NewPassword)
		if err := core.DB.UpdateUser(user); err != nil {
			r.Fail(route.Oops(err, "Unable to change your password"))
			return
		}

		r.Success("Password changed successfully")
	})
	// }}}
	r.Dispatch("PATCH /v2/auth/user/settings", func(r *route.Request) { // {{{
		var in struct {
			DefaultTenant string `json:"default_tenant"`
		}

		if !r.Payload(&in) {
			return
		}

		user, err := core.AuthenticatedUser(r)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to save settings"))
			return
		}

		if in.DefaultTenant != "" {
			user.DefaultTenant = in.DefaultTenant
		}
		if err := core.DB.UpdateUserSettings(user); err != nil {
			r.Fail(route.Oops(err, "Unable to save settings"))
			return
		}

		r.Success("Settings saved")
	})
	// }}}

	r.Dispatch("GET /v2/global/stores", func(r *route.Request) { // {{{
		if core.IsNotAuthenticated(r) {
			return
		}

		stores, err := core.DB.GetAllStores(
			&db.StoreFilter{
				SkipUsed:   r.ParamIs("unused", "t"),
				SkipUnused: r.ParamIs("unused", "f"),
				SearchName: r.Param("name", ""),
				ForPlugin:  r.Param("plugin", ""),
				ExactMatch: r.ParamIs("exact", "t"),
				ForTenant:  uuid.NIL.String(),
			},
		)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve storage systems information"))
			return
		}

		r.OK(stores)
	})
	// }}}
	r.Dispatch("GET /v2/global/stores/:uuid", func(r *route.Request) { // {{{
		if core.IsNotAuthenticated(r) {
			return
		}

		store, err := core.DB.GetStore(uuid.Parse(r.Args[1]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve storage system information"))
			return
		}

		if store == nil || !uuid.Equal(store.TenantUUID, uuid.NIL) {
			r.Fail(route.NotFound(nil, "No such storage system"))
			return
		}

		/* FIXME: we also have to handle public, for operators */
		if err := store.DisplayPublic(); err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve storage systems information"))
			return
		}

		r.OK(store)
	})
	// }}}""
	r.Dispatch("POST /v2/global/stores", func(r *route.Request) { // {{{
		if core.IsNotSystemEngineer(r) {
			return
		}

		var in struct {
			Name      string `json:"name"`
			Summary   string `json:"summary"`
			Agent     string `json:"agent"`
			Plugin    string `json:"plugin"`
			Threshold int64  `json:"threshold"`

			Config map[string]interface{} `json:"config"`
		}

		if !r.Payload(&in) {
			return
		}

		if r.Missing("name", in.Name, "agent", in.Agent, "plugin", in.Plugin, "threshold", fmt.Sprint(in.Threshold)) {
			return
		}

		store, err := core.DB.CreateStore(&db.Store{
			TenantUUID: uuid.NIL,
			Name:       in.Name,
			Summary:    in.Summary,
			Agent:      in.Agent,
			Plugin:     in.Plugin,
			Config:     in.Config,
			Threshold:  in.Threshold,
		})
		if store == nil || err != nil {
			r.Fail(route.Oops(err, "Unable to create new storage system"))
			return
		}

		r.OK(store)
	})
	// }}}
	r.Dispatch("PUT /v2/global/stores/:uuid", func(r *route.Request) { // {{{
		if core.IsNotSystemEngineer(r) {
			return
		}

		var in struct {
			Name      string `json:"name"`
			Summary   string `json:"summary"`
			Agent     string `json:"agent"`
			Plugin    string `json:"plugin"`
			Threshold int64  `json:"threshold"`

			Config map[string]interface{} `json:"config"`
		}
		if !r.Payload(&in) {
			r.Fail(route.Bad(nil, "Unable to update storage system"))
			return
		}

		store, err := core.DB.GetStore(uuid.Parse(r.Args[1]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve storage system information"))
			return
		}
		if store == nil || !uuid.Equal(store.TenantUUID, uuid.NIL) {
			r.Fail(route.Oops(err, "No such storage system"))
			return
		}

		if in.Name != "" {
			store.Name = in.Name
		}
		if in.Summary != "" {
			store.Summary = in.Summary
		}
		if in.Agent != "" {
			store.Agent = in.Agent
		}
		if in.Plugin != "" {
			store.Plugin = in.Plugin
		}
		if in.Threshold != 0 {
			store.Threshold = in.Threshold
		}
		if in.Config != nil {
			store.Config = in.Config
		}

		if err := core.DB.UpdateStore(store); err != nil {
			r.Fail(route.Oops(err, "Unable to update storage system"))
			return
		}

		store, err = core.DB.GetStore(store.UUID)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve storage system information"))
			return
		}

		r.OK(store)
	})
	// }}}
	r.Dispatch("DELETE /v2/global/stores/:uuid", func(r *route.Request) { // {{{
		if core.IsNotSystemEngineer(r) {
			return
		}

		store, err := core.DB.GetStore(uuid.Parse(r.Args[1]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve storage system information"))
			return
		}
		if store == nil || !uuid.Equal(store.TenantUUID, uuid.NIL) {
			r.Fail(route.Oops(err, "No such storage system"))
			return
		}

		deleted, err := core.DB.DeleteStore(store.UUID)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to delete storage system"))
			return
		}
		if !deleted {
			r.Fail(route.Bad(nil, "The storage system cannot be deleted at this time"))
			return
		}

		r.Success("Storage system deleted successfully")
	})
	// }}}

	r.Dispatch("GET /v2/global/policies", func(r *route.Request) { // {{{
		if core.IsNotAuthenticated(r) {
			return
		}

		policies, err := core.DB.GetAllRetentionPolicies(
			&db.RetentionFilter{
				ForTenant:  uuid.NIL.String(),
				SkipUsed:   r.ParamIs("unused", "t"),
				SkipUnused: r.ParamIs("unused", "f"),
				SearchName: r.Param("name", ""),
				ExactMatch: r.ParamIs("exact", "t"),
			},
		)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve retention policy templates information"))
			return
		}

		r.OK(policies)
	})
	// }}}
	r.Dispatch("POST /v2/global/policies", func(r *route.Request) { // {{{
		if core.IsNotSystemEngineer(r) {
			return
		}

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
		if in.Expires%86400 != 0 {
			r.Fail(route.Bad(nil, "Retention policy expiry must be a multiple of 1 day"))
			return
		}

		policy, err := core.DB.CreateRetentionPolicy(&db.RetentionPolicy{
			TenantUUID: uuid.Parse(uuid.NIL.String()),
			Name:       in.Name,
			Summary:    in.Summary,
			Expires:    in.Expires,
		})
		if policy == nil || err != nil {
			r.Fail(route.Oops(err, "Unable to create retention policy template"))
			return
		}

		r.OK(policy)
	})
	// }}}
	r.Dispatch("GET /v2/global/policies/:uuid", func(r *route.Request) { // {{{
		if core.IsNotAuthenticated(r) {
			return
		}

		policy, err := core.DB.GetRetentionPolicy(uuid.Parse(r.Args[1]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve retention policy template information"))
			return
		}

		if policy == nil || policy.TenantUUID.String() != uuid.NIL.String() {
			r.Fail(route.NotFound(nil, "No such retention policy template"))
			return
		}

		r.OK(policy)
	})
	// }}}
	r.Dispatch("PUT /v2/global/policies/:uuid", func(r *route.Request) { // {{{
		if core.IsNotSystemEngineer(r) {
			return
		}

		var in struct {
			Name    string `json:"name"`
			Summary string `json:"summary"`
			Expires uint   `json:"expires"`
		}
		if !r.Payload(&in) {
			return
		}

		policy, err := core.DB.GetRetentionPolicy(uuid.Parse(r.Args[1]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve retention policy template information"))
			return
		}

		if policy == nil || policy.TenantUUID.String() != uuid.NIL.String() {
			r.Fail(route.NotFound(nil, "No such retention policy template"))
			return
		}

		if in.Name != "" {
			policy.Name = in.Name
		}
		if in.Summary != "" {
			policy.Summary = in.Name
		}
		if in.Expires != 0 {
			/* FIXME: for v2, flip expires over to days, not seconds */
			if in.Expires < 86400 {
				r.Fail(route.Bad(nil, "Retention policy expiry must be greater than 1 day"))
				return
			}
			if in.Expires%86400 != 0 {
				r.Fail(route.Bad(nil, "Retention policy expiry must be a multiple of 1 day"))
				return
			}
			policy.Expires = in.Expires
		}

		if err := core.DB.UpdateRetentionPolicy(policy); err != nil {
			r.Fail(route.Oops(err, "Unable to update retention policy template"))
			return
		}

		r.OK(policy)
	})
	// }}}
	r.Dispatch("DELETE /v2/global/policies/:uuid", func(r *route.Request) { // {{{
		if core.IsNotSystemEngineer(r) {
			return
		}

		policy, err := core.DB.GetRetentionPolicy(uuid.Parse(r.Args[1]))
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve retention policy template information"))
			return
		}

		if policy == nil || policy.TenantUUID.String() != uuid.NIL.String() {
			r.Fail(route.NotFound(nil, "No such retention policy template"))
			return
		}

		deleted, err := core.DB.DeleteRetentionPolicy(policy.UUID)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to delete retention policy template"))
			return
		}
		if !deleted {
			r.Fail(route.Forbidden(nil, "The retention policy template cannot be deleted at this time"))
			return
		}

		r.Success("Retention policy deleted successfully")
	})
	// }}}

	if core.debug {
		core.dispatchDebug(r)
	}
	return r
}

func (core *Core) v2copyTarget(dst *v2System, target *db.Target) error {
	dst.UUID = target.UUID
	dst.Name = target.Name
	dst.Notes = target.Summary
	dst.OK = true /* FIXME */

	jobs, err := core.DB.GetAllJobs(
		&db.JobFilter{
			ForTarget: target.UUID.String(),
		},
	)
	if err != nil {
		return err
	}

	dst.Jobs = make([]v2SystemJob, len(jobs))
	for j, job := range jobs {
		dst.Jobs[j].UUID = job.UUID
		dst.Jobs[j].Schedule = job.Schedule
		dst.Jobs[j].From = job.Target.Plugin
		dst.Jobs[j].To = job.Store.Plugin
		dst.Jobs[j].OK = job.Healthy()
		dst.Jobs[j].Store.UUID = job.Store.UUID
		dst.Jobs[j].Store.Name = job.Store.Name
		dst.Jobs[j].Store.Summary = job.Store.Summary
		dst.Jobs[j].Retention.UUID = job.Policy.UUID
		dst.Jobs[j].Retention.Name = job.Policy.Name
		dst.Jobs[j].Retention.Summary = job.Policy.Summary

		if !job.Healthy() {
			dst.OK = false
		}

		dst.Jobs[j].Keep.Days = job.Expiry / 86400
		dst.Jobs[j].Retention.Days = dst.Jobs[j].Keep.Days

		tspec, err := timespec.Parse(job.Schedule)
		if err != nil {
			return err
		}
		switch tspec.Interval {
		case timespec.Daily:
			dst.Jobs[j].Keep.N = dst.Jobs[j].Keep.Days
		case timespec.Weekly:
			dst.Jobs[j].Keep.N = dst.Jobs[j].Keep.Days / 7
		case timespec.Monthly:
			dst.Jobs[j].Keep.N = dst.Jobs[j].Keep.Days / 30
		}
	}
	return nil
}
