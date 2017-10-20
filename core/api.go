package core

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/goutils/log"

	"github.com/starkandwayne/shield/db"
	"github.com/starkandwayne/shield/timespec"
)

//APIVersion is the maximum supported version of the core Shield Daemon API.
// Supported as of Version 2.
const APIVersion = 2

func (core *Core) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch {
	case match(req, `GET /init.js`):
		core.initJS(w, req)
		return

	case match(req, `GET /auth/([^/]+)/(redir|web|cli)`):
		re := regexp.MustCompile("/auth/([^/]+)/(redir|web|cli)")
		m := re.FindStringSubmatch(req.URL.Path)

		name := m[1]
		provider, ok := core.auth[name]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "Unrecognized authentication provider %s", name)
			return
		}

		if m[2] == "redir" {
			via := "web"
			if cookie, err := req.Cookie("via"); err == nil {
				via = cookie.Value
			}
			log.Debugf("handling redirection for authentication provider flow; via='%s'", via)

			user := provider.HandleRedirect(req)
			if user == nil {
				fmt.Fprintf(w, "The authentication process broke down\n")
				w.WriteHeader(500)
			}

			session, err := core.createSession(user)
			if err != nil {
				log.Errorf("failed to create a session for user %s@%s: %s", user.Account, user.Backend, err)
				w.Header().Set("Location", "/")
			} else if via == "cli" {
				w.Header().Set("Location", fmt.Sprintf("/#!/cliauth:s:%s", session.UUID.String()))
			} else {
				w.Header().Set("Location", "/")

				if session, err := core.createSession(user); err != nil {
					log.Errorf("failed to create a session for user %s@%s: %s", user.Account, user.Backend, err)
				} else {
					http.SetCookie(w, SessionCookie(session.UUID.String(), true))
				}
			}
			w.WriteHeader(302)

		} else {
			http.SetCookie(w, &http.Cookie{
				Name:  "via",
				Value: m[2],
				Path:  "/auth",
			})
			provider.Initiate(w, req)
		}
		return

	case match(req, `GET /v1/ping`):
		core.v1Ping(w, req)
		return

	case match(req, `GET /v1/status`):
		core.v1Status(w, req)
		return

	case match(req, `GET /v1/meta/pubkey`):
		core.v1GetPublicKey(w, req)
		return

	//All api endpoints below have the mustBeUnlocked requirement such that if vault
	//	is sealed or uninitialized they will return a 401
	case match(req, `GET /v1/status/internal`):
		core.v1DetailedStatus(w, req)
		return
	case match(req, `GET /v1/status/jobs`):
		core.v1JobsStatus(w, req)
		return

	case match(req, `GET /v1/archives`):
		core.v1GetArchives(w, req)
		return
	case match(req, `POST /v1/archive/[a-fA-F0-9-]+/restore`):
		if locked := core.mustBeUnlocked(w); locked {
			return
		}
		core.v1RestoreArchive(w, req)
		return
	case match(req, `GET /v1/archive/[a-fA-F0-9-]+`):
		core.v1GetArchive(w, req)
		return
	case match(req, `PUT /v1/archive/[a-fA-F0-9-]+`):
		if locked := core.mustBeUnlocked(w); locked {
			return
		}
		core.v1UpdateArchive(w, req)
		return
	case match(req, `DELETE /v1/archive/[a-fA-F0-9-]+`):
		if locked := core.mustBeUnlocked(w); locked {
			return
		}
		core.v1DeleteArchive(w, req)
		return

	case match(req, `GET /v1/jobs`):
		core.v1GetJobs(w, req)
		return
	case match(req, `POST /v1/jobs`):
		core.v1CreateJob(w, req)
		return
	case match(req, `POST /v1/job/[a-fA-F0-9-]+/pause`):
		core.v1PauseJob(w, req)
		return
	case match(req, `POST /v1/job/[a-fA-F0-9-]+/unpause`):
		core.v1UnpauseJob(w, req)
		return
	case match(req, `POST /v1/job/[a-fA-F0-9-]+/run`):
		if locked := core.mustBeUnlocked(w); locked {
			return
		}
		core.v1RunJob(w, req)
		return
	case match(req, `GET /v1/job/[a-fA-F0-9-]+`):
		core.v1GetJob(w, req)
		return
	case match(req, `PUT /v1/job/[a-fA-F0-9-]+`):
		core.v1UpdateJob(w, req)
		return
	case match(req, `DELETE /v1/job/[a-fA-F0-9-]+`):
		if locked := core.mustBeUnlocked(w); locked {
			return
		}
		core.v1DeleteJob(w, req)
		return

	case match(req, `GET /v1/retention`):
		core.v1GetRetentionPolicies(w, req)
		return
	case match(req, `POST /v1/retention`):
		core.v1CreateRetentionPolicy(w, req)
		return
	case match(req, `GET /v1/retention/[a-fA-F0-9-]+`):
		core.v1GetRetentionPolicy(w, req)
		return
	case match(req, `PUT /v1/retention/[a-fA-F0-9-]+`):
		core.v1UpdateRetentionPolicy(w, req)
		return
	case match(req, `DELETE /v1/retention/[a-fA-F0-9-]+`):
		core.v1DeleteRetentionPolicy(w, req)
		return

	case match(req, `GET /v1/stores`):
		core.v1GetStores(w, req)
		return
	case match(req, `POST /v1/stores`):
		core.v1CreateStore(w, req)
		return
	case match(req, `GET /v1/store/[a-fA-F0-9-]+`):
		core.v1GetStore(w, req)
		return
	case match(req, `PUT /v1/store/[a-fA-F0-9-]+`):
		core.v1UpdateStore(w, req)
		return
	case match(req, `DELETE /v1/store/[a-fA-F0-9-]+`):
		core.v1DeleteStore(w, req)
		return

	case match(req, `GET /v1/targets`):
		core.v1GetTargets(w, req)
		return
	case match(req, `POST /v1/targets`):
		core.v1CreateTarget(w, req)
		return
	case match(req, `GET /v1/target/[a-fA-F0-9-]+`):
		core.v1GetTarget(w, req)
		return
	case match(req, `PUT /v1/target/[a-fA-F0-9-]+`):
		core.v1UpdateTarget(w, req)
		return
	case match(req, `DELETE /v1/target/[a-fA-F0-9-]+`):
		core.v1DeleteTarget(w, req)
		return

	case match(req, `GET /v1/tasks`):
		if locked := core.mustBeUnlocked(w); locked {
			return
		}
		core.v1GetTasks(w, req)
		return
	case match(req, `GET /v1/task/[a-fA-F0-9-]+`):
		if locked := core.mustBeUnlocked(w); locked {
			return
		}
		core.v1GetTask(w, req)
		return
	case match(req, `DELETE /v1/task/[a-fA-F0-9-]+`):
		if locked := core.mustBeUnlocked(w); locked {
			return
		}
		core.v1CancelTask(w, req)
		return
	}

	w.WriteHeader(501)
}

