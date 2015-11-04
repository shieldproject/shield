// Jamie: This contains the go source code that will become shield.

package api

import (
	"db"

	"github.com/pborman/uuid"

	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
)

type ScheduleAPI struct {
	Data *db.DB
}

func (self ScheduleAPI) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch {
	case match(req, `GET /v1/schedules`):
		schedules, err := self.Data.GetAllAnnotatedSchedules(
			&db.ScheduleFilter{
				SkipUsed:   paramEquals(req, "unused", "t"),
				SkipUnused: paramEquals(req, "unused", "f"),
			},
		)
		if err != nil {
			bail(w, err)
			return
		}

		JSON(w, schedules)
		return

	case match(req, `POST /v1/schedules`):
		if req.Body == nil {
			w.WriteHeader(400)
			return
		}

		var params struct {
			Name    string `json:"name"`
			Summary string `json:"summary"`
			When    string `json:"when"`
		}
		json.NewDecoder(req.Body).Decode(&params)

		if params.Name == "" || params.When == "" {
			w.WriteHeader(400)
			return
		}

		id, err := self.Data.CreateSchedule(params.When)
		if err != nil {
			bail(w, err)
			return
		}

		_ = self.Data.AnnotateSchedule(id, params.Name, params.Summary)
		JSONLiteral(w, fmt.Sprintf(`{"ok":"created","uuid":"%s"}`, id.String()))
		return

	case match(req, `PUT /v1/schedule/[a-fA-F0-9-]+`):
		if req.Body == nil {
			w.WriteHeader(400)
			return
		}

		var params struct {
			Name    string `json:"name"`
			Summary string `json:"summary"`
			When    string `json:"when"`
		}
		json.NewDecoder(req.Body).Decode(&params)

		if params.Name == "" || params.Summary == "" || params.When == "" {
			w.WriteHeader(400)
			return
		}

		re := regexp.MustCompile("^/v1/schedule/")
		id := uuid.Parse(re.ReplaceAllString(req.URL.Path, ""))
		if err := self.Data.UpdateSchedule(id, params.When); err != nil {
			bail(w, err)
			return
		}
		_ = self.Data.AnnotateSchedule(id, params.Name, params.Summary)

		JSONLiteral(w, fmt.Sprintf(`{"ok":"updated","uuid":"%s"}`, id.String()))
		return

	case match(req, `DELETE /v1/schedule/[a-fA-F0-9-]+`):
		re := regexp.MustCompile("^/v1/schedule/")
		id := uuid.Parse(re.ReplaceAllString(req.URL.Path, ""))
		deleted, err := self.Data.DeleteSchedule(id)

		if err != nil {
			bail(w, err)
		}
		if !deleted {
			w.WriteHeader(403)
			return
		}

		w.WriteHeader(200)
		return
	}

	w.WriteHeader(415)
	return
}
