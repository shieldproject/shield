package scheduler

import (
	"fmt"
)

var serial = 0

type Worker struct {
	id        int
	available bool
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
	t.available = false
}

func (t *Worker) Release() {
	t.available = true
}