func match(req *http.Request, pattern string) bool {
	uuider := regexp.MustCompile(":uuid")
	pattern = uuider.ReplaceAllString(pattern, "[a-fA-F0-9-]+")

	matched, _ := regexp.MatchString(
		fmt.Sprintf("^%s$", pattern),
		fmt.Sprintf("%s %s", req.Method, req.URL.Path))
	return matched
}

func bail(w http.ResponseWriter, e error) {
	w.WriteHeader(500)
	log.Errorf("Request bailed: <%s>", e)
}

func bailWithError(w http.ResponseWriter, err JSONError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(400)
	w.Write([]byte(err.JSON()))
	log.Errorf("Request bailed: <%s>", err)
}

func JSON(w http.ResponseWriter, thing interface{}) {
	bytes, err := json.Marshal(thing)
	if err != nil {
		log.Errorf("Cannot marshal JSON: <%s>\n", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(bytes)
}

func JSONLiteral(w http.ResponseWriter, thing string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write([]byte(thing))
}

func paramEquals(req *http.Request, name string, value string) bool {
	actual, set := req.URL.Query()[name]
	return set && actual[0] == value
}

func paramValue(req *http.Request, name string, defval string) string {
	value, set := req.URL.Query()[name]
	if set {
		return value[0]
	}
	return defval
}

func paramDate(req *http.Request, name string) *time.Time {
	value, set := req.URL.Query()[name]
	if !set {
		return nil
	}

	t, err := time.Parse("20060102", value[0])
	if err != nil {
		return nil
	}
	return &t
}

func (core *Core) mustBeUnlocked(w http.ResponseWriter) bool {
	status, err := core.vault.Status()
	if err != nil {
		bail(w, err)
		return true
	}
	if status == "unsealed" {
		return false
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(403)
	w.Write([]byte(ClientErrorf("Shield is currently locked").JSON()))
	return true
}

func (core *Core) initJS(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(200)

	fmt.Fprintf(w, "// init.js\n")
	fmt.Fprintf(w, "var $global = {}\n")

	health, err := core.checkHealth()
	if err != nil {
		log.Errorf("init.js: failed to check health of SHIELD core: %s", err)
		fmt.Fprintf(w, "// failed to determine health of SHIELD core...\n")
	} else {
		b, err := json.Marshal(health)
		if err != nil {
			log.Errorf("init.js: failed to marshal health data into JSON: %s", err)
			fmt.Fprintf(w, "// failed to determine health of SHIELD core...\n")
			fmt.Fprintf(w, "$global.hud = {\"health\":{\"shield\":{\"core\":\"down\"}}};\n")
		} else {
			fmt.Fprintf(w, "$global.hud = %s;\n", string(b))
		}
	}

	const unauthFail = "// failed to determine user authentication state...\n"
	const unauthJS = "$global.auth = {\"unauthenticated\":true};\n"

	user, err := core.DB.GetUserForSession(getSessionID(req))
	if err != nil {
		log.Errorf("init.js: failed to get user from session: %s", err)
		fmt.Fprintf(w, unauthFail)
		fmt.Fprintf(w, unauthJS)
		return
	}
	if user == nil {
		fmt.Fprintf(w, unauthJS)
		return
	}

	id, err := core.checkAuth(user)
	if err != nil {
		log.Errorf("failed to obtain tenancy info about user: %s", err)
		fmt.Fprintf(w, unauthFail)
		fmt.Fprintf(w, unauthJS)
		return
	}
	b, err := json.Marshal(id)
	if err != nil {
		log.Errorf("init.js: failed to marshal auth id data into JSON: %s", err)
		fmt.Fprintf(w, unauthFail)
		fmt.Fprintf(w, unauthJS)
	} else {
		fmt.Fprintf(w, "$global.auth = %s;\n", string(b))
	}
}

func (core *Core) v1Ping(w http.ResponseWriter, req *http.Request) {
	JSON(w, struct {
		OK string `json:"ok"`
	}{
		OK: "pong",
	})
}

func (core *Core) v1GetPublicKey(w http.ResponseWriter, req *http.Request) {
	pub := core.agent.key.PublicKey()
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s %s\n", pub.Type(), base64.StdEncoding.EncodeToString(pub.Marshal()))
}

func (core *Core) v1Status(w http.ResponseWriter, req *http.Request) {
	stat := struct {
		Version    string `json:"version,omitempty"`
		Name       string `json:"name"`
		APIVersion int    `json:"api_version"`
	}{
		Name:       os.Getenv("SHIELD_NAME"),
		APIVersion: APIVersion,
	}

	//TODO: Once this is in v2, lock behind a check for auth
	stat.Version = Version

	JSON(w, &stat)
}

func (core *Core) v1DetailedStatus(w http.ResponseWriter, req *http.Request) {
	pending, err := core.DB.GetAllTasks(&db.TaskFilter{ForStatus: db.PendingStatus})
	if err != nil {
		bail(w, err)
		return
	}
	running, err := core.DB.GetAllTasks(&db.TaskFilter{ForStatus: db.RunningStatus})
	if err != nil {
		bail(w, err)
		return
	}
	JSON(w, struct {
		PendingTasks []*db.Task `json:"pending_tasks"`
		RunningTasks []*db.Task `json:"running_tasks"`
	}{
		PendingTasks: pending,
		RunningTasks: running,
	})
}

type v1jobhealth struct {
	Name    string `json:"name"`
	LastRun int64  `json:"last_run"`
	NextRun int64  `json:"next_run"`
	Paused  bool   `json:"paused"`
	Status  string `json:"status"`
}

func (core *Core) v1JobsStatus(w http.ResponseWriter, req *http.Request) {
	jobs, err := core.DB.GetAllJobs(&db.JobFilter{})
	if err != nil {
		bail(w, err)
		return
	}

	health := make(map[string]v1jobhealth)
	for _, j := range jobs {
		var next, last int64
		if j.LastRun.Time().IsZero() {
			last = 0
		} else {
			last = j.LastRun.Time().Unix()
		}

		j.Reschedule() /* not really, just enough to get NextRun */
		if j.Paused || j.NextRun.IsZero() {
			next = 0
		} else {
			next = j.NextRun.Unix()
		}

		health[j.Name] = v1jobhealth{
			Name:    j.Name,
			Paused:  j.Paused,
			LastRun: last,
			NextRun: next,
			Status:  j.LastTaskStatus,
		}
	}

	JSON(w, health)
}

func (core *Core) v1GetArchives(w http.ResponseWriter, req *http.Request) {
	status := []string{}
	if s := paramValue(req, "status", ""); s != "" {
		status = append(status, s)
	}

	limit, err := strconv.Atoi(paramValue(req, "limit", "0"))
	if err != nil || limit < 0 {
		bailWithError(w, ClientErrorf("invalid limit supplied"))
		return
	}

	archives, err := core.DB.GetAllArchives(
		&db.ArchiveFilter{
			ForTarget:  paramValue(req, "target", ""),
			ForStore:   paramValue(req, "store", ""),
			Before:     paramDate(req, "before"),
			After:      paramDate(req, "after"),
			WithStatus: status,
			Limit:      limit,
		},
	)

	if err != nil {
		bail(w, err)
		return
	}

	JSON(w, archives)
}

func (core *Core) v1RestoreArchive(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		w.WriteHeader(400)
		return
	}

	var params struct {
		Target string `json:"target"`
		Owner  string `json:"owner"`
	}
	if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
		bailWithError(w, ClientErrorf("bad JSON payload: %s", err))
		return
	}

	if params.Owner == "" {
		params.Owner = "anon"
	}

	re := regexp.MustCompile(`^/v1/archive/([a-fA-F0-9-]+)/restore`)
	id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

	// find the archive
	archive, err := core.DB.GetArchive(id)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	var target *db.Target
	if params.Target == "" {
		target, err = core.DB.GetTarget(archive.TargetUUID)
	} else {
		target, err = core.DB.GetTarget(uuid.Parse(params.Target))
	}
	if err != nil {
		w.WriteHeader(501)
		return
	}

	task, err := core.DB.CreateRestoreTask(params.Owner, archive, target)
	if err != nil {
		bail(w, err)
		return
	}
	JSONLiteral(w, fmt.Sprintf(`{"ok":"scheduled","task_uuid":"%s"}`, task.UUID))
}

func (core *Core) v1GetArchive(w http.ResponseWriter, req *http.Request) {
	re := regexp.MustCompile(`^/v1/archive/([a-fA-F0-9-]+)`)
	id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

	archive, err := core.DB.GetArchive(id)
	if err != nil {
		bail(w, err)
		return
	}

	if archive == nil {
		w.WriteHeader(404)
		return
	}

	JSON(w, archive)
}

func (core *Core) v1UpdateArchive(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		w.WriteHeader(400)
		return
	}

	var params struct {
		Notes string `json:"notes"`
	}
	if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
		bailWithError(w, ClientErrorf("bad JSON payload: %s", err))
		return
	}

	if params.Notes == "" {
		w.WriteHeader(400)
		return
	}

	re := regexp.MustCompile(`^/v1/archive/([a-fA-F0-9-]+)`)
	id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

	archive, err := core.DB.GetArchive(id)
	if err != nil {
		bail(w, err)
		return
	}

	if archive == nil {
		w.WriteHeader(404)
		return
	}

	if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
		bail(w, err)
		return
	}

	if params.Notes != "" {
		archive.Notes = params.Notes
	}

	if err := core.DB.UpdateArchive(archive); err != nil {
		bail(w, err)
		return
	}

	JSONLiteral(w, fmt.Sprintf(`{"ok":"updated"}`))
}

