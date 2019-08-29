package bus

import (
	"fmt"
	"sync"

	"github.com/jhunt/go-log"
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
	//slotMap maps unique identifiers given to clients to slot slice indices
	slotMap map[int64]int
	backlog int

	lifetime, current, dropped struct {
		connections int64
	}
	events   map[string]int64
	messages map[string]int64
}

type slot struct {
	ch         chan Event
	id         int64
	mostQueued int
	acl        map[string]bool
}

func New(n, backlog int) *Bus {
	b := Bus{
		slots:    make([]slot, n),
		slotMap:  make(map[int64]int, n),
		backlog:  backlog,
		events:   make(map[string]int64),
		messages: make(map[string]int64),
	}
	return &b
}

func (b *Bus) Register(queues []string) (chan Event, int64, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	for i := range b.slots {
		if b.slots[i].ch == nil {
			b.current.connections += 1
			b.lifetime.connections += 1
			b.slots[i].id = b.lifetime.connections
			b.slotMap[b.lifetime.connections] = i

			b.slots[i].ch = make(chan Event, b.backlog)
			b.slots[i].acl = make(map[string]bool)
			for _, q := range queues {
				b.slots[i].acl[q] = true
			}

			return b.slots[i].ch, b.slots[i].id, nil
		}
	}

	return nil, -1, fmt.Errorf("too many message bus clients")
}

//Unregister causes the bus to stop routing events to the handler with the given ID.
// The channel returned from the matching call to register is closed. Multiple calls
// to Unregister with the same id are idempotent.
func (b *Bus) Unregister(id int64) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.unregister(id)
}

func (b *Bus) unregister(id int64) {
	idx, found := b.slotMap[id]
	if !found {
		return //already closed. moving on.
	}

	delete(b.slotMap, id)
	if idx < 0 || idx >= len(b.slots) {
		log.Errorf("could not unregister channel #%d: index out of range", idx)
		panic(fmt.Sprintf("could not unregister channel #%d: index out of range", idx))
	}

	b.current.connections -= 1
	close(b.slots[idx].ch)
	b.slots[idx].ch = nil
	b.slots[idx].acl = nil
	b.slots[idx].mostQueued = 0
	b.slots[idx].id = -1
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
	for i, s := range b.slots {
		if s.ch == nil {
			continue
		}

		for _, q := range queues {
			if q == "*" || s.acl[q] {
				ev.Queue = q
				select {
				case s.ch <- ev:
					queued := len(s.ch)
					if queued > s.mostQueued {
						b.slots[i].mostQueued = queued
					}
					b.messages[ev.Event] += 1
				default:
					b.unregister(s.id)
					b.dropped.connections++
				}
				break
			}
		}
	}
}
