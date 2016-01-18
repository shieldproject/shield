package supervisor

import (
	"net/http"
)

type PingAPI struct{}

func (p PingAPI) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" || req.URL.Path != "/v1/ping" {
		w.WriteHeader(501)
		return
	}

	JSON(w, struct {
		OK string `json:"ok"`
	}{OK: "pong"})
}
