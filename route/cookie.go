package route

import (
	"net/http"
)

func SessionID(req *http.Request) string {
	if s := req.Header.Get(SessionHeaderKey); s != "" {
		return s
	}

	if c, err := req.Cookie(SessionCookieKey); err == nil {
		return c.Value
	}

	return ""
}

func (r *Request) SessionID() string {
	return SessionID(r.Req)
}

func (r *Request) SetCookie(name, val, path string) {
	http.SetCookie(r.w, &http.Cookie{
		Name:  name,
		Value: val,
		Path:  path,
	})
}

func (r *Request) ClearCookie(name, path string) {
	http.SetCookie(r.w, &http.Cookie{
		Name:   name,
		Path:   path,
		MaxAge: 0,
	})
}

func (r *Request) SetSession(id string) {
	r.SetCookie(SessionCookieKey, id, "/")
	r.w.Header().Set(SessionHeaderKey, id)
}

func (r *Request) ClearSession() {
	r.ClearCookie(SessionCookieKey, "/")
}
