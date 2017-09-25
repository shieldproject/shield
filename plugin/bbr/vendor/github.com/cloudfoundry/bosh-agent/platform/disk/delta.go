package disk

func withinDelta(existing, expected, delta uint64) bool {
	switch {
	case existing > expected:
		return (existing - expected) <= delta
	case expected > existing:
		return (expected - existing) <= delta
	}
	return true
}
