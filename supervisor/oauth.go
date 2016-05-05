package supervisor

import (
	"fmt"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
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

		userName, ok := token.Claims["user"].(string)
		if !ok {
			log.Debugf("user claim is not a string: %#v", token.Claims["user"])
			return false
		}

		membership, ok := token.Claims["membership"].(map[string]interface{})
		if !ok {
			log.Debugf("membership claim is not a Membership: %#v", token.Claims["membership"])
			return false
		}
		return OAuthVerifier.Verify(userName, membership)
	}

	sess, err := gothic.Store.Get(r, gothic.SessionName)
	if err != nil {
		log.Debugf("Error retrieving session: %s", err)
		return false
	}

	user, ok := sess.Values["User"].(string)
	if ok {
		membership, ok := sess.Values["Membership"].(map[string]interface{})
		if ok {
			return OAuthVerifier.Verify(user, membership)
		}
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

func (oa OAuthenticator) OAuthCallback() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		if r.URL.Query().Get("code") == "" {
			log.Errorf("No code detected in oauth callback: %v", r)
			w.WriteHeader(403)
			w.Write([]byte("No oauth code issued from provider"))
			return
		}

		user, err := gothic.CompleteUserAuth(w, r)
		if err != nil {
			log.Errorf("Error verifying oauth success: %s. Request: %v", err, r)
			w.WriteHeader(403)
			w.Write([]byte("UnOAuthorized"))
			return
		}

		log.Debugf("Authenticated user %#v", user)

		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: user.AccessToken})
		ctx := context.WithValue(oauth2.NoContext, oauth2.HTTPClient, oa.Cfg.Client)
		tc := oauth2.NewClient(ctx, ts)

		log.Debugf("Checking authorization...")
		membership, err := OAuthVerifier.Membership(user, tc)
		if err != nil {
			log.Errorf("Error retreiving user membership: %s", err)
			w.WriteHeader(403)
			w.Write([]byte("Unable to verify your membership"))
			return
		}

		if !OAuthVerifier.Verify(user.NickName, membership) {
			log.Debugf("Authorization denied")
			w.WriteHeader(403)
			w.Write([]byte("You are not authorized to view this content"))
			return
		}

		log.Infof("Successful login for %s", user.NickName)

		redirect := "/"
		if flashes := sess.Flashes(); len(flashes) > 0 {
			if flash, ok := flashes[0].(string); ok {
				// don't redirect back to api calls, to prevent auth redirection loops
				if !apiCall.MatchString(flash) || cliAuthCall.MatchString(flash) {
					redirect = flash
				}
			}
		}

		sess.Values["User"] = user.NickName
		sess.Values["Membership"] = membership
		err = sess.Save(r, w)
		if err != nil {
			log.Errorf("Error saving session: %s", err)
			w.WriteHeader(500)
			w.Write([]byte("Unable to save authentication data. Check the SHIELD logs for more info."))
			return
		}

		http.Redirect(w, r, redirect, 302) // checks auth
	})
}
