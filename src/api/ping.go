package api

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

	header := w.Header()
	header["Content-Type"] = []string{"application/json"}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ok":"pong"}`))
}
