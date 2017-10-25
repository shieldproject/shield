package core

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/jhunt/go-log"

	"github.com/starkandwayne/shield/db"
)

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
			session := &db.Session{
				UserUUID:  user.UUID,
				IP:        req.RemoteAddr,
				UserAgent: req.UserAgent(),
			}

			session, err := core.createSession(session)
			if err != nil {
				log.Errorf("failed to create a session for user %s@%s: %s", user.Account, user.Backend, err)
				w.Header().Set("Location", "/")
			} else if via == "cli" {
				w.Header().Set("Location", fmt.Sprintf("/#!/cliauth:s:%s", session.UUID.String()))
			} else {
				w.Header().Set("Location", "/")
				http.SetCookie(w, SessionCookie(session.UUID.String(), true))
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

	case match(req, `GET /v1/status/internal`):
		core.v1DetailedStatus(w, req)
		return

	case match(req, `GET /v1/status/jobs`):
		core.v1JobsStatus(w, req)
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
