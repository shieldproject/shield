package core

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/jhunt/go-log"

	"github.com/shieldproject/shield/db"
	"github.com/shieldproject/shield/route"
	"github.com/shieldproject/shield/timespec"
	"github.com/shieldproject/shield/util"
)

type v2SystemArchive struct {
	UUID     string `json:"uuid"`
	Schedule string `json:"schedule"`
	TakenAt  int64  `json:"taken_at"`
	Expiry   int    `json:"expiry"`
	Size     int64  `json:"size"`
	OK       bool   `json:"ok"`
	Notes    string `json:"notes"`
}
type v2SystemTask struct {
	UUID        string           `json:"uuid"`
	Type        string           `json:"type"`
	Status      string           `json:"status"`
	Owner       string           `json:"owner"`
	RequestedAt int64            `json:"requested_at"`
	StartedAt   int64            `json:"started_at"`
	StoppedAt   int64            `json:"stopped_at"`
	OK          bool             `json:"ok"`
	Notes       string           `json:"notes"`
	Archive     *v2SystemArchive `json:"archive,omitempty"`
	Log         string           `json:"log"`

	JobUUID     string `json:"job_uuid"`
	ArchiveUUID string `json:"archive_uuid"`
	Bucket      string `json:"bucket"`
	TargetUUID  string `json:"target_uuid"`
}
type v2SystemJob struct {
	UUID     string `json:"uuid"`
	Schedule string `json:"schedule"`
	From     string `json:"from"`
	To       string `json:"to"`
	OK       bool   `json:"ok"`
	Bucket   string `json:"bucket"`

	Keep struct {
		N    int `json:"n"`
		Days int `json:"days"`
	} `json:"keep"`
}
type v2System struct {
	UUID  string `json:"uuid"`
	Name  string `json:"name"`
	Notes string `json:"notes"`
	OK    bool   `json:"ok"`

	Jobs  []v2SystemJob  `json:"jobs"`
	Tasks []v2SystemTask `json:"tasks"`
}

type v2LocalUser struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Account string `json:"account"`
	SysRole string `json:"sysrole"`
}

