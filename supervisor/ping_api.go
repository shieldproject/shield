// Jamie: This contains the go source code that will become shield.

package supervisor

import (
	"net/http"
)

type PingAPI struct{}

func (p PingAPI) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/v1/ping" {
		w.WriteHeader(404)
		return
	}
	if req.Method != "GET" {
		w.WriteHeader(415)
		return
	}

	JSON(w, struct {
		OK string `json:"ok"`
	}{OK: "pong"})
}
