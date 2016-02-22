package supervisor

import (
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/pborman/uuid"

	"github.com/starkandwayne/shield/db"
)

type TaskAPI struct {
	Data *db.DB
}

func (self TaskAPI) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch {
	case match(req, `GET /v1/tasks`):
		limit := paramValue(req, "limit", "")
		if invalidlimit(limit) {
			bailWithError(w, ClientErrorf("invalid limit supplied"))
			return
		}
		tasks, err := self.Data.GetAllAnnotatedTasks(
			&db.TaskFilter{
				SkipActive:   paramEquals(req, "active", "f"),
				SkipInactive: paramEquals(req, "active", "t"),
				ForStatus:    paramValue(req, "status", ""),
				Limit:        limit,
			},
		)
		if err != nil {
			bail(w, err)
			return
		}

		JSON(w, tasks)
		return

	case match(req, `GET /v1/task/[a-fA-F0-9-]+`):
		re := regexp.MustCompile(`^/v1/task/([a-fA-F0-9-]+)`)
		id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

		task, err := self.Data.GetAnnotatedTask(id)
		if err != nil {
			bail(w, err)
			return
		}

		if task == nil {
			w.WriteHeader(404)
			return
		}

		JSON(w, task)
		return

	case match(req, `DELETE /v1/task/[a-fA-F0-9-]+`):
		// cancel
		re := regexp.MustCompile(`^/v1/task/([a-fA-F0-9-]+)`)
		id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

		err := self.Data.CancelTask(id, time.Now())

		if err != nil {
			bail(w, err)
		}

		JSONLiteral(w, fmt.Sprintf(`{"ok":"canceled"}`))
		return
	}

	w.WriteHeader(501)
	return
}