func (core *Core) v1DeleteArchive(w http.ResponseWriter, req *http.Request) {
	re := regexp.MustCompile(`^/v1/archive/([a-fA-F0-9-]+)`)
	id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

	if deleted, err := core.DB.DeleteArchive(id); err != nil {
		bail(w, err)
		return
	} else if !deleted {
		w.WriteHeader(403)
	} else {
		JSONLiteral(w, fmt.Sprintf(`{"ok":"deleted"}`))
	}
}

func (core *Core) v1GetJobs(w http.ResponseWriter, req *http.Request) {
	jobs, err := core.DB.GetAllJobs(
		&db.JobFilter{
			SkipPaused:   paramEquals(req, "paused", "f"),
			SkipUnpaused: paramEquals(req, "paused", "t"),

			SearchName: paramValue(req, "name", ""),

			ForTarget:  paramValue(req, "target", ""),
			ForStore:   paramValue(req, "store", ""),
			ForPolicy:  paramValue(req, "retention", ""),
			ExactMatch: paramEquals(req, "exact", "t"),
		},
	)
	if err != nil {
		bail(w, err)
		return
	}

	JSON(w, jobs)
}

func (core *Core) v1CreateJob(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		w.WriteHeader(400)
		return
	}

	var params struct {
		Name    string `json:"name"`
		Summary string `json:"summary"`

		Store     string `json:"store"`
		Target    string `json:"target"`
		Schedule  string `json:"schedule"`
		Retention string `json:"retention"`

		Paused bool `json:"paused"`
	}
	if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
		bailWithError(w, ClientErrorf("bad JSON payload: %s", err))
		return
	}

	e := MissingParameters()
	e.Check("name", params.Name)
	e.Check("store", params.Store)
	e.Check("target", params.Target)
	e.Check("schedule", params.Schedule)
	e.Check("retention", params.Retention)
	if e.IsValid() {
		bailWithError(w, e)
		return
	}

	job, err := core.DB.CreateJob(&db.Job{
		Name:       params.Name,
		Summary:    params.Summary,
		Schedule:   params.Schedule,
		Paused:     params.Paused,
		TargetUUID: uuid.Parse(params.Target),
		StoreUUID:  uuid.Parse(params.Store),
		PolicyUUID: uuid.Parse(params.Retention),
	})
	if err != nil {
		bail(w, err)
		return
	}

	JSONLiteral(w, fmt.Sprintf(`{"ok":"created","uuid":"%s"}`, job.UUID.String()))
}

