package core

import (
	"github.com/shieldproject/shield/route"
)

func (c *Core) v1API() *route.Router {
	r := &route.Router{
		Debug: c.Config.Debug,
	}

	r.Dispatch("GET /v1/meta/pubkey", func(r *route.Request) {
		if !c.Config.LegacyAgents.Enabled {
			r.Respond(403, "text/plain", "# legacy agent communication disabled.\n")
		}
		r.Respond(200, "text/plain", "%s\n", c.Config.LegacyAgents.pub)
	})

	return r
}
