package core

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jhunt/go-log"
	"github.com/starkandwayne/shield/db"
)

type Event struct {
	Task *db.Task
	JSON []byte
}

type Broadcaster struct {
	lock  sync.Mutex
	chans []chan Event
}

func NewBroadcaster(slots int) Broadcaster {
	return Broadcaster{
		chans: make([]chan Event, slots),
	}
}

func (b *Broadcaster) Register(ch chan Event) (int, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	for i, slot := range b.chans {
		if slot == nil {
			b.chans[i] = ch
			return i, nil
		}
	}
	return -1, fmt.Errorf("out of broadcaster slots")
}

func (b *Broadcaster) Unregister(idx int) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if idx < 0 || idx >= len(b.chans) {
		return fmt.Errorf("broadcast receive index out of range")
	}

	ch := b.chans[idx]
	b.chans[idx] = nil
	for _ = range ch {
	}

	return nil
}

func (b *Broadcaster) Broadcast(ev Event) {
	b.lock.Lock()
	defer b.lock.Unlock()

	for _, ch := range b.chans {
		if ch != nil {
			ch <- ev
		}
	}
}

func (core *Core) broadcastTask(task *db.Task) {
	task, err := core.DB.GetTask(task.UUID)
	if err != nil {
		log.Errorf("unable to broadcast task update for %s, failed to refresh task from database: %s", task.UUID, err)
		return
	}

	b, err := json.Marshal(task)
	if err != nil {
		log.Errorf("unable to broadcast task update for %s, json marshalling failed: %s", task.UUID, err)
		return
	}

	log.Debugf("broadcasting event for task %s (tenant %s) json: %s", task.UUID, task.TenantUUID, b)
	core.events <- Event{
		Task: task,
		JSON: b,
	}
	log.Debugf("broadcast complete")
}

func (core *Core) startTask(task *db.Task) {
	log.Debugf("starting task %s", task.UUID)
	if err := core.DB.StartTask(task.UUID, time.Now()); err != nil {
		log.Errorf("  %s: !! failed to update database: %s", task.UUID, err)
	}
	core.broadcastTask(task)
}

func (core *Core) finishTask(task *db.Task) {
	log.Debugf("finishing task %s", task.UUID)
	if err := core.DB.CompleteTask(task.UUID, time.Now()); err != nil {
		log.Errorf("  %s: !! failed to update database: %s", task.UUID, err)
	}
	core.broadcastTask(task)
}

func (core *Core) failTask(task *db.Task, msg string, args ...interface{}) {
	if msg != "" {
		err := core.DB.UpdateTaskLog(task.UUID, fmt.Sprintf("TASK FAILED!!  "+msg, args...))
		if err != nil {
			log.Errorf("  %s: !! failed to update database: %s", task.UUID, err)
		}
	}

	log.Warnf("  %s: task failed!", task.UUID)
	if err := core.DB.FailTask(task.UUID, time.Now()); err != nil {
		log.Errorf("  %s: !! failed to update database: %s", task.UUID, err)
	}
	core.broadcastTask(task)
}

func (core *Core) logToTask(task *db.Task, msg string, args ...interface{}) {
	log.Debugf("appending to log of task %s", task.UUID)
	s := msg
	if len(args) > 0 {
		s = fmt.Sprintf(msg, args...)
	}

	log.Infof("  %s> %s", task.UUID, strings.Trim(s, "\n"))
	if err := core.DB.UpdateTaskLog(task.UUID, s); err != nil {
		log.Errorf("  %s: !! failed to update database: %s", task.UUID, err)
	}
	core.broadcastTask(task)
}