func (core *Core) v1PauseJob(w http.ResponseWriter, req *http.Request) {
	re := regexp.MustCompile(`^/v1/job/([a-fA-F0-9-]+)/pause`)
	id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

	found, err := core.DB.PauseJob(id)
	if !found {
		w.WriteHeader(404)
		return
	}
	if err != nil {
		bail(w, err)
		return
	}

	JSONLiteral(w, `{"ok":"paused"}`)
}

func (core *Core) v1UnpauseJob(w http.ResponseWriter, req *http.Request) {
	re := regexp.MustCompile(`^/v1/job/([a-fA-F0-9-]+)/unpause`)
	id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

	found, err := core.DB.UnpauseJob(id)
	if !found {
		w.WriteHeader(404)
		return
	}
	if err != nil {
		bail(w, err)
		return
	}

	JSONLiteral(w, `{"ok":"unpaused"}`)
}

func (core *Core) v1RunJob(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		w.WriteHeader(400)
		return
	}

	var params struct {
		Owner string `json:"owner"`
	}
	if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
		bailWithError(w, ClientErrorf("bad JSON payload: %s", err))
		return
	}

	if params.Owner == "" {
		params.Owner = "anon"
	}

	re := regexp.MustCompile(`^/v1/job/([a-fA-F0-9-]+)/run`)
	id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

	job, err := core.DB.GetJob(id)
	if err != nil {
		bail(w, err)
		return
	}

	task, err := core.DB.CreateBackupTask(params.Owner, job)
	if err != nil {
		bail(w, err)
		return
	}

	JSONLiteral(w, fmt.Sprintf(`{"ok":"scheduled","task_uuid":"%s"}`, task.UUID))
}

