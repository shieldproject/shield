package scheduler

type BacklogStatus struct {
	Priority int    `json:"priority"`
	Position int    `json:"position"`
	TaskUUID string `json:"task_uuid"`
}

type WorkerStatus struct {
	ID       int    `json:"id"`
	Idle     bool   `json:"idle"`
	TaskUUID string `json:"task_uuid"`
	LastSeen int    `json:"last_seen"`
}

type Status struct {
	Backlog []BacklogStatus `json:"backlog"`
	Workers []WorkerStatus  `json:"workers"`
}

func (s *Scheduler) Status() Status {
	status := Status{
		Workers: make([]WorkerStatus, len(s.workers)),
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	for i, w := range s.workers {
		status.Workers[i].ID = w.id
		status.Workers[i].Idle = w.available
		status.Workers[i].TaskUUID = w.task
		status.Workers[i].LastSeen = w.last
	}

	for prio, lst := range s.chores {
		for i, chore := range lst {
			status.Backlog = append(status.Backlog, BacklogStatus{
				Priority: prio,
				Position: i,
				TaskUUID: chore.TaskUUID,
			})
		}
	}

	return status
}
