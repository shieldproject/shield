package core

import (
	"fmt"
	"net/http"
	"regexp"
)

func (core *Core) oauth2(w http.ResponseWriter, req *http.Request) {
	//todo change regex to be "anything but a slash"
	//todo add optional trailing piece to end of regex
	re := regexp.MustCompile("/auth/oauth/([^/]+)(/redir)?")
	m := re.FindStringSubmatch(req.URL.Path)

	name := m[1]
	provider, err := core.FindAuthProvider(name)
	if err != nil {
		w.WriteHeader(404)
		fmt.Fprintf(w, "couldn't find that auth provider...: %s", err)
		return
	}

	if m[2] == "/redir" {
		provider.HandleRedirect(w, req)
	} else {
		provider.Initiate(w, req)
	}

}