func (core *Core) v1GetJob(w http.ResponseWriter, req *http.Request) {
	re := regexp.MustCompile(`^/v1/job/([a-fA-F0-9-]+)`)
	id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

	job, err := core.DB.GetJob(id)
	if err != nil {
		bail(w, err)
		return
	}

	if job == nil {
		w.WriteHeader(404)
		return
	}

	JSON(w, job)
}

func (core *Core) v1UpdateJob(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		w.WriteHeader(400)
		return
	}

	var params struct {
		Name    string `json:"name"`
		Summary string `json:"summary"`

		Store     string `json:"store"`
		Target    string `json:"target"`
		Schedule  string `json:"schedule"`
		Retention string `json:"retention"`
	}
	if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
		bailWithError(w, ClientErrorf("bad JSON payload: %s", err))
		return
	}

	e := MissingParameters()
	e.Check("name", params.Name)
	e.Check("store", params.Store)
	e.Check("target", params.Target)
	e.Check("schedule", params.Schedule)
	e.Check("retention", params.Retention)
	if e.IsValid() {
		bailWithError(w, e)
		return
	}

	re := regexp.MustCompile(`^/v1/job/([a-fA-F0-9-]+)`)
	id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

	job, err := core.DB.GetJob(id)
	if err != nil {
		bail(w, err)
		return
	}

	job.Name = params.Name
	job.Summary = params.Summary
	job.Schedule = params.Schedule
	job.TargetUUID = uuid.Parse(params.Target)
	job.StoreUUID = uuid.Parse(params.Store)
	job.PolicyUUID = uuid.Parse(params.Retention)

	if err := core.DB.UpdateJob(job); err != nil {
		bail(w, err)
		return
	}

	JSONLiteral(w, `{"ok":"updated"}`)
}

func (core *Core) v1DeleteJob(w http.ResponseWriter, req *http.Request) {
	re := regexp.MustCompile(`^/v1/job/([a-fA-F0-9-]+)`)
	id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

	deleted, err := core.DB.DeleteJob(id)

	if err != nil {
		bail(w, err)
		return
	}
	if !deleted {
		w.WriteHeader(403)
		return
	}

	JSONLiteral(w, `{"ok":"deleted"}`)
}

func (core *Core) v1GetRetentionPolicies(w http.ResponseWriter, req *http.Request) {
	policies, err := core.DB.GetAllRetentionPolicies(
		&db.RetentionFilter{
			SkipUsed:   paramEquals(req, "unused", "t"),
			SkipUnused: paramEquals(req, "unused", "f"),
			SearchName: paramValue(req, "name", ""),
			ExactMatch: paramEquals(req, "exact", "t"),
		},
	)
	if err != nil {
		bail(w, err)
		return
	}

	JSON(w, policies)
}

