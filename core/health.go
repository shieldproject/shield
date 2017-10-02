package core

import (
	"os"

	"github.com/pborman/uuid"
)

type StorageHealth struct {
	UUID    uuid.UUID `json:"uuid"`
	Name    string    `json:"name"`
	Healthy bool      `json:"healthy"`
}
type JobHealth struct {
	UUID    uuid.UUID `json:"uuid"`
	Target  string    `json:"target"`
	Job     string    `json:"job"`
	Healthy bool      `json:"healthy"`
}
type Health struct {
	SHIELD struct {
		Version string `json:"version"`
		IP      string `json:"ip"`
		FQDN    string `json:"fqdn"`
		Env     string `json:"env"`
		Color   string `json:"color"`
	} `json:"shield"`
	Health struct {
		Core    string `json:"core"`
		Storage bool   `json:"storage_ok"`
		Jobs    bool   `json:"jobs_ok"`
	} `json:"health"`

	Storage []StorageHealth `json:"storage"`
	Jobs    []JobHealth     `json:"jobs"`

	Stats struct {
		Jobs     int `json:"jobs"`
		Systems  int `json:"systems"`
		Archives int `json:"archives"`
		Storage  int `json:"storage"`
		Daily    int `json:"daily"`
	} `json:"stats"`
}

func (core *Core) checkHealth() (Health, error) {
	var health Health

	health.SHIELD.Version = Version
	health.SHIELD.Env = os.Getenv("SHIELD_NAME")
	health.SHIELD.IP = core.ip
	health.SHIELD.FQDN = core.fqdn

	health.Health.Storage = true
	stores, err := core.DB.GetAllStores(nil)
	if err != nil {
		return health, err
	}
	health.Storage = make([]StorageHealth, len(stores))
	for i, store := range stores {
		health.Storage[i].UUID = store.UUID
		health.Storage[i].Name = store.Name
		health.Storage[i].Healthy = true // FIXME
		if !health.Storage[i].Healthy {
			health.Health.Storage = false
		}
	}

	health.Health.Jobs = true
	jobs, err := core.DB.GetAllJobs(nil)
	if err != nil {
		return health, err
	}
	health.Jobs = make([]JobHealth, len(jobs))
	for i, job := range jobs {
		health.Jobs[i].UUID = job.UUID
		health.Jobs[i].Target = job.TargetName
		health.Jobs[i].Job = job.Name
		health.Jobs[i].Healthy = job.Healthy()

		if !health.Jobs[i].Healthy {
			health.Health.Jobs = false
		}
	}
	health.Stats.Jobs = len(jobs)

	if health.Health.Core, err = core.vault.Status(); err != nil {
		return health, err
	}

	if health.Stats.Systems, err = core.DB.CountTargets(nil); err != nil {
		return health, err
	}

	if health.Stats.Archives, err = core.DB.CountArchives(nil); err != nil {
		return health, err
	}

	if health.Stats.Storage, err = core.DB.ArchiveStorageFootprint(nil); err != nil {
		return health, err
	}

	health.Stats.Daily = 0 // FIXME
	return health, nil
}
