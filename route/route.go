package route

import (
	"net/http"

	"github.com/starkandwayne/goutils/log"
)

type Handler func(r *Request)

type route struct {
	matcher matcher
	handler Handler
}

type Router struct {
	routes []route
}

func (r *Router) Dispatch(match string, handler Handler) {
	r.routes = append(r.routes, route{
		matcher: newMatch(match),
		handler: handler,
	})
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	request := &Request{
		Req:  req,
		w:    w,
		done: false,
	}

	for _, rt := range r.routes {
		if args, ok := rt.matcher(req); ok {
			w.Header().Set("Content-Type", "application/json")

			request.Args = args
			rt.handler(request)
			if !request.done {
				log.Errorf("%s handler bug: failed to call either OK() or Fail()", request)
				request.Fail(Oops(nil, "an unknown error has occurred"))
			}
			return
		}
	}

	request.Fail(NotFound(nil, "SHIELD API endpoint `%s' not found", request))
}
