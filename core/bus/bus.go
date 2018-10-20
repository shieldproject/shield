package bus

import (
	"fmt"
	"sync"
)

const (
	NewObjectEvent        = "new-object"
	UpdateObjectEvent     = "update-object"
	DeleteObjectEvent     = "delete-object"
	TaskStatusUpdateEvent = "task-status-update"
	TaskLogUpdateEvent    = "task-log-update"
)

type Event struct {
	Type   string      `json:"type"`
	Tenant string      `json:"tenant"`
	Data   interface{} `json:"data"`
}

type Bus struct {
	lock  sync.Mutex
	chans []chan Event
}

func New(n int) *Bus {
	return &Bus{
		chans: make([]chan Event, n),
	}
}

func (b *Bus) Register() (chan Event, int, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	for i, slot := range b.chans {
		if slot == nil {
			b.chans[i] = make(chan Event, 0)
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

func (b *Bus) Send(typ, tenant string, thing interface{}) error {
	data, err := reflectOn(thing)
	if err != nil {
		return err
	}

	b.SendEvent(Event{
		Type:   typ,
		Tenant: tenant,
		Data:   data,
	})
	return nil
}

func (b *Bus) SendEvent(ev Event) {
	b.lock.Lock()
	defer b.lock.Unlock()

	for _, ch := range b.chans {
		if ch != nil {
			ch <- ev
		}
	}
}