func (core *Core) v1CreateRetentionPolicy(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		w.WriteHeader(400)
		return
	}

	var params struct {
		Name    string `json:"name"`
		Summary string `json:"summary"`
		Expires uint   `json:"expires"`
	}
	if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
		bailWithError(w, ClientErrorf("bad JSON payload: %s", err))
		return
	}

	e := MissingParameters()
	e.Check("name", params.Name)
	if params.Expires == 0 {
		e.Check("expires", "")
	}
	if e.IsValid() {
		bailWithError(w, e)
		return
	}

	v := InvalidParameters()
	v.Validate("expires", params.Expires, func(n string, v interface{}) error {
		if v.(uint) < 3600 {
			return fmt.Errorf("%d is less than 3600", v.(uint))
		}
		return nil
	})
	if v.IsValid() {
		bailWithError(w, v)
		return
	}

	policy, err := core.DB.CreateRetentionPolicy(&db.RetentionPolicy{
		Name:    params.Name,
		Summary: params.Summary,
		Expires: params.Expires,
	})
	if err != nil {
		bail(w, err)
		return
	}

	JSONLiteral(w, fmt.Sprintf(`{"ok":"created","uuid":"%s"}`, policy.UUID.String()))
}

func (core *Core) v1GetRetentionPolicy(w http.ResponseWriter, req *http.Request) {
	re := regexp.MustCompile("^/v1/retention/")
	id := uuid.Parse(re.ReplaceAllString(req.URL.Path, ""))

	policy, err := core.DB.GetRetentionPolicy(id)
	if err != nil {
		bail(w, err)
		return
	}

	if policy == nil {
		w.WriteHeader(404)
		return
	}

	JSON(w, policy)
}

func (core *Core) v1UpdateRetentionPolicy(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		w.WriteHeader(400)
		return
	}

	var params struct {
		Name    string `json:"name"`
		Summary string `json:"summary"`
		Expires uint   `json:"expires"`
	}
	if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
		bailWithError(w, ClientErrorf("bad JSON payload: %s", err))
		return
	}

	e := MissingParameters()
	e.Check("name", params.Name)
	if params.Expires == 0 {
		e.Check("expires", "")
	}
	if e.IsValid() {
		bailWithError(w, e)
		return
	}

	v := InvalidParameters()
	v.Validate("expires", params.Expires, func(n string, v interface{}) error {
		if v.(uint) < 3600 {
			return fmt.Errorf("%d is less than 3600", v.(uint))
		}
		return nil
	})
	if v.IsValid() {
		bailWithError(w, v)
		return
	}

	re := regexp.MustCompile("^/v1/retention/")
	id := uuid.Parse(re.ReplaceAllString(req.URL.Path, ""))

	policy, err := core.DB.GetRetentionPolicy(id)
	if err != nil {
		bail(w, err)
		return
	}

	policy.Name = params.Name
	policy.Summary = params.Summary
	policy.Expires = params.Expires
	if err := core.DB.UpdateRetentionPolicy(policy); err != nil {
		bail(w, err)
		return
	}

	JSONLiteral(w, fmt.Sprintf(`{"ok":"updated"}`))
}

func (core *Core) v1DeleteRetentionPolicy(w http.ResponseWriter, req *http.Request) {
	re := regexp.MustCompile("^/v1/retention/")
	id := uuid.Parse(re.ReplaceAllString(req.URL.Path, ""))
	deleted, err := core.DB.DeleteRetentionPolicy(id)

	if err != nil {
		bail(w, err)
		return
	}
	if !deleted {
		w.WriteHeader(403)
		return
	}

	JSONLiteral(w, fmt.Sprintf(`{"ok":"deleted"}`))
}

func (core *Core) v1GetStores(w http.ResponseWriter, req *http.Request) {
	stores, err := core.DB.GetAllStores(
		&db.StoreFilter{
			SkipUsed:   paramEquals(req, "unused", "t"),
			SkipUnused: paramEquals(req, "unused", "f"),
			SearchName: paramValue(req, "name", ""),
			ForPlugin:  paramValue(req, "plugin", ""),
			ExactMatch: paramEquals(req, "exact", "t"),
		},
	)
	if err != nil {
		bail(w, err)
		return
	}

	JSON(w, stores)
}

