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

// SameSite=Lax, not Strict.  SHIELD's OAuth2 providers (github, okta, uaa)
// send the browser back to /auth/:provider/redir as a top-level cross-site
// navigation.  Strict cookies are withheld on such a navigation, so both the
// "via" cookie and any pre-existing session cookie would be missing by the
// time the redirect handler runs -- breaking `shield login` against every
// oauth provider.  Lax still withholds the cookie from cross-site POSTs and
// subresource requests, which is where the CSRF risk actually lives.
const cookieSameSite = http.SameSiteLaxMode

func (r *Request) SetCookie(name, val, path string) {
	http.SetCookie(r.w, &http.Cookie{
		Name:     name,
		Value:    val,
		Path:     path,
		HttpOnly: true,
		Secure:   true,
		SameSite: cookieSameSite,
	})
}

func (r *Request) ClearCookie(name, path string) {
	http.SetCookie(r.w, &http.Cookie{
		Name:     name,
		Path:     path,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: cookieSameSite,
	})
}

func (r *Request) SetSession(id string) {
	r.SetCookie(SessionCookieKey, id, "/")
	r.w.Header().Set(SessionHeaderKey, id)
}

func (r *Request) ClearSession() {
	r.ClearCookie(SessionCookieKey, "/")
}
