package supervisor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/pborman/uuid"

	"github.com/starkandwayne/shield/db"
)

type TargetAPI struct {
	Data       *db.DB
	ResyncChan chan int
}

func (self TargetAPI) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	switch {
	case match(req, `GET /v1/targets`):
		targets, err := self.Data.GetAllAnnotatedTargets(
			&db.TargetFilter{
				SkipUsed:   paramEquals(req, "unused", "t"),
				SkipUnused: paramEquals(req, "unused", "f"),
				SearchName: paramValue(req, "name", ""),
				ForPlugin:  paramValue(req, "plugin", ""),
			},
		)
		if err != nil {
			bail(w, err)
			return
		}

		JSON(w, targets)
		return

	case match(req, `POST /v1/targets`):
		if req.Body == nil {
			w.WriteHeader(400)
			return
		}

		var params struct {
			Name     string `json:"name"`
			Summary  string `json:"summary"`
			Plugin   string `json:"plugin"`
			Endpoint string `json:"endpoint"`
			Agent    string `json:"agent"`
		}
		if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
			bailWithError(w, ClientErrorf("bad JSON payload: %s", err))
			return
		}

		e := MissingParameters()
		e.Check("name", params.Name)
		e.Check("plugin", params.Plugin)
		e.Check("endpoint", params.Endpoint)
		e.Check("agent", params.Agent)
		if e.IsValid() {
			bailWithError(w, e)
			return
		}

		id, err := self.Data.CreateTarget(params.Plugin, params.Endpoint, params.Agent)
		if err != nil {
			bail(w, err)
			return
		}

		_ = self.Data.AnnotateTarget(id, params.Name, params.Summary)
		self.ResyncChan <- 1
		JSONLiteral(w, fmt.Sprintf(`{"ok":"created","uuid":"%s"}`, id.String()))
		return

	case match(req, `GET /v1/target/[a-fA-F0-9-]+`):
		re := regexp.MustCompile(`^/v1/target/([a-fA-F0-9-]+)`)
		id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

		target, err := self.Data.GetAnnotatedTarget(id)
		if err != nil {
			bail(w, err)
			return
		}

		if target == nil {
			w.WriteHeader(404)
			return
		}

		JSON(w, target)
		return

	case match(req, `PUT /v1/target/[a-fA-F0-9-]+`):
		if req.Body == nil {
			w.WriteHeader(400)
			return
		}

		var params struct {
			Name     string `json:"name"`
			Summary  string `json:"summary"`
			Plugin   string `json:"plugin"`
			Endpoint string `json:"endpoint"`
			Agent    string `json:"agent"`
		}
		if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
			bailWithError(w, ClientErrorf("bad JSON payload: %s", err))
			return
		}

		e := MissingParameters()
		e.Check("name", params.Name)
		e.Check("plugin", params.Plugin)
		e.Check("endpoint", params.Endpoint)
		e.Check("agent", params.Agent)
		if e.IsValid() {
			bailWithError(w, e)
			return
		}

		re := regexp.MustCompile("^/v1/target/")
		id := uuid.Parse(re.ReplaceAllString(req.URL.Path, ""))
		if err := self.Data.UpdateTarget(id, params.Plugin, params.Endpoint, params.Agent); err != nil {
			bail(w, err)
			return
		}
		_ = self.Data.AnnotateTarget(id, params.Name, params.Summary)
		self.ResyncChan <- 1
		JSONLiteral(w, fmt.Sprintf(`{"ok":"updated"}`))
		return

	case match(req, `DELETE /v1/target/[a-fA-F0-9-]+`):
		re := regexp.MustCompile("^/v1/target/")
		id := uuid.Parse(re.ReplaceAllString(req.URL.Path, ""))
		deleted, err := self.Data.DeleteTarget(id)

		if err != nil {
			bail(w, err)
		}
		if !deleted {
			w.WriteHeader(403)
			return
		}

		self.ResyncChan <- 1
		JSONLiteral(w, fmt.Sprintf(`{"ok":"deleted"}`))
		return
	}

	w.WriteHeader(501)
	return
}
