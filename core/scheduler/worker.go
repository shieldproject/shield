package scheduler

import (
	"fmt"

	"github.com/jhunt/go-log"

	"github.com/starkandwayne/shield/db"
)

var serial = 0

type Worker struct {
	id        int
	available bool
	task      string
	last      int
	db        *db.DB
}

func NewWorker(db *db.DB) *Worker {
	serial += 1
	return &Worker{
		id:        serial,
		available: true,
		db:        db,
	}
}

func (t Worker) String() string {
	return fmt.Sprintf("worker t#%03d", t.id)
}

func (t Worker) Available() bool {
	return t.available
}

func (t *Worker) Reserve(task string) {
	log.Infof("reserving %s...", t)
	t.available = false
	t.task = task
}

func (t *Worker) Release() {
	log.Infof("releasing %s...", t)
	t.available = true
	t.task = ""
}
