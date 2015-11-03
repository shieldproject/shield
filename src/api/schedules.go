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
	Data *db.ORM
}

func (s ScheduleAPI) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	switch {
	case match(req, `GET /v1/schedules`):
		schedules, err := s.Data.GetAllAnnotatedSchedules()
		if err != nil {
			w.WriteHeader(500)
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

		id, err := s.Data.CreateSchedule(params.When)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		_ = s.Data.AnnotateSchedule(id, params.Name, params.Summary)
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
		if err := s.Data.UpdateSchedule(id, params.When); err != nil {
			w.WriteHeader(500)
			return
		}
		_ = s.Data.AnnotateSchedule(id, params.Name, params.Summary)

		JSONLiteral(w, fmt.Sprintf(`{"ok":"updated","uuid":"%s"}`, id.String()))
		return
	}

	w.WriteHeader(415)
	return
}
