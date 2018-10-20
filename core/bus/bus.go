package bus

import (
	"fmt"
	"sync"
)

const (
	ErrorEvent            = "error"
	HeartbeatEvent        = "heartbeat"
	CreateObjectEvent     = "create-object"
	UpdateObjectEvent     = "update-object"
	DeleteObjectEvent     = "delete-object"
	TaskStatusUpdateEvent = "task-status-update"
	TaskLogUpdateEvent    = "task-log-update"
)

type Event struct {
	Event string      `json:"event"`
	Type  string      `json:"type,omitempty"`
	Data  interface{} `json:"data"`
}

type Bus struct {
	lock  sync.Mutex
	chans []chan Event

	lastHeartbeatEvent *Event
}

func New(n int) *Bus {
	return &Bus{
		chans: make([]chan Event, n),
	}
}

func catchup(ch chan Event, events ...Event) {
	for _, ev := range events {
		ch <- ev
	}
}

func (b *Bus) Register() (chan Event, int, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	for i, slot := range b.chans {
		if slot == nil {
			b.chans[i] = make(chan Event, 0)

			if b.lastHeartbeatEvent != nil {
				go catchup(b.chans[i], *b.lastHeartbeatEvent)
			}

			return b.chans[i], i, nil
		}
	}

	return nil, -1, fmt.Errorf("too many message bus clients")
}

func (b *Bus) Unregister(idx int) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if idx < 0 || idx >= len(b.chans) {
		return fmt.Errorf("could not unregister channel #%d: index out of range", idx)
	}

	ch := b.chans[idx]
	b.chans[idx] = nil
	for range ch {
	}

	return nil
}

func (b *Bus) SendError(err error, queues ...string) {
	b.SendEvent(queues, Event{
		Event: ErrorEvent,
		Data:  map[string]interface{}{"error": err},
	})
}

func (b *Bus) Send(event, typ string, thing interface{}, queues ...string) {
	b.SendEvent(queues, Event{
		Event: event,
		Type:  typ,
		Data:  marshal(thing),
	})
}

func (b *Bus) SendEvent(queues []string, ev Event) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if ev.Event == HeartbeatEvent {
		b.lastHeartbeatEvent = &ev
	}

	for _, ch := range b.chans {
		/* FIXME: acls on bus; only send the message once, and only if queues match up */
		if ch != nil {
			ch <- ev
		}
	}
}
