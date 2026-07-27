package route

import (
	"net/http"
	"strings"
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

// secure reports whether the browser reached us over TLS, and so whether the
// Secure attribute belongs on the cookies we hand back.
//
// It cannot be set unconditionally: shieldd itself only ever speaks plain
// HTTP, and a browser discards a Secure cookie that arrives over an
// unencrypted connection.  Marking every cookie Secure therefore locks users
// out of the web UI of any SHIELD that is not fronted by a TLS terminator.
// On a plain connection the session id travels in the clear regardless, so
// the attribute has nothing left to protect there anyway.
func (r *Request) secure() bool {
	if r.Req == nil {
		return false
	}
	if r.Req.TLS != nil {
		return true
	}

	/* A TLS-terminating proxy reports the browser's original scheme here.
	   Honoring it from an untrusted client can only ever add the attribute,
	   never remove it, so a forged header costs that client its own cookie
	   and gains it nothing. */
	return strings.EqualFold(r.Req.Header.Get("X-Forwarded-Proto"), "https")
}

func (r *Request) SetCookie(name, val, path string) {
	http.SetCookie(r.w, &http.Cookie{
		Name:     name,
		Value:    val,
		Path:     path,
		HttpOnly: true,
		Secure:   r.secure(),
		SameSite: cookieSameSite,
	})
}

func (r *Request) ClearCookie(name, path string) {
	http.SetCookie(r.w, &http.Cookie{
		Name:     name,
		Path:     path,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.secure(),
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
