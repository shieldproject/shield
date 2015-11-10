package api

import (
	"db"

	"github.com/pborman/uuid"

	"fmt"
	"regexp"
	"encoding/json"
	"net/http"
)

type JobAPI struct {
	Data *db.DB
	SuperChan chan int
}

func (self JobAPI) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch {
	case match(req, `GET /v1/jobs`):
		jobs, err := self.Data.GetAllAnnotatedJobs(
			&db.JobFilter{
				SkipPaused:   paramEquals(req, "paused", "f"),
				SkipUnpaused: paramEquals(req, "paused", "t"),

				ForTarget:    paramValue(req, "target", ""),
				ForStore:     paramValue(req, "store", ""),
				ForSchedule:  paramValue(req, "schedule", ""),
				ForRetention: paramValue(req, "retention", ""),
			},
		)
		if err != nil {
			bail(w, err)
			return
		}

		JSON(w, jobs)
		return

	case match(req, `POST /v1/jobs`):
		if req.Body == nil {
			w.WriteHeader(400)
			return
		}

		var params struct {
			Name    string `json:"name"`
			Summary string `json:"summary"`

			Store     string `json:"store"`
			Target    string `json:"target"`
			Schedule  string `json:"schedule"`
			Retention string `json:"retention"`

			Paused bool `json:"paused"`
		}
		json.NewDecoder(req.Body).Decode(&params)

		if params.Name == "" || params.Store == "" || params.Target == "" || params.Schedule == "" || params.Retention == "" {
			w.WriteHeader(400)
			return
		}

		id, err := self.Data.CreateJob(params.Target, params.Store, params.Schedule, params.Retention, params.Paused)
		if err != nil {
			bail(w, err)
			return
		}

		_ = self.Data.AnnotateJob(id, params.Name, params.Summary)
		self.SuperChan <- 1
		JSONLiteral(w, fmt.Sprintf(`{"ok":"created","uuid":"%s"}`, id.String()))
		return

	case match(req, `POST /v1/job/[a-fA-F0-9-]+/pause`):
		re := regexp.MustCompile(`^/v1/job/([a-fA-F0-9-]+)/pause`)
		id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

		found, err := self.Data.PauseJob(id)
		if !found {
			w.WriteHeader(404)
			return
		}
		if err != nil {
			bail(w, err)
			return
		}

		self.SuperChan <- 1
		JSONLiteral(w, fmt.Sprintf(`{"ok":"paused"`))
		return

	case match(req, `POST /v1/job/[a-fA-F0-9-]+/unpause`):
		re := regexp.MustCompile(`^/v1/job/([a-fA-F0-9-]+)/unpause`)
		id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

		found, err := self.Data.UnpauseJob(id)
		if !found {
			w.WriteHeader(404)
			return
		}
		if err != nil {
			bail(w, err)
			return
		}

		self.SuperChan <- 1
		JSONLiteral(w, fmt.Sprintf(`{"ok":"unpaused"`))
		return

	case match(req, `PUT /v1/job/[a-fA-F0-9-]+`):
		if req.Body == nil {
			w.WriteHeader(400)
			return
		}

		var params struct {
			Name    string `json:"name"`
			Summary string `json:"summary"`

			Store     string `json:"store"`
			Target    string `json:"target"`
			Schedule  string `json:"schedule"`
			Retention string `json:"retention"`
		}
		json.NewDecoder(req.Body).Decode(&params)

		if params.Name == "" || params.Summary == "" || params.Store == "" || params.Target == "" || params.Schedule == "" || params.Retention == "" {
			w.WriteHeader(400)
			return
		}

		re := regexp.MustCompile(`^/v1/job/([a-fA-F0-9-]+)`)
		id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

		if err := self.Data.UpdateJob(id, params.Target, params.Store, params.Schedule, params.Retention); err != nil {
			bail(w, err)
			return
		}
		_ = self.Data.AnnotateJob(id, params.Name, params.Summary)
		self.SuperChan <- 1
		JSONLiteral(w, fmt.Sprintf(`{"ok":"updated"}`))
		return

	case match(req, `DELETE /v1/job/[a-fA-F0-9-]+`):
		re := regexp.MustCompile(`^/v1/job/([a-fA-F0-9-]+)`)
		id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

		deleted, err := self.Data.DeleteJob(id)

		if err != nil {
			bail(w, err)
		}
		if !deleted {
			w.WriteHeader(403)
			return
		}

		self.SuperChan <- 1
		JSONLiteral(w, fmt.Sprintf(`{"ok":"deleted"}`))
		return
	}

	w.WriteHeader(415)
	return
}
