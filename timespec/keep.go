package timespec

func (s *Spec) KeepN(days int) int {
	switch s.Interval {
	default:
		return -1
	case Minutely:
		return days * 1440 / int(s.Cardinality)
	case Hourly:
		if s.Cardinality == 0 {
			return days * 24
		} else {
			return days * 24 / int(s.Cardinality)
		}
	case Daily:
		return days
	case Weekly:
		return days / 7
	case Monthly:
		return days / 30
	}
}
