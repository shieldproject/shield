package supervisor

import (
	"encoding/base64"
	"github.com/starkandwayne/goutils/log"
	"net/http"
	"strings"
)

func obfuscate(p string) string {
	return strings.Repeat("•", len(p))
}

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
	log.Debugf("Checking `Authorization: %s %s` against our configuration", authType, authToken)

	if strings.ToLower(authType) == "basic" {
		decoded, err := base64.StdEncoding.DecodeString(authToken)
		if err != nil {
			log.Infof("Authorization header is corrupt: %s", err)
			return false
		}

		creds := strings.SplitN(string(decoded), ":", 2)
		if len(creds) != 2 {
			log.Infof("Authorization header is corrupt: '%s' does not contain a ':' delimiter",
				string(decoded))
			return false
		}

		log.Debugf("Received Authorization credentials for user '%s', password '%s'", creds[0], obfuscate(creds[1]))
		log.Debugf("checking against the configured credentials '%s', password '%s'", ba.Cfg.User, obfuscate(ba.Cfg.Password))
		return creds[0] == ba.Cfg.User && creds[1] == ba.Cfg.Password
	}

	log.Infof("Received an invalid Authorization header type '%s' (not 'Basic')", authType)
	return false
}

func (ba BasicAuthenticator) RequireAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", "Basic realm=\"shield\"")
	w.WriteHeader(401)
	w.Write([]byte("Unauthorized"))
}

func Authenticate(tokens map[string]string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providedToken := r.Header.Get("X-Shield-Token")
		if providedToken != "" {
			log.Debugf("Checking X-Shield-Token against available tokens")
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
	log.Debugf("Retrieving auth header `%v`", r.Header.Get("Authorization"))
	auth := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(auth) != 2 {
		return "", ""
	}
	return auth[0], auth[1]
}