func (core *Core) v1CreateStore(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		w.WriteHeader(400)
		return
	}

	var params struct {
		Name     string `json:"name"`
		Summary  string `json:"summary"`
		Plugin   string `json:"plugin"`
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
		bailWithError(w, ClientErrorf("bad JSON payload: %s", err))
		return
	}

	e := MissingParameters()
	e.Check("name", params.Name)
	e.Check("plugin", params.Plugin)
	e.Check("endpoint", params.Endpoint)
	if e.IsValid() {
		bailWithError(w, e)
		return
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(params.Endpoint), &config); err != nil {
		bailWithError(w, e)
		return
	}

	store, err := core.DB.CreateStore(&db.Store{
		Name:    params.Name,
		Agent:   core.purgeAgent,
		Plugin:  params.Plugin,
		Config:  config,
		Summary: params.Summary,
	})
	if err != nil {
		bail(w, err)
		return
	}

	JSONLiteral(w, fmt.Sprintf(`{"ok":"created","uuid":"%s"}`, store.UUID.String()))
}

func (core *Core) v1GetStore(w http.ResponseWriter, req *http.Request) {
	re := regexp.MustCompile(`^/v1/store/([a-fA-F0-9-]+)`)
	id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

	store, err := core.DB.GetStore(id)
	if err != nil {
		bail(w, err)
		return
	}

	if store == nil {
		w.WriteHeader(404)
		return
	}

	JSON(w, store)
}

func (core *Core) v1UpdateStore(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		w.WriteHeader(400)
		return
	}

	var params struct {
		Name     string `json:"name"`
		Summary  string `json:"summary"`
		Plugin   string `json:"plugin"`
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
		bailWithError(w, ClientErrorf("bad JSON payload: %s", err))
		return
	}

	e := MissingParameters()
	e.Check("name", params.Name)
	e.Check("plugin", params.Plugin)
	e.Check("endpoint", params.Endpoint)
	if e.IsValid() {
		bailWithError(w, e)
		return
	}

	re := regexp.MustCompile("^/v1/store/")
	id := uuid.Parse(re.ReplaceAllString(req.URL.Path, ""))

	store, err := core.DB.GetStore(id)
	if err != nil {
		bail(w, err)
		return
	}
	if store == nil {
		w.WriteHeader(404)
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
		if err := json.Unmarshal([]byte(params.Endpoint), &store.Config); err != nil {
			bailWithError(w, e)
			return
		}
	}

	if err := core.DB.UpdateStore(store); err != nil {
		bail(w, err)
		return
	}

	JSONLiteral(w, fmt.Sprintf(`{"ok":"updated"}`))
}

func (core *Core) v1DeleteStore(w http.ResponseWriter, req *http.Request) {
	re := regexp.MustCompile("^/v1/store/")
	id := uuid.Parse(re.ReplaceAllString(req.URL.Path, ""))
	deleted, err := core.DB.DeleteStore(id)

	if err != nil {
		bail(w, err)
		return
	}
	if !deleted {
		w.WriteHeader(403)
		return
	}

	JSONLiteral(w, fmt.Sprintf(`{"ok":"deleted"}`))
}

func (core *Core) v1GetTargets(w http.ResponseWriter, req *http.Request) {
	targets, err := core.DB.GetAllTargets(
		&db.TargetFilter{
			SkipUsed:   paramEquals(req, "unused", "t"),
			SkipUnused: paramEquals(req, "unused", "f"),
			SearchName: paramValue(req, "name", ""),
			ForPlugin:  paramValue(req, "plugin", ""),
			ExactMatch: paramEquals(req, "exact", "t"),
		},
	)
	if err != nil {
		bail(w, err)
		return
	}

	JSON(w, targets)
}

func (core *Core) v1CreateTarget(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		w.WriteHeader(400)
		return
	}

	var params struct {
		Name     string `json:"name"`
		Summary  string `json:"summary"`
		Plugin   string `json:"plugin"`
		Endpoint string `json:"endpoint"`
		Agent    string `json:"agent"`
	}
	if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
		bailWithError(w, ClientErrorf("bad JSON payload: %s", err))
		return
	}

	e := MissingParameters()
	e.Check("name", params.Name)
	e.Check("plugin", params.Plugin)
	e.Check("endpoint", params.Endpoint)
	e.Check("agent", params.Agent)
	if e.IsValid() {
		bailWithError(w, e)
		return
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(params.Endpoint), &config); err != nil {
		bailWithError(w, e)
		return
	}

	t, err := core.DB.CreateTarget(&db.Target{
		Name: params.Name, Summary: params.Summary,
		Plugin: params.Plugin,
		Config: config,
		Agent:  params.Agent,
	})
	if err != nil {
		bail(w, err)
		return
	}

	JSONLiteral(w, fmt.Sprintf(`{"ok":"created","uuid":"%s"}`, t.UUID.String()))
}

