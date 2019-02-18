package core

import (
	"github.com/starkandwayne/shield/route"
)

func (c *Core) v1API() *route.Router {
	r := &route.Router{
		Debug: c.Config.Debug,
	}

	r.Dispatch("GET /v1/meta/pubkey", func(r *route.Request) {
		for _, fc := range c.Config.Fabrics {
			if fc.Name == "legacy" {
				r.Respond(200, "text/plain", "%s\n", fc.legacy.pub)
				return
			}
		}
		r.Respond(404, "text/plain", "# no legacy fabric(s) configured!\n")
	})

	return r
}
