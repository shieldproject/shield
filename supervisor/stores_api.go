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

type StoreAPI struct {
	Data       *db.DB
	ResyncChan chan int
}

func (self StoreAPI) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	switch {
	case match(req, `GET /v1/stores`):
		stores, err := self.Data.GetAllAnnotatedStores(
			&db.StoreFilter{
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

		JSON(w, stores)
		return

	case match(req, `POST /v1/stores`):
		if req.Body == nil {
			w.WriteHeader(400)
			return
		}

		var params struct {
			Name     string `json:"name"`
			Summary  string `json:"summary"`
			Plugin   string `json:"plugin"`
			Endpoint string `json:"endpoint"`
		}
		if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
			bailWithError(w, ClientErrorf("bad JSON payload: %s", err))
			return
		}

		e := MissingParameters()
		e.Check("name", params.Name)
		e.Check("plugin", params.Plugin)
		e.Check("endpoint", params.Endpoint)
		if e.IsValid() {
			bailWithError(w, e)
			return
		}

		id, err := self.Data.CreateStore(params.Plugin, params.Endpoint)
		if err != nil {
			bail(w, err)
			return
		}

		_ = self.Data.AnnotateStore(id, params.Name, params.Summary)
		self.ResyncChan <- 1
		JSONLiteral(w, fmt.Sprintf(`{"ok":"created","uuid":"%s"}`, id.String()))
		return

	case match(req, `GET /v1/store/[a-fA-F0-9-]+`):
		re := regexp.MustCompile(`^/v1/store/([a-fA-F0-9-]+)`)
		id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

		store, err := self.Data.GetAnnotatedStore(id)
		if err != nil {
			bail(w, err)
			return
		}

		if store == nil {
			w.WriteHeader(404)
			return
		}

		JSON(w, store)
		return

	case match(req, `PUT /v1/store/[a-fA-F0-9-]+`):
		if req.Body == nil {
			w.WriteHeader(400)
			return
		}

		var params struct {
			Name     string `json:"name"`
			Summary  string `json:"summary"`
			Plugin   string `json:"plugin"`
			Endpoint string `json:"endpoint"`
		}
		if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
			bailWithError(w, ClientErrorf("bad JSON payload: %s", err))
			return
		}

		e := MissingParameters()
		e.Check("name", params.Name)
		e.Check("plugin", params.Plugin)
		e.Check("endpoint", params.Endpoint)
		if e.IsValid() {
			bailWithError(w, e)
			return
		}

		re := regexp.MustCompile("^/v1/store/")
		id := uuid.Parse(re.ReplaceAllString(req.URL.Path, ""))
		if err := self.Data.UpdateStore(id, params.Plugin, params.Endpoint); err != nil {
			bail(w, err)
			return
		}
		_ = self.Data.AnnotateStore(id, params.Name, params.Summary)
		self.ResyncChan <- 1
		JSONLiteral(w, fmt.Sprintf(`{"ok":"updated"}`))
		return

	case match(req, `DELETE /v1/store/[a-fA-F0-9-]+`):
		re := regexp.MustCompile("^/v1/store/")
		id := uuid.Parse(re.ReplaceAllString(req.URL.Path, ""))
		deleted, err := self.Data.DeleteStore(id)

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
