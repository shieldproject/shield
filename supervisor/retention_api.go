// Jamie: This contains the go source code that will become shield.

package supervisor

import (
	"encoding/json"
	"fmt"
	"github.com/pborman/uuid"
	"github.com/starkandwayne/shield/db"
	"net/http"
	"regexp"
)

type RetentionAPI struct {
	Data       *db.DB
	ResyncChan chan int
}

func (self RetentionAPI) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	switch {
	case match(req, `GET /v1/retention`):
		policies, err := self.Data.GetAllAnnotatedRetentionPolicies(
			&db.RetentionFilter{
				SkipUsed:   paramEquals(req, "unused", "t"),
				SkipUnused: paramEquals(req, "unused", "f"),
			},
		)
		if err != nil {
			bail(w, err)
			return
		}

		JSON(w, policies)
		return

	case match(req, `POST /v1/retention`):
		if req.Body == nil {
			w.WriteHeader(400)
			return
		}

		var params struct {
			Name    string `json:"name"`
			Summary string `json:"summary"`
			Expires uint   `json:"expires"`
		}
		json.NewDecoder(req.Body).Decode(&params)

		if params.Name == "" || params.Expires < 3600 {
			w.WriteHeader(400)
			return
		}

		id, err := self.Data.CreateRetentionPolicy(params.Expires)
		if err != nil {
			bail(w, err)
			return
		}

		_ = self.Data.AnnotateRetentionPolicy(id, params.Name, params.Summary)
		self.ResyncChan <- 1
		JSONLiteral(w, fmt.Sprintf(`{"ok":"created","uuid":"%s"}`, id.String()))
		return

	case match(req, `GET /v1/retention/[a-fA-F0-9-]+`):
		re := regexp.MustCompile("^/v1/retention/")
		id := uuid.Parse(re.ReplaceAllString(req.URL.Path, ""))

		policy, err := self.Data.GetAnnotatedRetentionPolicy(id)
		if err != nil {
			bail(w, err)
			return
		}

		if policy == nil {
			w.WriteHeader(404)
			return
		}

		JSON(w, policy)
		return

	case match(req, `PUT /v1/retention/[a-fA-F0-9-]+`):
		if req.Body == nil {
			w.WriteHeader(400)
			return
		}

		var params struct {
			Name    string `json:"name"`
			Summary string `json:"summary"`
			Expires uint   `json:"expires"`
		}
		json.NewDecoder(req.Body).Decode(&params)

		if params.Name == "" || params.Summary == "" || params.Expires < 3600 {
			w.WriteHeader(400)
			return
		}

		re := regexp.MustCompile("^/v1/retention/")
		id := uuid.Parse(re.ReplaceAllString(req.URL.Path, ""))
		if err := self.Data.UpdateRetentionPolicy(id, params.Expires); err != nil {
			bail(w, err)
			return
		}

		_ = self.Data.AnnotateRetentionPolicy(id, params.Name, params.Summary)
		self.ResyncChan <- 1
		JSONLiteral(w, fmt.Sprintf(`{"ok":"updated"}`))
		return

	case match(req, `DELETE /v1/retention/[a-fA-F0-9-]+`):
		re := regexp.MustCompile("^/v1/retention/")
		id := uuid.Parse(re.ReplaceAllString(req.URL.Path, ""))
		deleted, err := self.Data.DeleteRetentionPolicy(id)

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

	w.WriteHeader(415)
	return
}
