package core2

import (
	//"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
)

const APIVersion = 2

func (core *Core) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch {
	case match(req, `GET /v1/meta/pubkey`):
		core.v1GetPublicKey(w, req)
		return
	}

	w.WriteHeader(501)
}

func match(req *http.Request, pattern string) bool {
	matched, _ := regexp.MatchString(
		fmt.Sprintf("^%s$", pattern),
		fmt.Sprintf("%s %s", req.Method, req.URL.Path))
	return matched
}

func (core *Core) v1GetPublicKey(w http.ResponseWriter, req *http.Request) {
	//	pub := core.agent.key.PublicKey()
	w.WriteHeader(200)
	//	fmt.Fprintf(w, "%s %s\n", pub.Type(), base64.StdEncoding.EncodeToString(pub.Marshal()))
	fmt.Fprintf(w, "...\n")
}
