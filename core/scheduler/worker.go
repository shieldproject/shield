package scheduler

import (
	"fmt"

	"github.com/jhunt/go-log"
)

var serial = 0

type Worker struct {
	id        int
	available bool
	task      string
	last      int
}

func NewWorker() Worker {
	serial += 1
	return Worker{
		id:        serial,
		available: true,
	}
}

func (t Worker) String() string {
	return fmt.Sprintf("worker t#%03d", t.id)
}

func (t Worker) Available() bool {
	return t.available
}

func (t *Worker) Reserve() {
	log.Infof("reserving %s...", t)
	t.available = false
}

func (t *Worker) Release() {
	log.Infof("releasing %s...", t)
	t.available = true
}
