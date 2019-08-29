package bus

import "sort"

type Metrics struct {
	Configuration struct {
		MaxSlots int `json:"max_slots"`
		Backlog  int `json:"backlog"`
	} `json:"configuration"`
	Connections struct {
		Lifetime int64 `json:"lifetime"`
		Current  int64 `json:"current"`
		Dropped  int64 `json:"dropped"`
	} `json:"connections"`
	Events   map[string]int64 `json:"events"`
	Messages map[string]int64 `json:"messages"`

	Slots []MetricSlot `json:"clients"`
}

type MetricSlot struct {
	ID         int64    `json:"id"`
	Queued     int      `json:"queued"`
	MostQueued int      `json:"most_queued"`
	Index      int      `json:"index"`
	ACLs       []string `json:"acls"`
}

func (b *Bus) DumpState() Metrics {
	b.lock.Lock()
	defer b.lock.Unlock()

	var m Metrics
	m.Configuration.MaxSlots = len(b.slots)
	m.Configuration.Backlog = b.backlog
	m.Connections.Lifetime = b.lifetime.connections
	m.Connections.Current = b.current.connections
	m.Connections.Dropped = b.dropped.connections

	m.Events = make(map[string]int64)
	for t, n := range b.events {
		m.Events[t] = n
	}

	m.Messages = make(map[string]int64)
	for t, n := range b.messages {
		m.Messages[t] = n
	}

	m.Slots = []MetricSlot{}
	for i := range b.slots {
		if b.slots[i].ch == nil {
			continue
		}
		m.Slots = append(m.Slots, MetricSlot{
			Index:      i,
			ID:         b.slots[i].id,
			Queued:     len(b.slots[i].ch),
			MostQueued: b.slots[i].mostQueued,
			ACLs:       make([]string, 0, len(b.slots[i].acl)),
		})
		for q := range b.slots[i].acl {
			m.Slots[i].ACLs = append(m.Slots[i].ACLs, q)
		}

		sort.Strings(m.Slots[i].ACLs)
	}

	return m
}
