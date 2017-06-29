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

func (core *Core) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch {
	case match(req, `GET /v1/ping`):
		core.v1Ping(w, req)

	case match(req, `GET /v1/meta/pubkey`):
		core.v1GetPublicKey(w, req)

	case match(req, `GET /v1/status`):
		core.v1Status(w, req)
	case match(req, `GET /v1/status/internal`):
		core.v1DetailedStatus(w, req)
	case match(req, `GET /v1/status/jobs`):
		core.v1JobsStatus(w, req)

	case match(req, `GET /v1/archives`):
		core.v1GetArchives(w, req)
	case match(req, `POST /v1/archive/[a-fA-F0-9-]+/restore`):
		core.v1RestoreArchive(w, req)
	case match(req, `GET /v1/archive/[a-fA-F0-9-]+`):
		core.v1GetArchive(w, req)
	case match(req, `PUT /v1/archive/[a-fA-F0-9-]+`):
		core.v1UpdateArchive(w, req)
	case match(req, `DELETE /v1/archive/[a-fA-F0-9-]+`):
		core.v1DeleteArchive(w, req)

	case match(req, `GET /v1/jobs`):
		core.v1GetJobs(w, req)
	case match(req, `POST /v1/jobs`):
		core.v1CreateJob(w, req)
	case match(req, `POST /v1/job/[a-fA-F0-9-]+/pause`):
		core.v1PauseJob(w, req)
	case match(req, `POST /v1/job/[a-fA-F0-9-]+/unpause`):
		core.v1UnpauseJob(w, req)
	case match(req, `POST /v1/job/[a-fA-F0-9-]+/run`):
		core.v1RunJob(w, req)
	case match(req, `GET /v1/job/[a-fA-F0-9-]+`):
		core.v1GetJob(w, req)
	case match(req, `PUT /v1/job/[a-fA-F0-9-]+`):
		core.v1UpdateJob(w, req)
	case match(req, `DELETE /v1/job/[a-fA-F0-9-]+`):
		core.v1DeleteJob(w, req)

	case match(req, `GET /v1/retention`):
		core.v1GetRetentionPolicies(w, req)
	case match(req, `POST /v1/retention`):
		core.v1CreateRetentionPolicy(w, req)
	case match(req, `GET /v1/retention/[a-fA-F0-9-]+`):
		core.v1GetRetentionPolicy(w, req)
	case match(req, `PUT /v1/retention/[a-fA-F0-9-]+`):
		core.v1UpdateRetentionPolicy(w, req)
	case match(req, `DELETE /v1/retention/[a-fA-F0-9-]+`):
		core.v1DeleteRetentionPolicy(w, req)

	case match(req, `GET /v1/stores`):
		core.v1GetStores(w, req)
	case match(req, `POST /v1/stores`):
		core.v1CreateStore(w, req)
	case match(req, `GET /v1/store/[a-fA-F0-9-]+`):
		core.v1GetStore(w, req)
	case match(req, `PUT /v1/store/[a-fA-F0-9-]+`):
		core.v1UpdateStore(w, req)
	case match(req, `DELETE /v1/store/[a-fA-F0-9-]+`):
		core.v1DeleteStore(w, req)

	case match(req, `GET /v1/targets`):
		core.v1GetTargets(w, req)
	case match(req, `POST /v1/targets`):
		core.v1CreateTarget(w, req)
	case match(req, `GET /v1/target/[a-fA-F0-9-]+`):
		core.v1GetTarget(w, req)
	case match(req, `PUT /v1/target/[a-fA-F0-9-]+`):
		core.v1UpdateTarget(w, req)
	case match(req, `DELETE /v1/target/[a-fA-F0-9-]+`):
		core.v1DeleteTarget(w, req)

	case match(req, `GET /v1/tasks`):
		core.v1GetTasks(w, req)
	case match(req, `GET /v1/task/[a-fA-F0-9-]+`):
		core.v1GetTask(w, req)
	case match(req, `DELETE /v1/task/[a-fA-F0-9-]+`):
		core.v1CancelTask(w, req)

	case match(req, `GET /v2/health`):
		core.v2GetHealth(w, req)

	case match(req, `GET /v2/agents`):
		core.v2GetAgents(w, req)
	case match(req, `POST /v2/agents`):
		core.v2PostAgents(w, req)

	case match(req, `GET /v2/systems`):
		core.v2GetSystems(w, req)
	case match(req, `POST /v2/systems`):
		core.v2PostSystem(w, req)
	case match(req, `GET /v2/systems/:uuid`):
		core.v2GetSystem(w, req)
	case match(req, `PUT /v2/systems/:uuid`):
		core.v2PutSystem(w, req)
	case match(req, `PATCH /v2/systems/:uuid`):
		core.v2PatchSystem(w, req)
	case match(req, `DELETE /v2/systems/:uuid`):
		core.v2DeleteSystem(w, req)

	default:
		w.WriteHeader(501)
	}
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