func (core *Core) v1GetTarget(w http.ResponseWriter, req *http.Request) {
	re := regexp.MustCompile(`^/v1/target/([a-fA-F0-9-]+)`)
	id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

	target, err := core.DB.GetTarget(id)
	if err != nil {
		bail(w, err)
		return
	}

	if target == nil {
		w.WriteHeader(404)
		return
	}

	JSON(w, target)

}

func (core *Core) v1UpdateTarget(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		w.WriteHeader(400)
		return
	}

	var params struct {
		Name     string `json:"name"`
		Summary  string `json:"summary"`
		Plugin   string `json:"plugin"`
		Endpoint string `json:"endpoint"`
		Agent    string `json:"agent"`
	}
	if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
		bailWithError(w, ClientErrorf("bad JSON payload: %s", err))
		return
	}

	e := MissingParameters()
	e.Check("name", params.Name)
	e.Check("plugin", params.Plugin)
	e.Check("endpoint", params.Endpoint)
	e.Check("agent", params.Agent)
	if e.IsValid() {
		bailWithError(w, e)
		return
	}

	re := regexp.MustCompile("^/v1/target/")
	id := uuid.Parse(re.ReplaceAllString(req.URL.Path, ""))

	target, err := core.DB.GetTarget(id)
	if err != nil {
		bail(w, err)
		return
	}
	target.Name = params.Name
	target.Plugin = params.Plugin
	target.Agent = params.Agent
	if params.Endpoint != "" {
		if err := json.Unmarshal([]byte(params.Endpoint), &target.Config); err != nil {
			bailWithError(w, e)
			return
		}
	}
	if params.Summary != "" {
		target.Summary = params.Summary
	}
	if err := core.DB.UpdateTarget(target); err != nil {
		bail(w, err)
		return
	}

	JSONLiteral(w, fmt.Sprintf(`{"ok":"updated"}`))
}

func (core *Core) v1DeleteTarget(w http.ResponseWriter, req *http.Request) {
	re := regexp.MustCompile("^/v1/target/")
	id := uuid.Parse(re.ReplaceAllString(req.URL.Path, ""))
	deleted, err := core.DB.DeleteTarget(id)

	if err != nil {
		bail(w, err)
		return
	}
	if !deleted {
		w.WriteHeader(403)
		return
	}

	JSONLiteral(w, fmt.Sprintf(`{"ok":"deleted"}`))
}

func (core *Core) v1GetTasks(w http.ResponseWriter, req *http.Request) {
	limit, err := strconv.Atoi(paramValue(req, "limit", "0"))
	if err != nil {
		bailWithError(w, ClientErrorf("invalid limit supplied"))
		return
	}
	tasks, err := core.DB.GetAllTasks(
		&db.TaskFilter{
			SkipActive:   paramEquals(req, "active", "f"),
			SkipInactive: paramEquals(req, "active", "t"),
			ForStatus:    paramValue(req, "status", ""),
			Limit:        limit,
		},
	)
	if err != nil {
		bail(w, err)
		return
	}

	JSON(w, tasks)
}

func (core *Core) v1GetTask(w http.ResponseWriter, req *http.Request) {
	re := regexp.MustCompile(`^/v1/task/([a-fA-F0-9-]+)`)
	id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

	task, err := core.DB.GetTask(id)
	if err != nil {
		bail(w, err)
		return
	}

	if task == nil {
		w.WriteHeader(404)
		return
	}

	JSON(w, task)
}

func (core *Core) v1CancelTask(w http.ResponseWriter, req *http.Request) {
	re := regexp.MustCompile(`^/v1/task/([a-fA-F0-9-]+)`)
	id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

	err := core.DB.CancelTask(id, time.Now())

	if err != nil {
		bail(w, err)
		return
	}

	JSONLiteral(w, fmt.Sprintf(`{"ok":"canceled"}`))
}

/*

  GET /v2/systems

  Retrieves a list of all protected systems, their job configurations,
  recent archive metadata, and failed task metadata.  This endpoint is
  the be-all, end-all of interacting with targets + jobs under the new
  SHIELD UI.

  Response:
  [
    {
      "uuid"  : "93815474-126f-4934-aead-aaee29a34f3c",
      "name"  : "Important Database",
      "notes" : "This is the most important data we have",
      "ok"    : 1,

      "jobs": [
        {
          "schedule" : "daily 7am",
          "from"     : "Postgres",
          "to"       : "S3",
          "keep"     : { "n": 9, "days": 9 },
          "ok"       : true
        },
        {
          "schedule" : "weekly 9:30am",
          "from"     : "Postgres",
          "to"       : "S3",
          "keep"     : { "n": 12, "days": 90 },
          "ok"       : true
        }
      ]
    }
  ]
*/

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
