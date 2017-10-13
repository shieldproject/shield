package core

import (
	"net/http"

	"github.com/starkandwayne/goutils/log"

	"github.com/starkandwayne/shield/db"
)

type AuthProvider interface {
	DisplayName() string

	Configure(map[interface{}]interface{}) error
	Initiate(http.ResponseWriter, *http.Request)
	HandleRedirect(*http.Request) *db.User
}

type AuthProviderBase struct {
	Name       string
	Identifier string
	Type       string
}

func (p AuthProviderBase) DisplayName() string {
	return p.Name
}

func (p AuthProviderBase) Errorf(m string, args ...interface{}) {
	args = append([]interface{}{p.Identifier, p.Type}, args...)
	log.Errorf("auth provider %s (%s): "+m, args...)
}

func (p AuthProviderBase) Infof(m string, args ...interface{}) {
	args = append([]interface{}{p.Identifier, p.Type}, args...)
	log.Infof("auth provider %s (%s): "+m, args...)
}

func (p AuthProviderBase) Debugf(m string, args ...interface{}) {
	args = append([]interface{}{p.Identifier, p.Type}, args...)
	log.Debugf("auth provider %s (%s): "+m, args...)
}

func (p AuthProviderBase) Fail(w http.ResponseWriter) {
	w.Header().Set("Location", "/fail/e500")
	w.WriteHeader(302)
}
