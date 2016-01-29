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

type ScheduleAPI struct {
	Data       *db.DB
	ResyncChan chan int
}

func (self ScheduleAPI) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch {
	case match(req, `GET /v1/schedules`):
		schedules, err := self.Data.GetAllAnnotatedSchedules(
			&db.ScheduleFilter{
				SkipUsed:   paramEquals(req, "unused", "t"),
				SkipUnused: paramEquals(req, "unused", "f"),

				SearchName: paramValue(req, "name", ""),
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
		if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
			bailWithError(w, ClientErrorf("bad JSON payload: %s", err))
			return
		}

		e := MissingParameters()
		e.Check("name", params.Name)
		e.Check("when", params.When)
		if e.IsValid() {
			bailWithError(w, e)
			return
		}

		id, err := self.Data.CreateSchedule(params.When)
		if err != nil {
			bail(w, err)
			return
		}

		_ = self.Data.AnnotateSchedule(id, params.Name, params.Summary)
		self.ResyncChan <- 1
		JSONLiteral(w, fmt.Sprintf(`{"ok":"created","uuid":"%s"}`, id.String()))
		return

	case match(req, `GET /v1/schedule/[a-fA-F0-9-]+`):
		re := regexp.MustCompile(`^/v1/schedule/([a-fA-F0-9-]+)`)
		id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

		schedule, err := self.Data.GetAnnotatedSchedule(id)
		if err != nil {
			bail(w, err)
			return
		}

		if schedule == nil {
			w.WriteHeader(404)
			return
		}

		JSON(w, schedule)
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
		if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
			bailWithError(w, ClientErrorf("bad JSON payload: %s", err))
			return
		}

		e := MissingParameters()
		e.Check("name", params.Name)
		e.Check("when", params.When)
		if e.IsValid() {
			bailWithError(w, e)
			return
		}

		re := regexp.MustCompile("^/v1/schedule/")
		id := uuid.Parse(re.ReplaceAllString(req.URL.Path, ""))
		if err := self.Data.UpdateSchedule(id, params.When); err != nil {
			bail(w, err)
			return
		}
		_ = self.Data.AnnotateSchedule(id, params.Name, params.Summary)
		self.ResyncChan <- 1
		JSONLiteral(w, fmt.Sprintf(`{"ok":"updated"}`))
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

		self.ResyncChan <- 1
		JSONLiteral(w, fmt.Sprintf(`{"ok":"deleted"}`))
		return
	}

	w.WriteHeader(501)
	return
}
