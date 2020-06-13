package core

import (
	"fmt"

	"github.com/shieldproject/shield/db"
)

type JobHealth struct {
	UUID    string `json:"uuid"`
	Target  string `json:"target"`
	Job     string `json:"job"`
	Healthy bool   `json:"healthy"`
}
type Health struct {
	Health struct {
		Storage bool `json:"storage_ok"`
		Jobs    bool `json:"jobs_ok"`
	} `json:"health"`

	Jobs []JobHealth `json:"jobs"`

	Stats struct {
		Jobs     int `json:"jobs"`
		Systems  int `json:"systems"`
		Archives int `json:"archives"`
	} `json:"stats"`
}

func (c *Core) checkHealth() (Health, error) {
	var health Health
	health.Health.Jobs = true
	jobs, err := c.db.GetAllJobs(nil)
	if err != nil {
		return health, fmt.Errorf("failed to retrieve all jobs: %s", err)
	}
	health.Jobs = make([]JobHealth, len(jobs))
	for i, job := range jobs {
		health.Jobs[i].UUID = job.UUID
		health.Jobs[i].Target = job.Target.Name
		health.Jobs[i].Job = job.Name
		health.Jobs[i].Healthy = job.Healthy

		if !health.Jobs[i].Healthy {
			health.Health.Jobs = false
		}
	}
	health.Stats.Jobs = len(jobs)

	if health.Stats.Systems, err = c.db.CountTargets(nil); err != nil {
		return health, fmt.Errorf("failed to count systems/targets: %s", err)
	}

	if health.Stats.Archives, err = c.db.CountArchives(&db.ArchiveFilter{
		WithStatus: []string{"valid"},
	}); err != nil {
		return health, fmt.Errorf("failed to retrieve count of valid archives: %s", err)
	}

	return health, nil
}
