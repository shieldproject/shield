package route

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/starkandwayne/goutils/log"
)

type Request struct {
	Req  *http.Request
	Args []string

	w    http.ResponseWriter
	done bool
}

func (r *Request) String() string {
	return fmt.Sprintf("%s %s", r.Req.Method, r.Req.URL.Path)
}

func (r *Request) Success(msg string, args ...interface{}) {
	r.OK(struct {
		Ok string `json:"ok"`
	}{Ok: fmt.Sprintf(msg, args...)})
}

func (r *Request) OK(resp interface{}) {
	if r.done {
		log.Errorf("%s handler bug: called OK() a second time, or after having called Fail()", r)
		return
	}

	r.w.Header().Set("Content-Type", "application/json")

	b, err := json.Marshal(resp)
	if err != nil {
		log.Errorf("%s errored, trying to marshal a JSON error response: %s", r, err)
		r.Fail(Oops(err, "an unknown error has occurred"))
		return
	}

	log.Debugf("%s responding with HTTP 200, payload [%s]", r, string(b))
	r.w.WriteHeader(200)
	fmt.Fprintf(r.w, "%s\n", string(b))
	r.done = true
}

func (r *Request) Fail(e Error) {
	if r.done {
		log.Errorf("%s handler bug: called Fail() a second time, or after having called OK()", r)
		return
	}

	if e.e != nil {
		log.Errorf("%s errored: %s", r, e.e)
	}
	r.w.Header().Set("Content-Type", "application/json")

	b, err := json.Marshal(e)
	if err != nil {
		log.Errorf("%s %s errored again, trying to marshal a JSON error response: %s", err)
		r.Fail(Oops(err, "an unknown error has occurred"))
		return
	}

	log.Debugf("%s responding with HTTP %d, payload [%s]", r, e.code, string(b))
	r.w.WriteHeader(e.code)
	fmt.Fprintf(r.w, "%s\n", string(b))
	r.done = true
}

func (r *Request) Payload(v interface{}) bool {
	if r.Req.Body == nil {
		r.Fail(Bad(nil, "no JSON input payload present in request"))
		return false
	}

	if err := json.NewDecoder(r.Req.Body).Decode(v); err != nil && err != io.EOF {
		r.Fail(Bad(err, "invalid JSON input payload present in request"))
		return false
	}

	return true
}

func (r *Request) Param(name, def string) string {
	v, set := r.Req.URL.Query()[name]
	if set {
		return v[0]
	}
	return def
}

func (r *Request) ParamIs(name, want string) bool {
	v, set := r.Req.URL.Query()[name]
	return set && v[0] == want
}
