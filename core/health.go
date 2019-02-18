package core

import (
	"fmt"

	"github.com/jhunt/go-log"

	"github.com/starkandwayne/shield/db"
)

type StorageHealth struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Healthy bool   `json:"healthy"`
}
type JobHealth struct {
	UUID    string `json:"uuid"`
	Target  string `json:"target"`
	Job     string `json:"job"`
	Healthy bool   `json:"healthy"`
}
type Health struct {
	Health struct {
		Core    string `json:"core"`
		Storage bool   `json:"storage_ok"`
		Jobs    bool   `json:"jobs_ok"`
	} `json:"health"`

	Storage []StorageHealth `json:"storage"`
	Jobs    []JobHealth     `json:"jobs"`

	Stats struct {
		Jobs     int   `json:"jobs"`
		Systems  int   `json:"systems"`
		Archives int   `json:"archives"`
		Storage  int64 `json:"storage"`
		Daily    int64 `json:"daily"`
	} `json:"stats"`
}

func (c *Core) checkHealth() (Health, error) {
	var health Health

	stores, err := c.db.GetAllStores(nil)
	if err != nil {
		log.Errorf("Failed to get stores for health tests: %s", err)
	}
	health.Health.Storage = true
	for _, store := range stores {
		if !store.Healthy {
			health.Health.Storage = false
		}
	}

	if err != nil {
		return health, fmt.Errorf("failed to retrieve all stores: %s", err)
	}
	health.Storage = make([]StorageHealth, len(stores))
	for i, store := range stores {
		health.Storage[i].UUID = store.UUID
		health.Storage[i].Name = store.Name
		health.Storage[i].Healthy = store.Healthy
		if !health.Storage[i].Healthy {
			health.Health.Storage = false
		}
	}

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

	if health.Health.Core, err = c.vault.Status(); err != nil {
		return health, fmt.Errorf("failed to retrieve vault status: %s", err)
	}

	if health.Stats.Systems, err = c.db.CountTargets(nil); err != nil {
		return health, fmt.Errorf("failed to count systems/targets: %s", err)
	}

	if health.Stats.Archives, err = c.db.CountArchives(&db.ArchiveFilter{
		WithStatus: []string{"valid"},
	}); err != nil {
		return health, fmt.Errorf("failed to retrieve count of valid archives: %s", err)
	}

	if health.Stats.Storage, err = c.db.ArchiveStorageFootprint(&db.ArchiveFilter{
		WithStatus: []string{"valid"},
	}); err != nil {
		return health, fmt.Errorf("failed to calcualte storage footprint: %s", err)
	}

	health.Stats.Daily = 0 // FIXME
	return health, nil
}

func (c *Core) checkTenantHealth(tenantUUID string) (Health, error) {
	var health Health
	health.Health.Storage = true
	stores, err := c.db.GetAllStores(&db.StoreFilter{
		ForTenant: tenantUUID,
	})
	if err != nil {
		return health, err
	}
	health.Storage = make([]StorageHealth, len(stores))
	for i, store := range stores {
		health.Storage[i].UUID = store.UUID
		health.Storage[i].Name = store.Name
		health.Storage[i].Healthy = store.Healthy
		if !health.Storage[i].Healthy {
			health.Health.Storage = false
		}
	}

	health.Health.Jobs = true
	jobs, err := c.db.GetAllJobs(&db.JobFilter{
		ForTenant: tenantUUID,
	})
	if err != nil {
		return health, err
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

	if health.Stats.Systems, err = c.db.CountTargets(&db.TargetFilter{
		ForTenant: tenantUUID,
	}); err != nil {
		return health, err
	}

	if health.Health.Core, err = c.vault.Status(); err != nil {
		return health, err
	}

	tenant, err := c.db.GetTenant(tenantUUID)
	if err != nil {
		return health, err
	}
	if tenant == nil {
		return health, nil
	}

	health.Stats.Archives = tenant.ArchiveCount
	health.Stats.Storage = tenant.StorageUsed
	health.Stats.Daily = tenant.DailyIncrease

	return health, nil
}
