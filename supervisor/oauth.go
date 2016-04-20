package supervisor

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/securecookie"
	"github.com/markbates/goth/gothic"
	"github.com/starkandwayne/goutils/log"
)

type OAuthenticator struct {
	Cfg OAuthConfig
}

func (oa OAuthenticator) IsAuthenticated(r *http.Request) bool {
	authType, authToken := AuthHeader(r)
	if strings.ToLower(authType) == "bearer" {
		log.Debugf("Received bearer token auth request")
		// jwt.Parse does both parsing and validating of the token
		token, err := jwt.Parse(authToken, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}

			return oa.Cfg.JWTPublicKey, nil
		})
		if err != nil {
			return false
		}

		if expir, ok := token.Claims["expiration"].(float64); !ok || int64(expir) <= time.Now().Unix() {
			return false
		}
		return true
	}

	sess, err := gothic.Store.Get(r, gothic.SessionName)
	if err == nil && sess.Values["User"] != nil {
		return true
	}
	return false
}

func (oa OAuthenticator) RequireAuth(w http.ResponseWriter, r *http.Request) {
	sess, err := gothic.Store.Get(r, gothic.SessionName)
	if err != nil {
		if _, ok := err.(securecookie.Error); ok {
			r.Header.Set("Cookie", "")
			sess, err = gothic.Store.New(r, gothic.SessionName)
			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte("Failure generating a new session: " + err.Error()))
				return
			}
		} else {
			log.Errorf("%s", err)
			w.WriteHeader(500)
			w.Write([]byte("Unexpected error retrieving session data"))
			return
		}
	}
	sess.AddFlash(r.URL.Path)
	sess.Save(r, w)
	// only start oauth redirection if we're hitting the auth APIs, or web UI
	if ShouldOAuthRedirect(r.URL.Path) {
		log.Debugf("Starting OAuth Process for request: %s", r)
		gothic.BeginAuthHandler(w, r)
	} else {
		// otherwise set auth header for api clients to understand oauth is needed
		log.Debugf("Unauthenticated API Request received, OAuth required, sending 401")
		w.Header().Set("WWW-Authenticate", "Bearer")
		w.WriteHeader(401)
		w.Write([]byte("Unauthorized"))
	}
}

var apiCall = regexp.MustCompile(`^/v\d+/`)
var authCall = regexp.MustCompile(`^/v\d+/auth/`)
var cliAuthCall = regexp.MustCompile(`^/v\d+/auth/cli/?$`)

func ShouldOAuthRedirect(path string) bool {
	return !apiCall.MatchString(path) || authCall.MatchString(path)
}

var OAuthCallback = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	log.Debugf("Incoming Auth request: %s", r)
	sess, err := gothic.Store.Get(r, gothic.SessionName)
	if err != nil {
		log.Errorf("Error retrieving session info: %s", err)
		w.WriteHeader(500)
		return
	}
	log.Debugf("Processing oauth callback for '%s'", sess.ID)
	if gothic.GetState(r) != sess.Values["state"] {
		w.WriteHeader(403)
		w.Write([]byte("Unauthorized"))
		return
	}

	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		w.WriteHeader(403)
		w.Write([]byte("UnOAuthorized"))
		return
	} else {
		log.Infof("Successful login for %s", user)
	}
	//FIXME: do some sort of authorization on the user to restrict access to more than any user with a password

	redirect := "/"
	if flashes := sess.Flashes(); len(flashes) > 0 {
		if flash, ok := flashes[0].(string); ok {
			// don't redirect back to api calls, to prevent auth redirection loops
			if !apiCall.MatchString(flash) || cliAuthCall.MatchString(flash) {
				redirect = flash
			}
		}
	}
	sess.Values["User"] = user
	sess.Save(r, w)

	http.Redirect(w, r, redirect, 302) // checks auth
})
