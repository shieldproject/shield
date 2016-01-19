package supervisor

import (
	"net/http"
	"os"

	"github.com/starkandwayne/shield/version"
)

type StatusAPI struct{}

func (p StatusAPI) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" || req.URL.Path != "/v1/status" {
		w.WriteHeader(501)
		return
	}

	JSON(w, struct {
		Version string `json:"version"`
		Name    string `json:"name"`
	}{
		Version: version.String(),
		Name:    os.Getenv("SHIELD_NAME"),
	})
}
