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

type JobAPI struct {
	Data       *db.DB
	ResyncChan chan int
	AdhocChan  chan AdhocTask
}

func (self JobAPI) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch {
	case match(req, `GET /v1/jobs`):
		jobs, err := self.Data.GetAllAnnotatedJobs(
			&db.JobFilter{
				SkipPaused:   paramEquals(req, "paused", "f"),
				SkipUnpaused: paramEquals(req, "paused", "t"),

				SearchName: paramValue(req, "name", ""),

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
		if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
			bailWithError(w, ClientErrorf("bad JSON payload: %s", err))
			return
		}

		e := MissingParameters()
		e.Check("name", params.Name)
		e.Check("store", params.Store)
		e.Check("target", params.Target)
		e.Check("schedule", params.Schedule)
		e.Check("retention", params.Retention)
		if e.IsValid() {
			bailWithError(w, e)
			return
		}

		id, err := self.Data.CreateJob(params.Target, params.Store, params.Schedule, params.Retention, params.Paused)
		if err != nil {
			bail(w, err)
			return
		}

		_ = self.Data.AnnotateJob(id, params.Name, params.Summary)
		self.ResyncChan <- 1
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

		self.ResyncChan <- 1
		JSONLiteral(w, `{"ok":"paused"}`)
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

		self.ResyncChan <- 1
		JSONLiteral(w, `{"ok":"unpaused"}`)
		return

	case match(req, `POST /v1/job/[a-fA-F0-9-]+/run`):
		if req.Body == nil {
			w.WriteHeader(400)
			return
		}

		var params struct {
			Owner string `json:"owner"`
		}
		if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
			bailWithError(w, ClientErrorf("bad JSON payload: %s", err))
			return
		}

		if params.Owner == "" {
			params.Owner = "anon"
		}

		re := regexp.MustCompile(`^/v1/job/([a-fA-F0-9-]+)/run`)
		id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

		self.AdhocChan <- AdhocTask{
			Op:      BACKUP,
			Owner:   params.Owner,
			JobUUID: id,
		}
		JSONLiteral(w, `{"ok":"scheduled"}`)
		return

	case match(req, `GET /v1/job/[a-fA-F0-9-]+`):
		re := regexp.MustCompile(`^/v1/job/([a-fA-F0-9-]+)`)
		id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

		job, err := self.Data.GetAnnotatedJob(id)
		if err != nil {
			bail(w, err)
			return
		}

		if job == nil {
			w.WriteHeader(404)
			return
		}

		JSON(w, job)
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
		if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
			bailWithError(w, ClientErrorf("bad JSON payload: %s", err))
			return
		}

		e := MissingParameters()
		e.Check("name", params.Name)
		e.Check("store", params.Store)
		e.Check("target", params.Target)
		e.Check("schedule", params.Schedule)
		e.Check("retention", params.Retention)
		if e.IsValid() {
			bailWithError(w, e)
			return
		}

		re := regexp.MustCompile(`^/v1/job/([a-fA-F0-9-]+)`)
		id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

		if err := self.Data.UpdateJob(id, params.Target, params.Store, params.Schedule, params.Retention); err != nil {
			bail(w, err)
			return
		}
		_ = self.Data.AnnotateJob(id, params.Name, params.Summary)
		self.ResyncChan <- 1
		JSONLiteral(w, `{"ok":"updated"}`)
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

		self.ResyncChan <- 1
		JSONLiteral(w, `{"ok":"deleted"}`)
		return
	}

	w.WriteHeader(501)
	return
}
