package route

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jhunt/go-log"
)

const (
	SessionHeaderKey = "X-Shield-Session"
	SessionCookieKey = "shield7"
	TokenHeaderKey   = "X-Shield-Token"
)

type Request struct {
	Req  *http.Request
	Args []string

	w     http.ResponseWriter
	done  bool
	debug bool
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

func (r *Request) SessionID() string {
	if s := r.Req.Header.Get(SessionHeaderKey); s != "" {
		return s
	}

	if c, err := r.Req.Cookie(SessionCookieKey); err == nil {
		return c.Value
	}

	if s := r.Req.Header.Get(TokenHeaderKey); s != "" {
		return s
	}

	return ""
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
	if r.debug {
		e.ProvideDiagnostic()
	}

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

func (r *Request) ParamIs(name, want string) bool {
	v, set := r.Req.URL.Query()[name]
	return set && v[0] == want
}

func (r *Request) SetRespHeader(header, value string) {
	r.w.Header().Add(header, value)
}

func (r *Request) SetCookie(cookie *http.Cookie) {
	http.SetCookie(r.w, cookie)
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
