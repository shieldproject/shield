package scheduler

import (
	"fmt"
	"sync"
	"sync/atomic"

	log "github.com/shieldproject/shield/internal/log"

	"github.com/shieldproject/shield/db"
)

var serial = 0

type Worker struct {
	id        int
	available atomic.Bool
	mu        sync.Mutex
	task      string
	last      int
	db        *db.DB
}

func NewWorker(db *db.DB) *Worker {
	serial += 1
	w := &Worker{
		id: serial,
		db: db,
	}
	w.available.Store(true)
	return w
}

func (t *Worker) String() string {
	return fmt.Sprintf("worker t#%03d", t.id)
}

func (t *Worker) Available() bool {
	return t.available.Load()
}

func (t *Worker) Reserve(task string) {
	log.Infof("reserving %s...", t)
	t.mu.Lock()
	t.task = task
	t.mu.Unlock()
	t.available.Store(false)
}

func (t *Worker) Release() {
	log.Infof("releasing %s...", t)
	t.mu.Lock()
	t.task = ""
	t.mu.Unlock()
	t.available.Store(true)
}
