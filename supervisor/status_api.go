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

	default:
		w.WriteHeader(501)
		return
	}
}
