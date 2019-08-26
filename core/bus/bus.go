package bus

import (
	"fmt"
	"sync"
)

const (
	ErrorEvent            = "error"
	UnlockCoreEvent       = "unlock-core"
	CreateObjectEvent     = "create-object"
	UpdateObjectEvent     = "update-object"
	DeleteObjectEvent     = "delete-object"
	TaskStatusUpdateEvent = "task-status-update"
	TaskLogUpdateEvent    = "task-log-update"
	TenantInviteEvent     = "tenant-invite"
	TenantBanishEvent     = "tenant-banish"
)

type Event struct {
	Event string      `json:"event"`
	Queue string      `json:"queue"`
	Type  string      `json:"type,omitempty"`
	Data  interface{} `json:"data"`
}

type Bus struct {
	lock  sync.Mutex
	slots []slot

	lifetime, current struct {
		connections int64
	}
	events   map[string]int64
	messages map[string]int64
}

type slot struct {
	ch  chan Event
	acl map[string]bool
}

func New(n int) *Bus {
	b := Bus{
		slots: make([]slot, n),
	}
	b.events = make(map[string]int64)
	b.messages = make(map[string]int64)
	return &b
}

func (b *Bus) Register(queues []string) (chan Event, int, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	for i := range b.slots {
		if b.slots[i].ch == nil {
			b.lifetime.connections += 1
			b.current.connections += 1
			b.slots[i].ch = make(chan Event, 0)
			b.slots[i].acl = make(map[string]bool)
			for _, q := range queues {
				b.slots[i].acl[q] = true
			}

			return b.slots[i].ch, i, nil
		}
	}

	return nil, -1, fmt.Errorf("too many message bus clients")
}

func (b *Bus) Unregister(idx int) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if idx < 0 || idx >= len(b.slots) {
		return fmt.Errorf("could not unregister channel #%d: index out of range", idx)
	}

	b.current.connections -= 1
	ch := b.slots[idx].ch
	b.slots[idx].ch = nil
	b.slots[idx].acl = nil

	func () {
		defer recover()
		close(ch)
	}()
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

	if _, ok := b.events[ev.Event]; !ok {
		b.events[ev.Event] = 0
	}
	if _, ok := b.messages[ev.Event]; !ok {
		b.messages[ev.Event] = 0
	}

	b.events[ev.Event] += 1
	for _, s := range b.slots {
		if s.ch == nil {
			continue
		}

		func() {
			for _, q := range queues {
				if q == "*" {
					ev.Queue = q
					s.ch <- ev
					b.messages[ev.Event] += 1
					return
				}
			}
			for _, q := range queues {
				if _, ok := s.acl[q]; ok {
					ev.Queue = q
					s.ch <- ev
					b.messages[ev.Event] += 1
					return
				}
			}
		}()
	}
}
