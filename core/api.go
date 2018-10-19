package core

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/jhunt/go-log"

	"github.com/starkandwayne/shield/db"
	"github.com/starkandwayne/shield/route"
	"github.com/starkandwayne/shield/util"
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
				IP:        util.RemoteIP(req),
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

	case match(req, `GET /v1/meta/pubkey`):
		core.v1GetPublicKey(w, req)
		return
	}

	w.WriteHeader(501)
}

func match(req *http.Request, pattern string) bool {
	matched, _ := regexp.MatchString(
		fmt.Sprintf("^%s$", pattern),
		fmt.Sprintf("%s %s", req.Method, req.URL.Path))
	return matched
}

func (core *Core) initJS(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(200)

	fmt.Fprintf(w, "// init.js\n")
	fmt.Fprintf(w, "var $global = {}\n")

	const unauthFail = "// failed to determine user authentication state...\n"
	const unauthJS = "$global.auth = {\"unauthenticated\":true};\n"

	user, err := core.DB.GetUserForSession(route.SessionID(req))
	if err != nil {
		log.Errorf("init.js: failed to get user from session: %s", err)
		fmt.Fprintf(w, unauthFail)
		fmt.Fprintf(w, unauthJS)
		return
	}

	b, err := json.Marshal(core.checkInfo(user != nil))
	if err != nil {
		log.Errorf("init.js: failed to marshal SHIELD core info into JSON: %s", err)
		fmt.Fprintf(w, "// failed to retrieve SHIELD core info...\n")
		fmt.Fprintf(w, "$global.shield = {};\n")
	} else {
		fmt.Fprintf(w, "$global.shield = %s;\n", string(b))
	}

	if user == nil {
		fmt.Fprintf(w, unauthJS)
		return
	}

	id, err := core.checkAuth(user)
	if err != nil {
		log.Errorf("init.js: failed to obtain tenancy info about user: %s", err)
		fmt.Fprintf(w, unauthFail)
		fmt.Fprintf(w, unauthJS)
		return
	}
	b, err = json.Marshal(id)
	if err != nil {
		log.Errorf("init.js: failed to marshal auth id data into JSON: %s", err)
		fmt.Fprintf(w, unauthFail)
		fmt.Fprintf(w, unauthJS)
	} else {
		fmt.Fprintf(w, "$global.auth = %s;\n", string(b))
	}
}

func (core *Core) v1GetPublicKey(w http.ResponseWriter, req *http.Request) {
	pub := core.agent.key.PublicKey()
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s %s\n", pub.Type(), base64.StdEncoding.EncodeToString(pub.Marshal()))
}
