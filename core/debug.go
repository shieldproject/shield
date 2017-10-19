package core

import (
	"github.com/starkandwayne/shield/route"
)

func (core *Core) dispatchDebug(r *route.Router) {
	r.Dispatch("GET /v2/debug/sessioning", func(r *route.Request) {
		if core.IsNotAuthenticated(r) {
			return
		}
		r.OK(nil)
	})

	r.Dispatch("GET /v2/debug/200", func(r *route.Request) {
		r.OK(struct {
			Dog string `json:"dog"`
		}{
			Dog: "everything is fine",
		})
	})

	r.Dispatch("GET /v2/debug/401", func(r *route.Request) {
		r.Fail(route.Errorf(401, nil, "Please log in to receive 401s"))
	})

	r.Dispatch("GET /v2/debug/403", func(r *route.Request) {
		r.Fail(route.Errorf(403, nil, "I forbid you from making further requests for 403s"))
	})

	r.Dispatch("GET /v2/debug/500", func(r *route.Request) {
		r.Fail(route.Oops(nil, "An unknown error occurred when retrieving 500 status code"))
	})
}
