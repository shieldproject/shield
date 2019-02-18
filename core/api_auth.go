package core

import (
	"fmt"

	"github.com/jhunt/go-log"

	"github.com/starkandwayne/shield/db"
	"github.com/starkandwayne/shield/route"
)

func (c *Core) authAPI() *route.Router {
	r := &route.Router{
		Debug: c.Config.Debug,
	}

	r.Dispatch("GET /auth/:provider/:action", func(r *route.Request) {
		provider, ok := c.providers[r.Args[1]]
		if !ok {
			r.Respond(404, "text/plain", "Unrecognized authentication provider %s\n", r.Args[1])
			return
		}

		switch r.Args[2] {
		case "redir":
			via := "web"
			if cookie, err := r.Req.Cookie("via"); err == nil {
				via = cookie.Value
			}
			log.Debugf("handling redirection for authentication provider flow; via='%s'", via)

			user := provider.HandleRedirect(r)
			if user == nil {
				r.Respond(500, "text/plain", "The authentication process broke down\n")
				return
			}

			session, err := c.db.CreateSession(&db.Session{
				UserUUID:  user.UUID,
				IP:        r.RemoteIP(),
				UserAgent: r.UserAgent(),
			})
			if err != nil {
				log.Errorf("failed to create a session for user %s@%s: %s", user.Account, user.Backend, err)
				r.Redirect(302, "/")
				return

			} else if via == "cli" {
				r.Redirect(302, fmt.Sprintf("/#!/cliauth:s:%s", session.UUID))
				return

			} else {
				r.SetSession(session.UUID)
				r.Redirect(302, "/")
				return
			}

		case "web", "cli":
			r.SetCookie("via", r.Args[2], "/auth")
			provider.Initiate(r)
		}
	})

	return r
}
