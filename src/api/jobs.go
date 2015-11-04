package api

import (
	"db"

	//"github.com/pborman/uuid"

	//"regexp"
	//"encoding/json"
	"net/http"
)

type JobAPI struct {
	Data *db.DB
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
	}

	w.WriteHeader(415)
	return
}
