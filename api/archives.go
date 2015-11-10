package api

import (
	"encoding/json"
	"fmt"
	"github.com/pborman/uuid"
	"github.com/starkandwayne/shield/db"
	"net/http"
	"regexp"
)

type ArchiveAPI struct {
	Data      *db.DB
	SuperChan chan int
}

func (self ArchiveAPI) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch {
	case match(req, `GET /v1/archives`):
		archives, err := self.Data.GetAllAnnotatedArchives(
			&db.ArchiveFilter{
				ForTarget: paramValue(req, "target", ""),
				ForStore:  paramValue(req, "store", ""),
				/* FIXME: before/after stuff */
			},
		)
		if err != nil {
			bail(w, err)
			return
		}

		JSON(w, archives)
		return

	case match(req, `POST /v1/archive/[a-fA-F0-9-]+/restore`):
		w.WriteHeader(420)
		return

	case match(req, `PUT /v1/archive/[a-fA-F0-9-]+`):
		if req.Body == nil {
			w.WriteHeader(400)
			return
		}

		var params struct {
			Notes string `json:"notes"`
		}
		json.NewDecoder(req.Body).Decode(&params)

		if params.Notes == "" {
			w.WriteHeader(400)
			return
		}

		re := regexp.MustCompile(`^/v1/archive/([a-fA-F0-9-]+)`)
		id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

		_ = self.Data.AnnotateArchive(id, params.Notes)
		self.SuperChan <- 1
		JSONLiteral(w, fmt.Sprintf(`{"ok":"updated"}`))
		return

	case match(req, `DELETE /v1/archive/[a-fA-F0-9-]+`):
		re := regexp.MustCompile(`^/v1/archive/([a-fA-F0-9-]+)`)
		id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

		deleted, err := self.Data.DeleteArchive(id)

		if err != nil {
			bail(w, err)
		}
		if !deleted {
			w.WriteHeader(403)
			return
		}

		JSONLiteral(w, fmt.Sprintf(`{"ok":"deleted"}`))
		return
	}

	w.WriteHeader(415)
	return
}