func (c *Core) v2API() *route.Router {
	r := &route.Router{
		Debug: c.Config.Debug,
	}

	r.Dispatch("GET /v2/info", func(r *route.Request) { // {{{
		r.OK(c.info)
	})
	// }}}
	r.Dispatch("GET /v2/bearings", func(r *route.Request) { // {{{
		var out struct {
			/* Information about this SHIELD installation itself,
			   including its name, the MOTD, the UI theme color,
			   API and software versions, etc. */
			SHIELD interface{} `json:"shield"`

			/* The currently logged-in user. */
			User *db.User `json:"user"`

			/* Storage buckets */
			Buckets []bucket `json:"buckets"`

			Archives []*db.Archive `json:"archives"`
			Jobs     []*db.Job     `json:"jobs"`
			Targets  []*db.Target  `json:"targets"`
			Agents   []*db.Agent   `json:"agents"`
		}
		out.SHIELD = c.info

		if user, err := c.db.GetUserForSession(r.SessionID()); err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve user information"))
			return

		} else if user != nil {
			out.User = user
			out.Buckets = c.buckets

			var err error
			out.Archives, err = c.db.GetAllArchives(nil)
			if err != nil {
				r.Fail(route.Oops(err, "unable to retrieve archives"))
				return
			}

			/* assemble jobs */
			out.Jobs, err = c.db.GetAllJobs(nil)
			if err != nil {
				r.Fail(route.Oops(err, "unable to retrieve jobs"))
				return
			}

			/* assemble targets */
			out.Targets, err = c.db.GetAllTargets(nil)
			if err != nil {
				r.Fail(route.Oops(err, "unable to retrieve targets"))
				return
			}

			/* assemble agents and plugins */
			out.Agents, err = c.db.GetAllAgents(&db.AgentFilter{SkipHidden: true, InflateMetadata: true})
			if err != nil {
				r.Fail(route.Oops(err, "unable to retrieve agents"))
				return
			}
		}

		r.OK(out)
	})
	// }}}
	r.Dispatch("GET /v2/buckets", func(r *route.Request) { // {{{
		if c.IsNotAuthenticated(r) {
			return
		}
		r.OK(c.buckets)
	})
	// }}}
	r.Dispatch("GET /v2/health", func(r *route.Request) { // {{{
		//you must be logged into shield to access shield health
		if c.IsNotAuthenticated(r) {
			return
		}
		health, err := c.checkHealth()
		if err != nil {
			r.Fail(route.Oops(err, "Unable to check SHIELD health"))
			return
		}
		r.OK(health)
	})
	// }}}

	r.Dispatch("GET /v2/scheduler/status", func(r *route.Request) { // {{{
		if c.IsNotSystemEngineer(r) {
			return
		}

		type named struct {
			UUID string `json:"uuid"`
			Name string `json:"name"`
		}

		type job struct {
			UUID     string `json:"uuid"`
			Name     string `json:"name"`
			Schedule string `json:"schedule"`
		}

		type archive struct {
			UUID string `json:"uuid"`
			Size int64  `json:"size"`
		}

		type backlogStatus struct {
			Priority int    `json:"priority"`
			Position int    `json:"position"`
			TaskUUID string `json:"task_uuid"`

			Op    string `json:"op"`
			Agent string `json:"agent"`

			System  *named   `json:"system,omitempty"`
			Job     *job     `json:"job,omitempty"`
			Archive *archive `json:"archive,omitempty"`
		}

		type workerStatus struct {
			ID       int    `json:"id"`
			Idle     bool   `json:"idle"`
			TaskUUID string `json:"task_uuid"`
			LastSeen int    `json:"last_seen"`

			Op     string `json:"op"`
			Status string `json:"status"`
			Agent  string `json:"agent"`

			System  *named   `json:"system,omitempty"`
			Job     *job     `json:"job,omitempty"`
			Archive *archive `json:"archive,omitempty"`
		}

		type status struct {
			Backlog []backlogStatus `json:"backlog"`
			Workers []workerStatus  `json:"workers"`
		}

		ps := c.scheduler.Status()
		out := status{
			Backlog: make([]backlogStatus, len(ps.Backlog)),
			Workers: make([]workerStatus, len(ps.Workers)),
		}

		systems := make(map[string]*db.Target)
		jobs := make(map[string]*db.Job)
		archives := make(map[string]*db.Archive)

		for i, x := range ps.Backlog {
			out.Backlog[i].Priority = x.Priority + 1
			out.Backlog[i].Position = x.Position
			out.Backlog[i].TaskUUID = x.TaskUUID

			if task, err := c.db.GetTask(x.TaskUUID); err == nil && task != nil {
				out.Backlog[i].Op = task.Op
				out.Backlog[i].Agent = task.Agent

				if task.JobUUID != "" {
					j, found := jobs[task.JobUUID]
					if !found {
						j, err = c.db.GetJob(task.JobUUID)
						if j != nil && err == nil {
							jobs[j.UUID] = j
							found = true
						}
					}
					if found {
						out.Backlog[i].Job = &job{
							UUID: j.UUID,
							Name: j.Name,
						}
					}
				}

				if task.TargetUUID != "" {
					t, found := systems[task.TargetUUID]
					if !found {
						t, err = c.db.GetTarget(task.TargetUUID)
						if t != nil && err == nil {
							systems[t.UUID] = t
							found = true
						}
					}
					if found {
						out.Backlog[i].System = &named{
							UUID: t.UUID,
							Name: t.Name,
						}
					}
				}

				if task.ArchiveUUID != "" {
					a, found := archives[task.ArchiveUUID]
					if !found {
						a, err = c.db.GetArchive(task.ArchiveUUID)
						if a != nil && err == nil {
							archives[a.UUID] = a
							found = true
						}
					}
					if found {
						out.Backlog[i].Archive = &archive{
							UUID: a.UUID,
							Size: a.Size,
						}
					}
				}
			}
		}

		for i, x := range ps.Workers {
			out.Workers[i].ID = x.ID
			out.Workers[i].Idle = x.Idle
			out.Workers[i].TaskUUID = x.TaskUUID
			out.Workers[i].LastSeen = x.LastSeen

			if x.TaskUUID == "" {
				continue
			}

			if task, err := c.db.GetTask(x.TaskUUID); err == nil && task != nil {
				out.Workers[i].Op = task.Op
				out.Workers[i].Status = task.Status
				out.Workers[i].Agent = task.Agent

				if task.JobUUID != "" {
					j, found := jobs[task.JobUUID]
					if !found {
						j, err = c.db.GetJob(task.JobUUID)
						if j != nil && err == nil {
							jobs[j.UUID] = j
							found = true
						}
					}
					if found {
						out.Workers[i].Job = &job{
							UUID: j.UUID,
							Name: j.Name,
						}
					}
				}

				if task.TargetUUID != "" {
					t, found := systems[task.TargetUUID]
					if !found {
						t, err = c.db.GetTarget(task.TargetUUID)
						if t != nil && err == nil {
							systems[t.UUID] = t
							found = true
						}
					}
					if found {
						out.Workers[i].System = &named{
							UUID: t.UUID,
							Name: t.Name,
						}
					}
				}

				if task.ArchiveUUID != "" {
					a, found := archives[task.ArchiveUUID]
					if !found {
						a, err = c.db.GetArchive(task.ArchiveUUID)
						if a != nil && err == nil {
							archives[a.UUID] = a
							found = true
						}
					}
					if found {
						out.Workers[i].Archive = &archive{
							UUID: a.UUID,
							Size: a.Size,
						}
					}
				}
			}
		}

		r.OK(out)
	})
	// }}}
	r.Dispatch("GET /v2/mbus/status", func(r *route.Request) { // {{{
		if c.IsNotSystemEngineer(r) {
			return
		}

		r.OK(c.bus.DumpState())
	})
	// }}}

	r.Dispatch("GET /v2/events", func(r *route.Request) { // {{{
		//you must be logged into shield to access the event stream
		if c.IsNotAuthenticated(r) {
			return
		}

		user, err := c.AuthenticatedUser(r)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to configure your SHIELD events stream"))
			return
		}

		queues := []string{
			"user:" + user.UUID,
		}

		if user.SysRole != "" {
			queues = append(queues, "admins")
		}

		socket := r.Upgrade(route.WebSocketSettings{
			WriteTimeout: time.Duration(c.Config.API.Websocket.WriteTimeout) * time.Second,
		})
		if socket == nil {
			return
		}

		log.Infof("registering message bus web client")
		ch, slot, err := c.bus.Register(queues)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to begin streaming SHIELD events"))
			return
		}
		log.Infof("registered with message bus as [id:%d]", slot)

		closeMeSoftly := func() { c.bus.Unregister(slot) }
		go socket.Discard(closeMeSoftly)

		pingInterval := time.Duration(c.Config.API.Websocket.PingInterval) * time.Second
		pingTimer := time.NewTimer(pingInterval)
	writeLoop:
		for {
			select {
			case event := <-ch:
				b, err := json.Marshal(event)
				if err != nil {
					log.Errorf("message bus web client [id:%d] failed to marshal JSON for websocket relay: %s", slot, err)
				} else {
					if done, err := socket.Write(b); done {
						log.Infof("message bus web client [id:%d] closed their end of the socket", slot)
						log.Infof("message bus web client [id:%d] shutting down", slot)
						closeMeSoftly()
						break writeLoop
					} else if err != nil {
						log.Errorf("message bus web client [id:%d] failed to write message to remote end: %s", slot, err)
						log.Errorf("message bus web client [id:%d] shutting down", slot)
						closeMeSoftly()
						err := socket.SendClose()
						if err != nil {
							log.Warnf("message bus web client [id:%d] failed to write close message")
						}
						break writeLoop
					}
				}

				if !pingTimer.Stop() {
					<-pingTimer.C
				}
			case <-pingTimer.C:
				if err := socket.Ping(); err != nil {
					log.Infof("message bus web client [id:%d] failed to write ping")
					closeMeSoftly()
					break writeLoop
				}
			}
			pingTimer.Reset(pingInterval)
		}

		pingTimer.Stop()
		log.Infof("message bus web client [id:%d] disconnected; unregistering...", slot)
		closeMeSoftly()
	})
	// }}}

	r.Dispatch("GET /v2/tasks", func(r *route.Request) { // {{{
		if c.IsNotSystemEngineer(r) {
			return
		}

		limit, err := strconv.Atoi(r.Param("limit", "30"))
		if err != nil || limit < 0 || limit > 30 {
			r.Fail(route.Bad(err, "Invalid limit parameter given"))
			return
		}

		// check to see if we're offseting task requests
		paginationDate, err := strconv.ParseInt(r.Param("before", "0"), 10, 64)
		if err != nil || paginationDate < 0 {
			r.Fail(route.Bad(err, "Invalid before parameter given"))
			return
		}

		tasks, err := c.db.GetAllTasks(
			&db.TaskFilter{
				UUID:          r.Param("uuid", ""),
				ExactMatch:    r.ParamIs("exact", "t"),
				SkipActive:    r.ParamIs("active", "f"),
				SkipInactive:  r.ParamIs("active", "t"),
				ForStatus:     r.Param("status", ""),
				ForTarget:     r.Param("target", ""),
				ForOp:         r.Param("type", ""),
				Limit:         limit,
				Before:        paginationDate,
				StartedAfter:  r.ParamDuration("started_after"),
				StoppedAfter:  r.ParamDuration("stopped_after"),
				StartedBefore: r.ParamDuration("started_before"),
				StoppedBefore: r.ParamDuration("stopped_before"),
			},
		)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve task information"))
			return
		}

		r.OK(tasks)
	})
	// }}}
	r.Dispatch("GET /v2/tasks/:uuid", func(r *route.Request) { // {{{
		if c.IsNotSystemOperator(r) {
			return
		}

		task, err := c.db.GetTask(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve task information"))
			return
		}
		r.OK(task)
	})
	// }}}
	r.Dispatch("DELETE /v2/tasks/:uuid", func(r *route.Request) { // {{{
		if c.IsNotSystemOperator(r) {
			return
		}

		task, err := c.db.GetTask(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve task information"))
			return
		}
		if task == nil {
			r.Fail(route.NotFound(err, "No such task"))
			return
		}

		if err := c.db.CancelTask(task.UUID, time.Now()); err != nil {
			r.Fail(route.Oops(err, "Unable to cancel task"))
			return
		}

		r.Success("Canceled task successfully")
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

		users, err := c.db.GetAllUsers(&db.UserFilter{
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

		for _, auth := range c.providers {
			cfg := auth.Configuration(false)
			l = append(l, cfg)
		}
		r.OK(l)
	})
	// }}}
	r.Dispatch("GET /v2/auth/providers/:name", func(r *route.Request) { // {{{
		if c.IsNotSystemAdmin(r) {
			return
		}

		a, ok := c.providers[r.Args[1]]
		if !ok {
			r.Fail(route.NotFound(nil, "No such authentication provider"))
			return
		}
		r.OK(a.Configuration(true))
	})
	// }}}

	r.Dispatch("GET /v2/auth/local/users", func(r *route.Request) { // {{{
		if c.IsNotSystemManager(r) {
			return
		}

		limit, err := strconv.Atoi(r.Param("limit", "0"))
		if err != nil || limit < 0 {
			r.Fail(route.Bad(err, "Invalid limit parameter given"))
			return
		}

		l, err := c.db.GetAllUsers(&db.UserFilter{
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
			users[i] = v2LocalUser{
				UUID:    user.UUID,
				Name:    user.Name,
				Account: user.Account,
				SysRole: user.SysRole,
			}
		}

		r.OK(users)
	})
	// }}}
	r.Dispatch("GET /v2/auth/local/users/:uuid", func(r *route.Request) { // {{{
		if c.IsNotSystemManager(r) {
			return
		}

		user, err := c.db.GetUserByID(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve local user information"))
			return
		}

		if user == nil {
			r.Fail(route.NotFound(nil, "user '%s' not found (for local auth provider)", r.Args[1]))
			return
		}

		local_user := v2LocalUser{
			UUID:    user.UUID,
			Name:    user.Name,
			Account: user.Account,
			SysRole: user.SysRole,
		}

		r.OK(local_user)
	})
	// }}}
	r.Dispatch("POST /v2/auth/local/users", func(r *route.Request) { // {{{
		if c.IsNotSystemManager(r) {
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
			UUID:    in.UUID,
			Name:    in.Name,
			Account: in.Account,
			Backend: "local",
			SysRole: in.SysRole,
		}
		u.SetPassword(in.Password)

		exists, err := c.db.GetUser(u.Account, "local")
		if err != nil {
			r.Fail(route.Oops(err, "Unable to create local user '%s'", in.Account))
			return
		}

		if exists != nil {
			r.Fail(route.Bad(nil, "user '%s' already exists", u.Account))
			return
		}

		u, err = c.db.CreateUser(u)
		if u == nil || err != nil {
			r.Fail(route.Oops(err, "Unable to create local user '%s'", in.Account))
			return
		}
		r.OK(u)
	})
	// }}}
	r.Dispatch("PATCH /v2/auth/local/users/:uuid", func(r *route.Request) { // {{{
		if c.IsNotSystemManager(r) {
			return
		}

		var in struct {
			Name     string `json:"name"`
			Password string `json:"password"`
			SysRole  string `json:"sysrole"`
		}
		if !r.Payload(&in) {
			return
		}

		user, err := c.db.GetUserByID(r.Args[1])
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

		err = c.db.UpdateUser(user)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to update local user '%s'", user.Account))
			return
		}

		r.Success("Updated")
	})
	// }}}
	r.Dispatch("DELETE /v2/auth/local/users/:uuid", func(r *route.Request) { // {{{
		if c.IsNotSystemManager(r) {
			return
		}

		user, err := c.db.GetUserByID(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve local user information"))
			return
		}
		if user == nil || user.Backend != "local" {
			r.Fail(route.NotFound(nil, "Local User '%s' not found", r.Args[1]))
			return
		}

		err = c.db.DeleteUser(user)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to delete local user '%s' (%s)", r.Args[1], user.Account))
			return
		}
		r.Success("Successfully deleted local user")
	})
	// }}}

	r.Dispatch("GET /v2/auth/tokens", func(r *route.Request) { // {{{
		if c.IsNotAuthenticated(r) {
			return
		}

		user, _ := c.AuthenticatedUser(r)
		tokens, err := c.db.GetAllAuthTokens(&db.AuthTokenFilter{
			User: user,
		})
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve tokens information"))
			return
		}

		for i := range tokens {
			tokens[i].Session = ""
		}

		r.OK(tokens)
	})
	// }}}
	r.Dispatch("POST /v2/auth/tokens", func(r *route.Request) { // {{{
		if c.IsNotAuthenticated(r) {
			return
		}
		user, _ := c.AuthenticatedUser(r)

		var in struct {
			Name string `json:"name"`
		}
		if !r.Payload(&in) {
			return
		}
		if r.Missing("name", in.Name) {
			return
		}

		existing, err := c.db.GetAllAuthTokens(&db.AuthTokenFilter{
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

		token, id, err := c.db.GenerateAuthToken(in.Name, user)
		if id == "" || err != nil {
			r.Fail(route.Oops(err, "Unable to generate new token"))
			return
		}

		r.OK(token)
	})
	// }}}
	r.Dispatch("DELETE /v2/auth/tokens/:token", func(r *route.Request) { // {{{
		if c.IsNotAuthenticated(r) {
			return
		}

		user, _ := c.AuthenticatedUser(r)
		if err := c.db.DeleteAuthToken(r.Args[1], user); err != nil {
			r.Fail(route.Oops(err, "Unable to revoke auth token"))
			return
		}

		r.Success("Token revoked")
	})
	// }}}

	r.Dispatch("GET /v2/auth/sessions", func(r *route.Request) { // {{{
		if c.IsNotSystemAdmin(r) {
			return
		}

		limit, err := strconv.Atoi(r.Param("limit", "0"))
		if err != nil || limit < 0 {
			r.Fail(route.Bad(err, "Invalid limit parameter given"))
			return
		}

		sessions, err := c.db.GetAllSessions(
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
			if session.UUID == r.SessionID() {
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
		if c.IsNotSystemAdmin(r) {
			return
		}

		limit, err := strconv.Atoi(r.Param("limit", "0"))
		if err != nil || limit < 0 {
			r.Fail(route.Bad(err, "Invalid limit parameter given"))
			return
		}

		session, err := c.db.GetSession(r.Args[1])
		if session == nil || err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve session information"))
			return
		}
		if session.UUID == r.SessionID() {
			session.CurrentSession = true
		}

		r.OK(session)
	})
	// }}}
	r.Dispatch("DELETE /v2/auth/sessions/:uuid", func(r *route.Request) { // {{{
		if c.IsNotSystemAdmin(r) {
			return
		}
		session, err := c.db.GetSession(r.Args[1])
		if session == nil || err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve session information"))
			return
		}
		if session == nil {
			r.Fail(route.NotFound(nil, "Session not found"))
			return
		}

		if err := c.db.ClearSession(session.UUID); err != nil {
			r.Fail(route.Oops(err, "Unable to clear session '%s' (%s)", r.Args[1], session.IP))
			return
		}
		r.Success("Successfully cleared session '%s' (%s)", r.Args[1], session.IP)
	})
	// }}}

	// DEPRECATE THESE:
	r.Dispatch("GET /v2/systems", func(r *route.Request) { // {{{
		if c.IsNotSystemOperator(r) {
			return
		}

		targets, err := c.db.GetAllTargets(
			&db.TargetFilter{
				SkipUsed:   r.ParamIs("unused", "t"),
				SkipUnused: r.ParamIs("unused", "f"),

				UUID:       r.Param("uuid", ""),
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
			err := c.v2copyTarget(&systems[i], target)
			if err != nil {
				r.Fail(route.Oops(err, "Unable to retrieve systems information"))
				return
			}
		}

		r.OK(systems)
	})
	// }}}
	r.Dispatch("GET /v2/systems/:uuid", func(r *route.Request) { // {{{
		if c.IsNotSystemOperator(r) {
			return
		}

		target, err := c.db.GetTarget(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve system information"))
			return
		}

		if target == nil {
			r.Fail(route.NotFound(err, "No such system"))
			return
		}

		var system v2System
		err = c.v2copyTarget(&system, target)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve system information"))
			return
		}

		// keep track of our archives, indexed by task UUID
		archives := make(map[string]*db.Archive)
		aa, err := c.db.GetAllArchives(
			&db.ArchiveFilter{
				ForTarget:  target.UUID,
				WithStatus: []string{"valid"},
			},
		)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve system information"))
			return
		}
		for _, archive := range aa {
			archives[archive.UUID] = archive
		}
		// check to see if we're offseting task requests
		paginationDate, err := strconv.ParseInt(r.Param("before", "0"), 10, 64)
		if err != nil || paginationDate < 0 {
			r.Fail(route.Bad(err, "Invalid before parameter given"))
			return
		}

		tasks, err := c.db.GetAllTasks(
			&db.TaskFilter{
				ForTarget:    target.UUID,
				OnlyRelevant: true,
				Before:       paginationDate,
				Limit:        30,
			},
		)

		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve system information"))
			return
		}

		//check if there's more tasks on the specific last date and append if so
		if len(tasks) > 0 {
			appendingtasks, err := c.db.GetAllTasks(
				&db.TaskFilter{
					ForTarget:    target.UUID,
					OnlyRelevant: true,
					RequestedAt:  tasks[len(tasks)-1].RequestedAt,
				},
			)
			if err != nil {
				r.Fail(route.Oops(err, "Unable to retrieve system information"))
				return
			}
			if (len(appendingtasks) > 1) && (tasks[len(tasks)-1].UUID != appendingtasks[len(appendingtasks)-1].UUID) {
				log.Infof("Got a misjointed request, need to merge these two arrays.")
				for i, task := range appendingtasks {
					if task.UUID == tasks[len(tasks)-1].UUID {
						tasks = append(tasks, appendingtasks[i+1:]...)
						break
					}
				}
			}
		}

		system.Tasks = make([]v2SystemTask, len(tasks))
		for i, task := range tasks {
			system.Tasks[i].UUID = task.UUID
			system.Tasks[i].Type = task.Op
			system.Tasks[i].Status = task.Status
			system.Tasks[i].Owner = task.Owner
			system.Tasks[i].OK = task.OK
			system.Tasks[i].Notes = task.Notes
			system.Tasks[i].RequestedAt = task.RequestedAt
			system.Tasks[i].StartedAt = task.StartedAt
			system.Tasks[i].StoppedAt = task.StoppedAt
			system.Tasks[i].Log = task.Log

			system.Tasks[i].JobUUID = task.JobUUID
			system.Tasks[i].ArchiveUUID = task.ArchiveUUID
			system.Tasks[i].TargetUUID = task.TargetUUID

			if archive, ok := archives[task.ArchiveUUID]; ok {
				system.Tasks[i].Archive = &v2SystemArchive{
					UUID:     archive.UUID,
					Schedule: archive.Job,
					Expiry:   (int)((archive.ExpiresAt - archive.TakenAt) / 86400),
					Notes:    archive.Notes,
					Size:     archive.Size,
				}
			}
		}

		r.OK(system)
	})
	// }}}
	r.Dispatch("GET /v2/systems/:uuid/config", func(r *route.Request) { // {{{
		if c.IsNotSystemOperator(r) {
			return
		}

		target, err := c.db.GetTarget(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve system information"))
			return
		}

		if target == nil {
			r.Fail(route.NotFound(err, "No such system"))
			return
		}

		config, err := target.Configuration(c.db, true) // FIXME
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve system information"))
			return
		}

		r.OK(config)
	})
	// }}}
	r.Dispatch("POST /v2/systems", func(r *route.Request) { // {{{
		if c.IsNotSystemEngineer(r) {
			return
		}

		var in struct {
			Target struct {
				UUID    string `json:"uuid"`
				Name    string `json:"name"`
				Summary string `json:"summary"`
				Plugin  string `json:"plugin"`
				Agent   string `json:"agent"`

				Config map[string]interface{} `json:"config"`
			} `json:"target"`

			Job struct {
				Name     string `json:"name"`
				Schedule string `json:"schedule"`
				Bucket   string `json:"bucket"`
				KeepDays int    `json:"keep_days"`
				Paused   bool   `json:"paused"`

				KeepN int
			} `json:"job"`
		}
		if !r.Payload(&in) {
			return
		}

		sched, err := timespec.Parse(in.Job.Schedule)
		if err != nil {
			r.Fail(route.Oops(err, "Invalid or malformed SHIELD Job Schedule '%s'", in.Job.Schedule))
			return
		}

		if in.Job.KeepDays < 0 {
			r.Fail(route.Oops(nil, "Invalid or malformed SHIELD Job Archive Retention Period '%dd'", in.Job.KeepDays))
			return
		}
		if in.Job.KeepDays < c.Config.Limit.Retention.Min {
			r.Fail(route.Oops(nil, "SHIELD Job Archive Retention Period '%dd' is too short, archives must be kept for a minimum of %d days", in.Job.KeepDays, c.Config.Limit.Retention.Min))
			return
		}
		if in.Job.KeepDays > c.Config.Limit.Retention.Max {
			r.Fail(route.Oops(nil, "SHIELD Job Archive Retention Period '%dd' is too long, archives may be kept for a maximum of %d days", in.Job.KeepDays, c.Config.Limit.Retention.Max))
			return
		}
		in.Job.KeepN = sched.KeepN(in.Job.KeepDays)

		var target *db.Target
		if in.Target.UUID != "" {
			target, err = c.db.GetTarget(in.Target.UUID)
			if err != nil {
				r.Fail(route.Oops(err, "Unable to retrieve system information"))
				return
			}
			if target == nil {
				r.Fail(route.NotFound(nil, "No such system"))
				return
			}

		} else {
			target, err = c.db.CreateTarget(&db.Target{
				Name:    in.Target.Name,
				Summary: in.Target.Summary,
				Plugin:  in.Target.Plugin,
				Config:  in.Target.Config,
				Agent:   in.Target.Agent,
				Healthy: true,
			})
			if target == nil || err != nil {
				r.Fail(route.Oops(err, "Unable to create new data target"))
				return
			}
		}

		job, err := c.db.CreateJob(&db.Job{
			Name:       in.Job.Name,
			Schedule:   in.Job.Schedule,
			Bucket:     in.Job.Bucket,
			KeepN:      in.Job.KeepN,
			KeepDays:   in.Job.KeepDays,
			Paused:     in.Job.Paused,
			TargetUUID: target.UUID,
		})
		if job == nil || err != nil {
			r.Fail(route.Oops(err, "Unable to create new job"))
			return
		}

		r.OK(target)
	})
	// }}}
	r.Dispatch("PATCH /v2/systems/:uuid", func(r *route.Request) { // {{{
		if c.IsNotSystemEngineer(r) {
			return
		}

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

		target, err := c.db.GetTarget(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve system information"))
			return
		}

		if target == nil {
			r.Fail(route.NotFound(nil, "No such system"))
			return
		}

		for _, ann := range in.Annotations {
			switch ann.Type {
			case "task":
				err = c.db.AnnotateTargetTask(
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
				err = c.db.AnnotateTargetArchive(
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

		_ = c.db.MarkTasksIrrelevant()
		r.Success("annotated successfully")
	})
	// }}}
	r.Dispatch("DELETE /v2/systems/:uuid", func(r *route.Request) { // {{{
		if c.IsNotSystemEngineer(r) {
			return
		}

		/* FIXME */
		r.Fail(route.Errorf(501, nil, "%s: not implemented", r))
	})
	// }}}

	r.Dispatch("GET /v2/agents", func(r *route.Request) { // {{{
		if c.IsNotSystemAdmin(r) {
			return
		}

		agents, err := c.db.GetAllAgents(&db.AgentFilter{
			UUID:        r.Param("uuid", ""),
			ExactMatch:  r.ParamIs("exact", "t"),
			SkipHidden:  r.ParamIs("hidden", "f"),
			SkipVisible: r.ParamIs("hidden", "t"),
		})
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
			id := agent.UUID
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
		if c.IsNotSystemAdmin(r) {
			return
		}

		agent, err := c.db.GetAgent(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve agent information"))
			return
		}
		if agent == nil {
			r.Fail(route.NotFound(nil, "No such agent"))
			return
		}

		raw, err := agent.Metadata()
		if err != nil {
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
	r.Dispatch("DELETE /v2/agents/:uuid", func(r *route.Request) { // {{{
		if c.IsNotSystemAdmin(r) {
			return
		}

		agent, err := c.db.GetAgent(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve agent information"))
			return
		}
		if agent == nil {
			r.Fail(route.NotFound(nil, "No such agent"))
			return
		}

		err = c.db.DeleteAgent(agent)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to delete agent"))
			return
		}

		r.Success("deleted agent %s (at %s)", agent.Name, agent.Address)
	})
	// }}}
	r.Dispatch("POST /v2/agents", func(r *route.Request) { // {{{
		var in struct {
			Name     string `json:"name"`
			Port     int    `json:"port"`
			Token    string `json:"token"`
			Endpoint string `json:"endpoint"`
		}
		if !r.Payload(&in) {
			return
		}

		peer := in.Endpoint
		if peer == "" {
			peer = regexp.MustCompile(`:\d+$`).ReplaceAllString(r.Req.Header.Get("X-Forwarded-For"), "")
			if peer == "" {
				peer = regexp.MustCompile(`:\d+$`).ReplaceAllString(r.Req.RemoteAddr, "")
				if peer == "" {
					r.Fail(route.Oops(nil, "Unable to determine remote peer address from '%s'", r.Req.RemoteAddr))
					return
				}
			}
		}

		if in.Name == "" {
			r.Fail(route.Bad(nil, "No `name' provided with pre-registration request"))
			return
		}
		if in.Port == 0 {
			r.Fail(route.Bad(nil, "No `port' provided with pre-registration request"))
			return
		}

		if in.Token == "" || in.Token != c.Config.LegacyAgents.RegistrationToken {
			r.Fail(route.Forbidden(nil, "Invalid agent registration token supplied"))
			return
		}

		err := c.db.PreRegisterAgent(peer, in.Name, in.Port)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to pre-register agent %s at %s:%d", in.Name, peer, in.Port))
			return
		}
		r.Success("pre-registered agent %s at %s:%d", in.Name, peer, in.Port)
	})
	// }}}
	r.Dispatch("POST /v2/agents/:uuid/(show|hide)", func(r *route.Request) { // {{{
		if c.IsNotSystemAdmin(r) {
			return
		}

		agent, err := c.db.GetAgent(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve agent information"))
			return
		}
		if agent == nil {
			r.Fail(route.NotFound(nil, "No such agent"))
			return
		}

		agent.Hidden = (r.Args[2] == "hide")
		if err := c.db.UpdateAgent(agent); err != nil {
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
	r.Dispatch("POST /v2/agents/:uuid/resync", func(r *route.Request) { // {{{
		if c.IsNotSystemAdmin(r) {
			return
		}

		agent, err := c.db.GetAgent(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve agent information"))
			return
		}
		if agent == nil {
			r.Fail(route.NotFound(nil, "No such agent"))
			return
		}

		c.ScheduleAgentStatusCheckTasks(&db.AgentFilter{UUID: agent.UUID})
		r.Success("Ad hoc agent resynchronization underway")
	})
	// }}}

	r.Dispatch("GET /v2/targets", func(r *route.Request) { // {{{
		if c.IsNotSystemOperator(r) {
			return
		}

		targets, err := c.db.GetAllTargets(
			&db.TargetFilter{
				SkipUsed:   r.ParamIs("unused", "t"),
				SkipUnused: r.ParamIs("unused", "f"),

				UUID:       r.Param("uuid", ""),
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
	r.Dispatch("POST /v2/targets", func(r *route.Request) { // {{{
		if c.IsNotSystemEngineer(r) {
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
				return
			}
			in.endpoint = string(b)
		} else {
			in.endpoint = "{}"
		}
		if r.Missing("name", in.Name, "plugin", in.Plugin, "agent", in.Agent) {
			return
		}

		if r.ParamIs("test", "t") {
			r.Success("validation suceeded (request made in ?test=t mode)")
			return
		}

		target, err := c.db.CreateTarget(&db.Target{
			Name:    in.Name,
			Summary: in.Summary,
			Plugin:  in.Plugin,
			Config:  in.Config,
			Agent:   in.Agent,
			Healthy: true,
		})
		if target == nil || err != nil {
			r.Fail(route.Oops(err, "Unable to create new data target"))
			return
		}

		r.OK(target)
	})
	// }}}
	r.Dispatch("GET /v2/targets/:uuid", func(r *route.Request) { // {{{
		if c.IsNotSystemOperator(r) {
			return
		}

		target, err := c.db.GetTarget(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve target information"))
			return
		}

		if target == nil {
			r.Fail(route.NotFound(nil, "No such target"))
			return
		}

		r.OK(target)
	})
	// }}}
	r.Dispatch("PUT /v2/targets/:uuid", func(r *route.Request) { // {{{
		if c.IsNotSystemEngineer(r) {
			return
		}

		target, err := c.db.GetTarget(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve target information"))
			return
		}

		if target == nil {
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

		if err := c.db.UpdateTarget(target); err != nil {
			r.Fail(route.Oops(err, "Unable to update target"))
			return
		}

		r.Success("Updated target successfully")
	})
	// }}}
	r.Dispatch("DELETE /v2/targets/:uuid", func(r *route.Request) { // {{{
		if c.IsNotSystemEngineer(r) {
			return
		}

		target, err := c.db.GetTarget(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve target information"))
			return
		}

		if target == nil {
			r.Fail(route.NotFound(nil, "No such target"))
			return
		}

		deleted, err := c.db.DeleteTarget(target.UUID)
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

	r.Dispatch("GET /v2/jobs", func(r *route.Request) { // {{{
		if c.IsNotSystemOperator(r) {
			return
		}

		jobs, err := c.db.GetAllJobs(
			&db.JobFilter{
				SkipPaused:   r.ParamIs("paused", "f"),
				SkipUnpaused: r.ParamIs("paused", "t"),

				UUID:       r.Param("uuid", ""),
				SearchName: r.Param("name", ""),

				ForTarget:  r.Param("target", ""),
				ForBucket:  r.Param("bucket", ""),
				ExactMatch: r.ParamIs("exact", "t"),
			},
		)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve job information."))
			return
		}

		r.OK(jobs)
	})
	// }}}
	r.Dispatch("POST /v2/jobs", func(r *route.Request) { // {{{
		if c.IsNotSystemEngineer(r) {
			return
		}

		var in struct {
			Name     string `json:"name"`
			Summary  string `json:"summary"`
			Schedule string `json:"schedule"`
			Paused   bool   `json:"paused"`
			Bucket   string `json:"bucket"`
			Target   string `json:"target"`
			Retain   string `json:"retain"`
		}
		if !r.Payload(&in) {
			return
		}

		if r.Missing("name", in.Name, "bucket", in.Bucket, "target", in.Target, "schedule", in.Schedule, "retain", in.Retain) {
			return
		}

		sched, err := timespec.Parse(in.Schedule)
		if err != nil {
			r.Fail(route.Oops(err, "Invalid or malformed SHIELD Job Schedule '%s'", in.Schedule))
			return
		}

		keepdays := util.ParseRetain(in.Retain)
		if keepdays < 0 {
			r.Fail(route.Oops(nil, "Invalid or malformed SHIELD Job Archive Retention Period '%s'", in.Retain))
			return
		}
		if keepdays < c.Config.Limit.Retention.Min {
			r.Fail(route.Oops(nil, "SHIELD Job Archive Retention Period '%s' is too short, archives must be kept for a minimum of %d days", in.Retain, c.Config.Limit.Retention.Min))
			return
		}
		if keepdays > c.Config.Limit.Retention.Max {
			r.Fail(route.Oops(nil, "SHIELD Job Archive Retention Period '%s' is too long, archives may be kept for a maximum of %d days", in.Retain, c.Config.Limit.Retention.Max))
			return
		}
		keepn := sched.KeepN(keepdays)

		job, err := c.db.CreateJob(&db.Job{
			Name:       in.Name,
			Summary:    in.Summary,
			Schedule:   in.Schedule,
			KeepDays:   keepdays,
			KeepN:      keepn,
			Paused:     in.Paused,
			Bucket:     in.Bucket,
			TargetUUID: in.Target,
		})
		if job == nil || err != nil {
			r.Fail(route.Oops(err, "Unable to create new job"))
			return
		}

		r.OK(job)
	})
	// }}}
	r.Dispatch("GET /v2/jobs/:uuid", func(r *route.Request) { // {{{
		if c.IsNotSystemOperator(r) {
			return
		}

		job, err := c.db.GetJob(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve job information"))
			return
		}

		if job == nil {
			r.Fail(route.NotFound(nil, "No such job"))
			return
		}

		r.OK(job)
	})
	// }}}
	r.Dispatch("PUT /v2/jobs/:uuid", func(r *route.Request) { // {{{
		if c.IsNotSystemEngineer(r) {
			return
		}

		var in struct {
			Name     string `json:"name"`
			Summary  string `json:"summary"`
			Schedule string `json:"schedule"`
			Retain   string `json:"retain"`

			Bucket     string `json:"bucket"`
			TargetUUID string `json:"target"`
		}
		if !r.Payload(&in) {
			return
		}

		job, err := c.db.GetJob(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve job information"))
			return
		}
		if job == nil {
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
			if _, err := timespec.Parse(in.Schedule); err != nil {
				r.Fail(route.Oops(err, "Invalid or malformed SHIELD Job Schedule '%s'", in.Schedule))
				return
			}
			job.Schedule = in.Schedule
		}
		if in.Retain != "" {
			keepdays := util.ParseRetain(in.Retain)
			if keepdays < 0 {
				r.Fail(route.Oops(nil, "Invalid or malformed SHIELD Job Archive Retention Period '%s'", in.Retain))
				return
			}
			if keepdays < c.Config.Limit.Retention.Min {
				r.Fail(route.Oops(nil, "SHIELD Job Archive Retention Period '%s' is too short, archives must be kept for a minimum of %d days", in.Retain, c.Config.Limit.Retention.Min))
				return
			}
			if keepdays > c.Config.Limit.Retention.Max {
				r.Fail(route.Oops(nil, "SHIELD Job Archive Retention Period '%s' is too long, archives may be kept for a maximum of %d days", in.Retain, c.Config.Limit.Retention.Max))
				return
			}

			job.KeepDays = keepdays
			job.KeepN = -1
			if sched, err := timespec.Parse(job.Schedule); err == nil {
				job.KeepN = sched.KeepN(job.KeepDays)
			}
		}

		job.TargetUUID = job.Target.UUID
		if in.TargetUUID != "" {
			job.TargetUUID = in.TargetUUID
		}
		if in.Bucket != "" {
			job.Bucket = in.Bucket
		}

		if err := c.db.UpdateJob(job); err != nil {
			r.Fail(route.Oops(err, "Unable to update job"))
			return
		}

		if in.Schedule != "" {
			if spec, err := timespec.Parse(in.Schedule); err == nil {
				if next, err := spec.Next(time.Now()); err == nil {
					c.db.RescheduleJob(job, next)
				}
			}
		}

		r.Success("Updated job successfully")
	})
	// }}}
	r.Dispatch("DELETE /v2/jobs/:uuid", func(r *route.Request) { // {{{
		if c.IsNotSystemEngineer(r) {
			return
		}

		job, err := c.db.GetJob(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve job information"))
			return
		}

		if job == nil {
			r.Fail(route.NotFound(nil, "No such job"))
			return
		}

		deleted, err := c.db.DeleteJob(job.UUID)
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
	r.Dispatch("POST /v2/jobs/:uuid/run", func(r *route.Request) { // {{{
		if c.IsNotSystemOperator(r) {
			return
		}

		job, err := c.db.GetJob(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve job information"))
			return
		}

		if job == nil {
			r.Fail(route.NotFound(nil, "No such job"))
			return
		}

		user, _ := c.AuthenticatedUser(r)
		task, err := c.db.CreateBackupTask(fmt.Sprintf("%s@%s", user.Account, user.Backend), job)
		if task == nil || err != nil {
			r.Fail(route.Oops(err, "Unable to schedule ad hoc backup job run"))
			return
		}

		var out struct {
			OK       string `json:"ok"`
			TaskUUID string `json:"task_uuid"`
		}

		out.OK = "Scheduled ad hoc backup job run"
		out.TaskUUID = task.UUID
		r.OK(out)
	})
	// }}}
	r.Dispatch("POST /v2/jobs/:uuid/pause", func(r *route.Request) { // {{{
		if c.IsNotSystemOperator(r) {
			return
		}

		job, err := c.db.GetJob(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve job information"))
			return
		}

		if job == nil {
			r.Fail(route.NotFound(nil, "No such job"))
			return
		}

		if _, err = c.db.PauseJob(job.UUID); err != nil {
			r.Fail(route.Oops(err, "Unable to pause job"))
			return
		}
		r.Success("Paused job successfully")
	})
	// }}}
	r.Dispatch("POST /v2/jobs/:uuid/unpause", func(r *route.Request) { // {{{
		if c.IsNotSystemOperator(r) {
			return
		}

		job, err := c.db.GetJob(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve job information"))
			return
		}

		if job == nil {
			r.Fail(route.NotFound(nil, "No such job"))
			return
		}

		if _, err = c.db.UnpauseJob(job.UUID); err != nil {
			r.Fail(route.Oops(err, "Unable to unpause job"))
			return
		}
		r.Success("Unpaused job successfully")
	})
	// }}}

	r.Dispatch("GET /v2/tasks", func(r *route.Request) { // {{{
		if c.IsNotSystemOperator(r) {
			return
		}

		limit, err := strconv.Atoi(r.Param("limit", "30"))
		if err != nil || limit < 0 || limit > 30 {
			r.Fail(route.Bad(err, "Invalid limit parameter given"))
			return
		}

		// check to see if we're offseting task requests
		paginationDate, err := strconv.ParseInt(r.Param("before", "0"), 10, 64)
		if err != nil || paginationDate < 0 {
			r.Fail(route.Bad(err, "Invalid before parameter given"))
			return
		}

		tasks, err := c.db.GetAllTasks(
			&db.TaskFilter{
				UUID:          r.Param("uuid", ""),
				ExactMatch:    r.ParamIs("exact", "t"),
				SkipActive:    r.ParamIs("active", "f"),
				SkipInactive:  r.ParamIs("active", "t"),
				ForStatus:     r.Param("status", ""),
				ForTarget:     r.Param("target", ""),
				ForOp:         r.Param("type", ""),
				Limit:         limit,
				Before:        paginationDate,
				StartedAfter:  r.ParamDuration("started_after"),
				StoppedAfter:  r.ParamDuration("stopped_after"),
				StartedBefore: r.ParamDuration("started_before"),
				StoppedBefore: r.ParamDuration("stopped_before"),
			},
		)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve task information"))
			return
		}

		r.OK(tasks)
	})
	// }}}
	r.Dispatch("GET /v2/tasks/:uuid", func(r *route.Request) { // {{{
		if c.IsNotSystemOperator(r) {
			return
		}

		task, err := c.db.GetTask(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve task information"))
			return
		}
		if task == nil {
			r.Fail(route.NotFound(err, "No such task"))
			return
		}

		r.OK(task)
	})
	// }}}
	r.Dispatch("DELETE /v2/tasks/:uuid", func(r *route.Request) { // {{{
		if c.IsNotSystemOperator(r) {
			return
		}

		task, err := c.db.GetTask(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve task information"))
			return
		}
		if task == nil {
			r.Fail(route.NotFound(err, "No such task"))
			return
		}

		if err := c.db.CancelTask(task.UUID, time.Now()); err != nil {
			r.Fail(route.Oops(err, "Unable to cancel task"))
			return
		}

		r.Success("Canceled task successfully")
	})
	// }}}

	r.Dispatch("GET /v2/archives", func(r *route.Request) { // {{{
		if c.IsNotSystemOperator(r) {
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

		archives, err := c.db.GetAllArchives(
			&db.ArchiveFilter{
				UUID:       r.Param("uuid", ""),
				ExactMatch: r.ParamIs("exact", "t"),
				ForTarget:  r.Param("target", ""),
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
	r.Dispatch("GET /v2/archives/:uuid", func(r *route.Request) { // {{{
		if c.IsNotSystemOperator(r) {
			return
		}

		archive, err := c.db.GetArchive(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve backup archive information"))
			return
		}

		if archive == nil {
			r.Fail(route.NotFound(nil, "Archive Not Found"))
			return
		}

		r.OK(archive)
	})
	// }}}
	r.Dispatch("PUT /v2/archives/:uuid", func(r *route.Request) { // {{{
		if c.IsNotSystemOperator(r) {
			return
		}

		var in struct {
			Notes string `json:"notes"`
		}
		if !r.Payload(&in) {
			return
		}

		archive, err := c.db.GetArchive(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve backup archive information"))
			return
		}

		if archive == nil {
			r.Fail(route.NotFound(nil, "No such backup archive"))
			return
		}

		if r.Missing("notes", in.Notes) {
			return
		}

		archive.Notes = in.Notes
		if err := c.db.UpdateArchive(archive); err != nil {
			r.Fail(route.Oops(err, "Unable to update backup archive"))
			return
		}

		r.OK(archive)
	})
	// }}}
	r.Dispatch("DELETE /v2/archives/:uuid", func(r *route.Request) { // {{{
		if c.IsNotSystemOperator(r) {
			return
		}

		archive, err := c.db.GetArchive(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve backup archive information"))
			return
		}

		if archive == nil {
			r.Fail(route.NotFound(nil, "No such backup archive"))
			return
		}

		if archive.Status != "valid" {
			r.Fail(route.Bad(err, "The backup archive could not be deleted at this time. Archive is already %s", archive.Status))
		}

		err = c.db.ManuallyPurgeArchive(archive.UUID)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to delete backup archive"))
			return
		}

		r.Success("Archive deleted successfully")
	})
	// }}}
	r.Dispatch("POST /v2/archives/:uuid/restore", func(r *route.Request) { // {{{
		if c.IsNotSystemOperator(r) {
			return
		}

		var in struct {
			Target string `json:"target"`
		}
		if !r.Payload(&in) {
			return
		}

		archive, err := c.db.GetArchive(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve backup archive information"))
			return
		}
		if archive == nil {
			r.Fail(route.NotFound(nil, "No such backup archive"))
			return
		}

		if in.Target == "" {
			in.Target = archive.TargetUUID
		}

		target, err := c.db.GetTarget(in.Target)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve backup archive information"))
			return
		}

		if target == nil {
			r.Fail(route.NotFound(nil, "No such backup archive"))
			return
		}

		user, _ := c.AuthenticatedUser(r)
		task, err := c.db.CreateRestoreTask(fmt.Sprintf("%s@%s", user.Account, user.Backend), archive, target)
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

		user, err := c.db.GetUser(in.Username, "local")
		if err != nil {
			r.Fail(route.Oops(err, "Unable to log you in"))
			return
		}

		if user == nil || !user.Authenticate(in.Password) {
			r.Fail(route.Errorf(401, nil, "Incorrect username or password"))
			return
		}

		session, err := c.db.CreateSession(&db.Session{
			UserUUID:  user.UUID,
			IP:        r.RemoteIP(),
			UserAgent: r.UserAgent(),
		})
		if err != nil {
			r.Fail(route.Oops(err, "Unable to log you in"))
			return
		}
		if session == nil {
			r.Fail(route.Oops(fmt.Errorf("no session created"), "Unable to log you in"))
			return
		}

		id, err := c.checkAuth(user)
		if err != nil || id == nil {
			r.Fail(route.Oops(err, "Unable to log you in"))
		}

		r.SetSession(session.UUID)
		r.OK(id)
	})
	// }}}
	r.Dispatch("GET /v2/auth/logout", func(r *route.Request) { // {{{
		if err := c.db.ClearSession(r.SessionID()); err != nil {
			r.Fail(route.Oops(err, "Unable to log you out"))
			return
		}

		r.ClearSession()
		r.Success("Successfully logged out")
	})
	// }}}
	r.Dispatch("GET /v2/auth/id", func(r *route.Request) { // {{{
		user, _ := c.AuthenticatedUser(r)
		if id, _ := c.checkAuth(user); id != nil {
			r.OK(id)
			return
		}

		r.OK(struct {
			Unauthenticated bool `json:"unauthenticated"`
		}{true})
	})
	// }}}
	r.Dispatch("POST /v2/auth/passwd", func(r *route.Request) { // {{{
		if c.IsNotAuthenticated(r) {
			return
		}

		var in struct {
			OldPassword string `json:"old_password"`
			NewPassword string `json:"new_password"`
		}

		if !r.Payload(&in) {
			return
		}

		user, _ := c.AuthenticatedUser(r)
		if !user.Authenticate(in.OldPassword) {
			r.Fail(route.Forbidden(nil, "Incorrect password"))
			return
		}

		user.SetPassword(in.NewPassword)
		if err := c.db.UpdateUser(user); err != nil {
			r.Fail(route.Oops(err, "Unable to change your password"))
			return
		}

		r.Success("Password changed successfully")
	})
	// }}}

	r.Dispatch("GET /v2/fixups", func(r *route.Request) { // {{{
		if c.IsNotSystemEngineer(r) {
			return
		}

		fixups, err := c.db.GetAllFixups(nil)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve data fixups information"))
			return
		}

		r.OK(fixups)
	})
	// }}}
	r.Dispatch("GET /v2/fixups/:id", func(r *route.Request) { // {{{
		if c.IsNotSystemEngineer(r) {
			return
		}
		fixup, err := c.db.GetFixup(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve data fixup"))
			return
		}

		r.OK(fixup)
	})
	// }}}
	r.Dispatch("POST /v2/fixups/:id/apply", func(r *route.Request) { // {{{
		if c.IsNotSystemEngineer(r) {
			return
		}

		fixup, err := c.db.GetFixup(r.Args[1])
		if err != nil {
			r.Fail(route.Oops(err, "Unable to retrieve data fixups information"))
			return
		}

		err = fixup.ReApply(c.db)
		if err != nil {
			r.Fail(route.Oops(err, "Unable to apply data fixup successfully"))
			return
		}

		r.Success("applied fixup")
	})
	// }}}

	r.Dispatch("GET /v2/export", func(r *route.Request) { // {{{
		if c.IsNotSystemAdmin(r) {
			return
		}

		if out := r.JSONEncoder(); out != nil {
			c.db.Export(out)
		} else {
			r.Fail(route.Oops(nil, "Failed to export SHIELD data"))
		}
	}) // }}}
	r.Dispatch("POST /v2/import", func(r *route.Request) { // {{{
		if c.IsNotSystemAdmin(r) {
			return
		}

		if in := r.JSONDecoder(); in != nil {

			err := c.db.Import(in, r.Param("key", ""), r.Param("task", ""))
			if err != nil {
				r.Fail(route.Oops(err, "Failed to import SHIELD data"))
				return
			}
			r.Success("imported successfully: %s    %s", r.Param("key", ""), r.Param("task", ""))
		} else {
			r.Fail(route.Oops(nil, "Failed to import SHIELD data"))
		}
	}) // }}}

	return r
}

func (c *Core) v2copyTarget(dst *v2System, target *db.Target) error {
	dst.UUID = target.UUID
	dst.Name = target.Name
	dst.Notes = target.Summary
	dst.OK = true

	jobs, err := c.db.GetAllJobs(&db.JobFilter{ForTarget: target.UUID})
	if err != nil {
		return err
	}

	dst.Jobs = make([]v2SystemJob, len(jobs))
	for j, job := range jobs {
		dst.Jobs[j].UUID = job.UUID
		dst.Jobs[j].Schedule = job.Schedule
		dst.Jobs[j].From = job.Target.Plugin
		dst.Jobs[j].OK = job.Healthy

		if !job.Healthy {
			dst.OK = false
		}

		tspec, err := timespec.Parse(job.Schedule)
		if err != nil {
			return err
		}
		switch tspec.Interval {
		case timespec.Minutely:
			dst.Jobs[j].Keep.N = dst.Jobs[j].Keep.Days * 1440 / int(tspec.Cardinality)
		case timespec.Hourly:
			if tspec.Cardinality == 0 {
				dst.Jobs[j].Keep.N = dst.Jobs[j].Keep.Days * 24
			} else {
				dst.Jobs[j].Keep.N = dst.Jobs[j].Keep.Days * 24 / int(tspec.Cardinality)
			}
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
