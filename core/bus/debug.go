package bus

func (b *Bus) DumpState() [][]string {
	b.lock.Lock()
	defer b.lock.Unlock()

	slots := make([][]string, len(b.slots))
	for i := range slots {
		slots[i] = make([]string, 0)
		for q := range b.slots[i].acl {
			slots[i] = append(slots[i], q)
		}
	}

	return slots
}
