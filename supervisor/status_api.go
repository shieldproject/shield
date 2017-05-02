package supervisor

import (
	"net/http"
	"os"

	"github.com/starkandwayne/shield/db"
)

var Version = "(development)"

type StatusAPI struct {
	Data  *db.DB
	Super *Supervisor
}

type JobHealth struct {
	Name    string `json:"name"`
	LastRun int64  `json:"last_run"`
	NextRun int64  `json:"next_run"`
	Paused  bool   `json:"paused"`
	Status  string `json:"status"`
}

func (p StatusAPI) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch {
	case match(req, `GET /v1/status`):
		JSON(w, struct {
			Version string `json:"version"`
			Name    string `json:"name"`
		}{
			Version: Version,
			Name:    os.Getenv("SHIELD_NAME"),
		})
		return

	case match(req, `GET /v1/status/internal`):
		pending, err := p.Data.GetAllTasks(&db.TaskFilter{ForStatus: db.PendingStatus})
		if err != nil {
			bail(w, err)
			return
		}
		running, err := p.Data.GetAllTasks(&db.TaskFilter{ForStatus: db.RunningStatus})
		if err != nil {
			bail(w, err)
			return
		}
		JSON(w, struct {
			PendingTasks  []*db.Task `json:"pending_tasks"`
			RunningTasks  []*db.Task `json:"running_tasks"`
			ScheduleQueue []*db.Task `json:"schedule_queue"`
			RunQueue      []*db.Task `json:"run_queue"`
		}{
			PendingTasks:  pending,
			RunningTasks:  running,
			ScheduleQueue: p.Super.schedq,
			RunQueue:      p.Super.runq,
		})
		return

	case match(req, `GET /v1/status/jobs`):
		jobs, err := p.Data.GetAllJobs(&db.JobFilter{})
		if err != nil {
			bail(w, err)
			return
		}

		health := make(map[string]JobHealth)
		for _, j := range jobs {
			var next, last int64
			if j.LastRun.Time().IsZero() {
				last = 0
			} else {
				last = j.LastRun.Time().Unix()
			}

			j.Reschedule() /* not really, just enough to get NextRun */
			if j.Paused || j.NextRun.IsZero() {
				next = 0
			} else {
				next = j.NextRun.Unix()
			}

			health[j.Name] = JobHealth{
				Name:    j.Name,
				Paused:  j.Paused,
				LastRun: last,
				NextRun: next,
				Status:  j.LastTaskStatus,
			}
		}

		JSON(w, health)
		return

	default:
		w.WriteHeader(501)
		return
	}
}
