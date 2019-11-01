package metrics

import (
	"net/http"
)

type BasicAuthenticator struct {
	username string
	password string
	realm    string

	handler http.Handler
}

func (b BasicAuthenticator) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok || user != b.username || pass != b.password {
		w.Header().Set("WWW-Authenticate", `Basic realm="`+b.realm+`"`)
		w.WriteHeader(401)
		w.Write([]byte("# unauthorised\n"))
		return
	}
	b.handler.ServeHTTP(w, r)
}
