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
		providedToken := r.Header.Get("X-SHIELD-TOKEN")
		if providedToken != "" {
			log.Debugf("Checking X-SHIELD-TOKEN against available tokens")
			for name, token := range tokens {
				if providedToken == token {
					log.Debugf("Matched token %s!", name)
					next.ServeHTTP(w, r)
					return
				}
			}
			log.Debugf("No tokens matched")
		}

		if UserAuthenticator.IsAuthenticated(r) {
			log.Debugf("Request was authenticated, continuing to process")
			next.ServeHTTP(w, r)
		} else {
			log.Debugf("Request not authenticated, denying")
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
