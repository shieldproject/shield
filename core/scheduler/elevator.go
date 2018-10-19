package scheduler

import (
	"time"
)

func (s *Scheduler) Elevator(deadline int) {
	t := time.NewTicker(time.Duration(deadline) * time.Second)
	for range t.C {
		s.Elevate()
	}
}

func (s *Scheduler) Elevate() {
	s.lock.Lock()
	defer s.lock.Unlock()

	old0 := s.chores[0]
	s.chores[0] = nil
	last := 0
	for i := range s.chores {
		/* skip priority 0, we handle it specially */
		if i == 0 {
			continue
		}

		/* skip empty priority queue slots */
		if len(s.chores[i]) == 0 {
			continue
		}

		/* swap to last priority */
		s.chores[last] = s.chores[i]
		s.chores[i] = nil
		last = i
	}

	/* merge previous top-priority chores onto
	   the back of the new top-priority chores */
	for _, chore := range old0 {
		s.chores[0] = append(s.chores[0], chore)
	}
}
