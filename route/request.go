package route

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/jhunt/go-log"
)

const (
	SessionHeaderKey = "X-Shield-Session"
	SessionCookieKey = "shield7"
)

type Request struct {
	Req  *http.Request
	Args []string

	w     http.ResponseWriter
	debug bool
	bt    []string
}

//NewRequest initializes and returns a new request object. Setting debug to
// true will cause errors to be logged.
func NewRequest(w http.ResponseWriter, r *http.Request, debug bool) *Request {
	return &Request{
		Req:   r,
		w:     w,
		debug: debug,
	}
}

func (r *Request) String() string {
	return fmt.Sprintf("%s %s", r.Req.Method, r.Req.URL.Path)
}

func (r *Request) RemoteIP() string {
	ip := ""
	if xff := r.Req.Header.Get("X-Forwarded-For"); xff != "" {
		ip = regexp.MustCompile("[, ].*$").ReplaceAllString(xff, "")
	}
	if ip == "" {
		return r.Req.RemoteAddr
	}
	return ip
}

func (r *Request) UserAgent() string {
	return r.Req.UserAgent()
}

func (r *Request) Done() bool {
	return len(r.bt) > 0
}

func (r *Request) respond(code int, fn, typ, msg string) {
	/* have we already responded for this request? */
	if r.Done() {
		log.Errorf("%s handler bug: called %s() having already called [%v]", r, fn, r.bt)
		return
	}

	/* respond ... */
	r.w.Header().Set("Content-Type", typ)
	r.w.WriteHeader(code)
	fmt.Fprintf(r.w, "%s\n", msg)

	/* track that OK() or Fail() called us... */
	r.bt = append(r.bt, fn)
}

func (r *Request) Respond(code int, typ, msg string, args ...interface{}) {
	r.respond(code, "Respond", typ, fmt.Sprintf(msg, args...))
}

func (r *Request) Redirect(code int, to string) {
	/* have we already responded for this request? */
	if r.Done() {
		log.Errorf("Redirect handler bug: called %s() having already called [%v]", r, r.bt)
		return
	}

	/* respond ... */
	r.w.Header().Set("Location", to)
	r.w.WriteHeader(code)

	/* track that we finished via Redirect() */
	r.bt = append(r.bt, "Redirect")
}

func (r *Request) Success(msg string, args ...interface{}) {
	r.OK(struct {
		Ok string `json:"ok"`
	}{Ok: fmt.Sprintf(msg, args...)})
}

func (r *Request) OK(resp interface{}) {
	b, err := json.Marshal(resp)
	if err != nil {
		log.Errorf("%s errored, trying to marshal a JSON error response: %s", r, err)
		r.Fail(Oops(err, "an unknown error has occurred"))
		return
	}

	r.respond(200, "OK", "application/json", string(b))
}

func (r *Request) Fail(e Error) {
	if e.e != nil {
		log.Errorf("%s errored: %s", r, e.e)
	}
	if r.debug {
		e.ProvideDiagnostic()
	}

	b, err := json.Marshal(e)
	if err != nil {
		log.Errorf("%s %s errored again, trying to marshal a JSON error response: %s", err)
		r.Fail(Oops(err, "an unknown error has occurred"))
		return
	}

	r.respond(e.code, "Fail", "application/json", string(b))
}

//Payload unmarshals the JSON body of this request into the given interface.
// Returns true if successful and false otherwise.
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

func (r *Request) ParamDate(name string) *time.Time {
	v, set := r.Req.URL.Query()[name]
	if !set {
		return nil
	}

	t, err := time.Parse("20060102", v[0])
	if err != nil {
		return nil
	}
	return &t
}

// ParamDuration parses a duration string, example: "1m30s"
// that will be 1 minute and 30 seconds
func (r *Request) ParamDuration(name string) *time.Duration {
	v, set := r.Req.URL.Query()[name]
	if !set {
		return nil
	}

	d, err := time.ParseDuration(v[0])
	if err != nil {
		return nil
	}
	return &d
}

func (r *Request) ParamIs(name, want string) bool {
	v, set := r.Req.URL.Query()[name]
	return set && v[0] == want
}

func (r *Request) Missing(params ...string) bool {
	e := Error{code: 400}

	for len(params) > 1 {
		if params[1] == "" {
			e.Missing = append(e.Missing, params[0])
		}
		params = params[2:]
	}

	if len(params) > 0 {
		log.Errorf("%s called Missing() with an odd number of arguments")
	}

	if len(e.Missing) > 0 {
		r.Fail(e)
		return true
	}

	return false
}
