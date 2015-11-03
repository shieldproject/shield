package api

import (
	"db"

	"encoding/json"
	"fmt"
	"net/http"
)

type ScheduleAPI struct {
	Data *db.ORM
}

func (s ScheduleAPI) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	switch req.Method {
	case "GET":
		schedules, err := s.Data.GetAllAnnotatedSchedules()
		if err != nil {
			w.WriteHeader(500)
			return
		}

		b, err := json.Marshal(schedules)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		header := w.Header()
		header["Content-Type"] = []string{"application/json"}
		w.WriteHeader(200)
		w.Write(b)
		return

	case "POST":
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
			w.Write([]byte(err.Error()))
			return
		}

		_ = s.Data.AnnotateSchedule(id, params.Name, params.Summary)
		header := w.Header()
		header["Content-Type"] = []string{"application/json"}
		w.WriteHeader(200)
		w.Write([]byte(fmt.Sprintf(`{"ok":"created","uuid":"%s"}`, id.String())))
		return
	}

	w.WriteHeader(415)
	return
}
