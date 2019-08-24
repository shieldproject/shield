package bus

type Metrics struct {
	Connections struct {
		Lifetime int64 `json:"lifetime"`
		Current  int64 `json:"current"`
	} `json:"connections"`
	Events   map[string]int64 `json:"events"`
	Messages map[string]int64 `json:"messages"`

	Slots [][]string `json:"clients"`
}

func (b *Bus) DumpState() Metrics {
	b.lock.Lock()
	defer b.lock.Unlock()

	var m Metrics
	m.Connections.Lifetime = b.lifetime.connections
	m.Connections.Current = b.current.connections

	m.Events = make(map[string]int64)
	for t, n := range b.events {
		m.Events[t] = n
	}

	m.Messages = make(map[string]int64)
	for t, n := range b.messages {
		m.Messages[t] = n
	}

	m.Slots = make([][]string, len(b.slots))
	for i := range b.slots {
		m.Slots[i] = make([]string, 0)
		for q := range b.slots[i].acl {
			m.Slots[i] = append(m.Slots[i], q)
		}
	}

	return m
}