func invalidlimit(limit string) bool {
	if limit != "" {
		limint, err := strconv.Atoi(limit)
		if err != nil || limint <= 0 {
			return true
		}
	}
	return false
}

func (core *Core) v1Ping(w http.ResponseWriter, req *http.Request) {
	JSONLiteral(w, `{"ok":"pong"}`)
}

func (core *Core) v1GetPublicKey(w http.ResponseWriter, req *http.Request) {
	pub := core.agent.key.PublicKey()
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s %s\n", pub.Type(), base64.StdEncoding.EncodeToString(pub.Marshal()))
}

func (core *Core) v1Status(w http.ResponseWriter, req *http.Request) {
	JSON(w, struct {
		Version string `json:"version"`
		Name    string `json:"name"`
	}{
		Version: Version,
		Name:    os.Getenv("SHIELD_NAME"),
	})
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

	limit := paramValue(req, "limit", "")
	if invalidlimit(limit) {
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

	if err := core.DB.AnnotateArchive(id, params.Notes); err != nil {
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

			ForTarget:    paramValue(req, "target", ""),
			ForStore:     paramValue(req, "store", ""),
			ForRetention: paramValue(req, "retention", ""),
			ExactMatch:   paramEquals(req, "exact", "t"),
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

	id, err := core.DB.CreateJob(params.Target, params.Store, params.Schedule, params.Retention, params.Paused)
	if err != nil {
		bail(w, err)
		return
	}

	err = core.DB.AnnotateJob(id, params.Name, params.Summary)
	if err != nil {
		bail(w, err)
		return
	}
	JSONLiteral(w, fmt.Sprintf(`{"ok":"created","uuid":"%s"}`, id.String()))
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

	if err := core.DB.UpdateJob(id, params.Target, params.Store, params.Schedule, params.Retention); err != nil {
		bail(w, err)
		return
	}
	if err := core.DB.AnnotateJob(id, params.Name, params.Summary); err != nil {
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

	id, err := core.DB.CreateRetentionPolicy(params.Expires)
	if err != nil {
		bail(w, err)
		return
	}

	if err := core.DB.AnnotateRetentionPolicy(id, params.Name, params.Summary); err != nil {
		bail(w, err)
		return
	}

	JSONLiteral(w, fmt.Sprintf(`{"ok":"created","uuid":"%s"}`, id.String()))
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
	if err := core.DB.UpdateRetentionPolicy(id, params.Expires); err != nil {
		bail(w, err)
		return
	}
	if err := core.DB.AnnotateRetentionPolicy(id, params.Name, params.Summary); err != nil {
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

	id, err := core.DB.CreateStore(params.Plugin, params.Endpoint)
	if err != nil {
		bail(w, err)
		return
	}

	if err := core.DB.AnnotateStore(id, params.Name, params.Summary); err != nil {
		bail(w, err)
		return
	}

	JSONLiteral(w, fmt.Sprintf(`{"ok":"created","uuid":"%s"}`, id.String()))
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
	if err := core.DB.UpdateStore(id, params.Plugin, params.Endpoint); err != nil {
		bail(w, err)
		return
	}
	if err := core.DB.AnnotateStore(id, params.Name, params.Summary); err != nil {
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

	id, err := core.DB.CreateTarget(params.Plugin, params.Endpoint, params.Agent)
	if err != nil {
		bail(w, err)
		return
	}
	if err := core.DB.AnnotateTarget(id, params.Name, params.Summary); err != nil {
		bail(w, err)
		return
	}

	JSONLiteral(w, fmt.Sprintf(`{"ok":"created","uuid":"%s"}`, id.String()))
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
	if err := core.DB.UpdateTarget(id, params.Plugin, params.Endpoint, params.Agent); err != nil {
		bail(w, err)
		return
	}
	if err := core.DB.AnnotateTarget(id, params.Name, params.Summary); err != nil {
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
	limit := paramValue(req, "limit", "")
	if invalidlimit(limit) {
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

  GET /v2/health

  Returns health information about the SHIELD core,
  connected storage accounts, and general metrics.

  {
    "shield": {
      "version" : "6.7.2",
      "ip"      : "10.0.0.5",
      "fqdn"    : "shield.example.com",
      "env"     : "PRODUCTION",
      "color"   : ""
    },
    "health": {
      "api_ok"     : true,
      "storage_ok" : true,
      "jobs_ok"    : true
    },
    "storage": [
      { "name": "s3", "healthy": true },
      { "name": "fs", "healthy": true } ],
    "jobs": [
      { "target": "BOSH DB", "job": "daily",  "healthy": true },
      { "target": "BOSH DB", "job": "weekly", "healthy": true } ],
    "stats": {
      "jobs"    : 8,
      "systems" : 7,
      "archives": 124,
      "storage" : 243567112,
      "daily"   : 12345000
    }
  }

*/
type v2StorageHealth struct {
	UUID    uuid.UUID `json:"uuid"`
	Name    string    `json:"name"`
	Healthy bool      `json:"healthy"`
}
type v2JobHealth struct {
	UUID    uuid.UUID `json:"uuid"`
	Target  string    `json:"target"`
	Job     string    `json:"job"`
	Healthy bool      `json:"healthy"`
}
type v2Health struct {
	SHIELD struct {
		Version string `json:"version"`
		IP      string `json:"ip"`
		FQDN    string `json:"fqdn"`
		Env     string `json:"env"`
		Color   string `json:"color"`
	} `json:"shield"`
	Health struct {
		API     bool `json:"api_ok"`
		Storage bool `json:"storage_ok"`
		Jobs    bool `json:"jobs_ok"`
	} `json:"health"`

	Storage []v2StorageHealth `json:"storage"`
	Jobs    []v2JobHealth     `json:"jobs"`

	Stats struct {
		Jobs     int `json:"jobs"`
		Systems  int `json:"systems"`
		Archives int `json:"archives"`
		Storage  int `json:"storage"`
		Daily    int `json:"daily"`
	} `json:"stats"`
}

func (core *Core) v2GetHealth(w http.ResponseWriter, req *http.Request) {
	var health v2Health

	health.Health.API = true
	health.SHIELD.Version = Version
	health.SHIELD.Env = os.Getenv("SHIELD_NAME")
	health.SHIELD.IP = core.ip
	health.SHIELD.FQDN = core.fqdn

	health.Health.Storage = true
	stores, err := core.DB.GetAllStores(nil)
	if err != nil {
		bail(w, err)
		return
	}
	health.Storage = make([]v2StorageHealth, len(stores))
	for i, store := range stores {
		health.Storage[i].UUID = store.UUID
		health.Storage[i].Name = store.Name
		health.Storage[i].Healthy = true // FIXME
		if !health.Storage[i].Healthy {
			health.Health.Storage = false
		}
	}

	health.Health.Jobs = true
	jobs, err := core.DB.GetAllJobs(nil)
	if err != nil {
		bail(w, err)
		return
	}
	health.Jobs = make([]v2JobHealth, len(jobs))
	for i, job := range jobs {
		health.Jobs[i].UUID = job.UUID
		health.Jobs[i].Target = job.TargetName
		health.Jobs[i].Job = job.Name
		health.Jobs[i].Healthy = job.Healthy()

		if !health.Jobs[i].Healthy {
			health.Health.Jobs = false
		}
	}
	health.Stats.Jobs = len(jobs)

	if health.Stats.Systems, err = core.DB.CountTargets(nil); err != nil {
		bail(w, err)
		return
	}

	if health.Stats.Archives, err = core.DB.CountArchives(nil); err != nil {
		bail(w, err)
		return
	}

	if health.Stats.Storage, err = core.DB.ArchiveStorageFootprint(nil); err != nil {
		bail(w, err)
		return
	}

	health.Stats.Daily = 0 // FIXME

	JSON(w, health)
}

func (core *Core) v2GetAgents(w http.ResponseWriter, req *http.Request) {
	agents, err := core.DB.GetAllAgents(nil)
	if err != nil {
		bail(w, err)
		return
	}

	r := struct {
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

		r.Problems[id] = pp
	}
	JSON(w, r)
}

/*

  POST /v2/agents

  Initiate agent registration.  The client must supply a POST body in
  the form of:

  {
    "name" : "some-identifier",
    "port" : "5444"
  }

  The SHIELD core will then schedule a "pingback", connecting to the
  agent using its remote peer address (from the registration HTTP
  conversation) and the given port.  This pingback occurs via the SSH
  protocol, with an op type of "ping".  The agent must respond with
  the same _name_ that it sent in the registration.

  This exchange allows the core to validate registration requests,
  using a weak form of authentication.
*/
func (core *Core) v2PostAgents(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		w.WriteHeader(400)
		return
	}

	var params struct {
		Name string `json:"name"`
		Port int    `json:"port"`
	}

	if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
		bailWithError(w, ClientErrorf("bad JSON payload: %s", err))
		return
	}

	peer := regexp.MustCompile(`:\d+$`).ReplaceAllString(req.RemoteAddr, "")
	if peer == "" {
		bailWithError(w, ClientErrorf("unable to determine remote peer address from RemoteAddr '%s'", req.RemoteAddr))
		return
	}

	if params.Name == "" {
		bailWithError(w, ClientErrorf("no `name' provided with pre-registration request"))
		return
	}
	if params.Port == 0 {
		bailWithError(w, ClientErrorf("no `port' provided with pre-registration request"))
		return
	}

	err := core.DB.PreRegisterAgent(peer, params.Name, params.Port)
	if err != nil {
		bail(w, err)
		return
	}

	JSONLiteral(w, `{"ok":"pre-registered"}`)
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
		dst.Jobs[j].From = job.TargetPlugin
		dst.Jobs[j].To = job.StorePlugin
		dst.Jobs[j].OK = job.Healthy()
		dst.Jobs[j].Store.UUID = job.StoreUUID
		dst.Jobs[j].Store.Name = job.StoreName
		dst.Jobs[j].Store.Summary = job.StoreSummary
		dst.Jobs[j].Retention.UUID = job.RetentionUUID
		dst.Jobs[j].Retention.Name = job.RetentionName
		dst.Jobs[j].Retention.Summary = job.RetentionSummary

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

func (core *Core) v2GetSystems(w http.ResponseWriter, req *http.Request) {
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

	systems := make([]v2System, len(targets))
	for i, target := range targets {
		err := core.v2copyTarget(&systems[i], target)
		if err != nil {
			bail(w, err)
			return
		}
	}

	JSON(w, systems)
	return
}

func (core *Core) v2GetSystem(w http.ResponseWriter, req *http.Request) {
	re := regexp.MustCompile(`/([a-fA-F0-9-]+)$`)
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

	var system v2System
	err = core.v2copyTarget(&system, target)
	if err != nil {
		bail(w, err)
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
		bail(w, err)
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
		bail(w, err)
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

	JSON(w, system)
	return
}

func (core *Core) v2PostSystem(w http.ResponseWriter, req *http.Request) {
}

func (core *Core) v2PutSystem(w http.ResponseWriter, req *http.Request) {
}

type v2PatchAnnotation struct {
	Type        string `json:"type"`
	UUID        string `json:"uuid"`
	Disposition string `json:"disposition"`
	Notes       string `json:"notes"`
	Clear       string `json:"clear"`
}

func (core *Core) v2PatchSystem(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		w.WriteHeader(400)
		return
	}

	var params struct {
		Annotations []v2PatchAnnotation `json:"annotations"`
	}

	if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
		bailWithError(w, ClientErrorf("bad JSON payload: %s", err))
		return
	}

	re := regexp.MustCompile(`^/v2/systems/([a-fA-F0-9-]+)`)
	id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

	target, err := core.DB.GetTarget(id)
	if err != nil {
		bail(w, err)
		return
	}

	for _, ann := range params.Annotations {
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
				bail(w, err)
				return
			}

		case "archive":
			err = core.DB.AnnotateTargetArchive(
				target.UUID,
				ann.UUID,
				ann.Notes,
			)
			if err != nil {
				bail(w, err)
				return
			}

		default:
			bailWithError(w, ClientErrorf("invalid system annotation type '%s'", ann.Type))
			return
		}
	}

	_ = core.DB.MarkTasksIrrelevant()

	JSONLiteral(w, `{"ok":"annotated"}`)
}

func (core *Core) v2DeleteSystem(w http.ResponseWriter, req *http.Request) {
}
