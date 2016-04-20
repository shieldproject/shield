package supervisor

import (
	"encoding/base64"
	"github.com/starkandwayne/goutils/log"
	"net/http"
	"strings"
)

type Authenticator interface {
	IsAuthenticated(*http.Request) bool
	RequireAuth(http.ResponseWriter, *http.Request)
}

var UserAuthenticator Authenticator

type BasicAuthenticator struct {
	Cfg BasicAuthConfig
}

func (ba BasicAuthenticator) IsAuthenticated(r *http.Request) bool {
	authType, authToken := AuthHeader(r)
	if strings.ToLower(authType) == "basic" {
		decoded, err := base64.StdEncoding.DecodeString(authToken)
		if err != nil {
			log.Infof("Received invalid auth request")
		} else {
			creds := strings.SplitN(string(decoded), ":", 2)
			if creds[0] == ba.Cfg.User && creds[1] == ba.Cfg.Password {
				return true
			}
		}
	}
	return false
}

func (ba BasicAuthenticator) RequireAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", "Basic realm=\"shield\"")
	w.WriteHeader(401)
	w.Write([]byte("Unauthorized"))
}

func Authenticate(tokens map[string]string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for name, token := range tokens {
			log.Debugf("Checking Tokens")
			if r.Header.Get("X-SHIELD-TOKEN") == token {
				log.Debugf("Matched token %s!", name)
				next.ServeHTTP(w, r)
				return
			}
		}

		if UserAuthenticator.IsAuthenticated(r) {
			next.ServeHTTP(w, r)
		} else {
			UserAuthenticator.RequireAuth(w, r)
			return
		}
	})
}

func AuthHeader(r *http.Request) (string, string) {
	auth := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(auth) != 2 {
		return "", ""
	}
	return auth[0], auth[1]
}
