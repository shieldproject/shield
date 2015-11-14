package supervisor

import (
	"fmt"
	"github.com/pborman/uuid"
	"github.com/starkandwayne/shield/db"
	"net/http"
	"regexp"
	"time"
)

type TaskAPI struct {
	Data *db.DB
}

func (self TaskAPI) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch {
	case match(req, `GET /v1/tasks`):
		tasks, err := self.Data.GetAllAnnotatedTasks(
			&db.TaskFilter{
				ForStatus: paramValue(req, "status", ""),
			},
		)
		if err != nil {
			bail(w, err)
			return
		}

		JSON(w, tasks)
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

	w.WriteHeader(415)
	return
}
