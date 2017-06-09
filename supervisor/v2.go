package supervisor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"

	"github.com/pborman/uuid"

	"github.com/starkandwayne/shield/db"
	"github.com/starkandwayne/shield/timespec"
)

type V2API struct {
	Data       *db.DB
	ResyncChan chan int
	Tasks      chan *db.Task
}

func (v2 V2API) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch {
	case match(req, `GET /v2/health`):
		v2.GetHealth(w, req)

	case match(req, `GET /v2/systems`):
		v2.GetSystems(w, req)
	case match(req, `POST /v2/systems`):
		v2.PostSystem(w, req)
	case match(req, `GET /v2/systems/:uuid`):
		v2.GetSystem(w, req)
	case match(req, `PUT /v2/systems/:uuid`):
		v2.PutSystem(w, req)
	case match(req, `PATCH /v2/systems/:uuid`):
		v2.PatchSystem(w, req)
	case match(req, `DELETE /v2/systems/:uuid`):
		v2.DeleteSystem(w, req)
	}

	w.WriteHeader(501)
	return
}

/*

  GET /v2/health

  Returns health information about the SHIELD core,
  connected storage accounts, and general metrics.

  {
    "shield": {
      "version" : "6.7.2",
      "ip"      : "10.0.0.5",
      "fqdn"    : "shield.example.com",
      "env"     : "PRODUCTION",
      "color"   : ""
    },
    "health": {
      "api_ok"     : true,
      "storage_ok" : true,
      "jobs_ok"    : true
    },
    "storage": [
      { "name": "s3", "healthy": true },
      { "name": "fs", "healthy": true } ],
    "jobs": [
      { "target": "BOSH DB", "job": "daily",  "healthy": true },
      { "target": "BOSH DB", "job": "weekly", "healthy": true } ],
    "stats": {
      "jobs"    : 8,
      "systems" : 7,
      "archives": 124,
      "storage" : 243567112,
      "daily"   : 12345000
    }
  }

*/
type v2StorageHealth struct {
	UUID    uuid.UUID `json:"uuid"`
	Name    string    `json:"name"`
	Healthy bool      `json:"healthy"`
}
type v2JobHealth struct {
	UUID    uuid.UUID `json:"uuid"`
	Target  string    `json:"target"`
	Job     string    `json:"job"`
	Healthy bool      `json:"healthy"`
}
type v2Health struct {
	SHIELD struct {
		Version string `json:"version"`
		IP      string `json:"ip"`
		FQDN    string `json:"fqdn"`
		Env     string `json:"env"`
		Color   string `json:"color"`
	} `json:"shield"`
	Health struct {
		API     bool `json:"api_ok"`
		Storage bool `json:"storage_ok"`
		Jobs    bool `json:"jobs_ok"`
	} `json:"health"`

	Storage []v2StorageHealth `json:"storage"`
	Jobs    []v2JobHealth     `json:"jobs"`

	Stats struct {
		Jobs     int `json:"jobs"`
		Systems  int `json:"systems"`
		Archives int `json:"archives"`
		Storage  int `json:"storage"`
		Daily    int `json:"daily"`
	} `json:"stats"`
}

func (v2 V2API) GetHealth(w http.ResponseWriter, req *http.Request) {
	var health v2Health

	health.Health.API = true
	health.SHIELD.Version = Version
	health.SHIELD.Env = os.Getenv("SHIELD_NAME")
	health.SHIELD.IP = "x.x.x.x"              // FIXME
	health.SHIELD.FQDN = "shield.example.com" // FIXME

	health.Health.Storage = true
	stores, err := v2.Data.GetAllStores(nil)
	if err != nil {
		bail(w, err)
		return
	}
	health.Storage = make([]v2StorageHealth, len(stores))
	for i, store := range stores {
		health.Storage[i].UUID = store.UUID
		health.Storage[i].Name = store.Name
		health.Storage[i].Healthy = true // FIXME
		if !health.Storage[i].Healthy {
			health.Health.Storage = false
		}
	}

	health.Health.Jobs = true
	jobs, err := v2.Data.GetAllJobs(nil)
	if err != nil {
		bail(w, err)
		return
	}
	health.Jobs = make([]v2JobHealth, len(jobs))
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

	if health.Stats.Systems, err = v2.Data.CountTargets(nil); err != nil {
		bail(w, err)
		return
	}

	if health.Stats.Archives, err = v2.Data.CountArchives(nil); err != nil {
		bail(w, err)
		return
	}

	if health.Stats.Storage, err = v2.Data.ArchiveStorageFootprint(nil); err != nil {
		bail(w, err)
		return
	}

	health.Stats.Daily = 0 // FIXME

	JSON(w, health)
}

/*

  GET /v2/systems

  Retrieves a list of all protected systems, their job configurations,
  recent archive metadata, and failed task metadata.  This endpoint is
  the be-all, end-all of interacting with targets + jobs under the new
  SHIELD UI.

  Response:
  [
    {
      "uuid"  : "93815474-126f-4934-aead-aaee29a34f3c",
      "name"  : "Important Database",
      "notes" : "This is the most important data we have",
      "ok"    : 1,

      "jobs": [
        {
          "schedule" : "daily 7am",
          "from"     : "Postgres",
          "to"       : "S3",
          "keep"     : { "n": 9, "days": 9 },
          "ok"       : true
        },
        {
          "schedule" : "weekly 9:30am",
          "from"     : "Postgres",
          "to"       : "S3",
          "keep"     : { "n": 12, "days": 90 },
          "ok"       : true
        }
      ]
    }
  ]
*/
type v2SystemArchive struct {
	Schedule string    `json:"schedule"`
	TakenAt  int64     `json:"taken_at"`
	UUID     uuid.UUID `json:"uuid"`
	TaskUUID uuid.UUID `json:"task_uuid"`
	Expiry   int       `json:"expiry"`
	Size     int       `json:"size"`
	OK       bool      `json:"ok"`
	Notes    string    `json:"notes"`
}
type v2SystemTask struct {
	UUID      uuid.UUID `json:"uuid"`
	Type      string    `json:"type"`
	Owner     string    `json:"owner"`
	StartedAt int64     `json:"started_at"`
	OK        bool      `json:"ok"`
	Notes     string    `json:"notes"`
}
type v2SystemJob struct {
	Schedule string `json:"schedule"`
	From     string `json:"from"`
	To       string `json:"to"`
	OK       bool   `json:"ok"`

	Keep struct {
		N    int `json:"n"`
		Days int `json:"days"`
	} `json:"keep"`
}
type v2System struct {
	UUID  uuid.UUID `json:"uuid"`
	Name  string    `json:"name"`
	Notes string    `json:"notes"`
	OK    bool      `json:"ok"`

	Jobs        []v2SystemJob     `json:"jobs"`
	Archives    []v2SystemArchive `json:"archives"`
	FailedTasks []v2SystemTask    `json:"failed_tasks"`
}

func (v2 V2API) copyTarget(dst *v2System, target *db.Target) error {
	dst.UUID = target.UUID
	dst.Name = target.Name
	dst.Notes = target.Summary
	dst.OK = true /* FIXME */

	jobs, err := v2.Data.GetAllJobs(
		&db.JobFilter{
			ForTarget: target.UUID.String(),
		},
	)
	if err != nil {
		return err
	}

	dst.Jobs = make([]v2SystemJob, len(jobs))
	for j, job := range jobs {
		dst.Jobs[j].Schedule = job.Schedule
		dst.Jobs[j].From = job.TargetPlugin
		dst.Jobs[j].To = job.StorePlugin
		dst.Jobs[j].OK = job.Healthy()
		if !job.Healthy() {
			dst.OK = false
		}

		dst.Jobs[j].Keep.Days = job.Expiry / 86400

		tspec, err := timespec.Parse(job.Schedule)
		if err != nil {
			return err
		}
		switch tspec.Interval {
		case timespec.Daily:
			dst.Jobs[j].Keep.N = dst.Jobs[j].Keep.Days
		case timespec.Weekly:
			dst.Jobs[j].Keep.N = dst.Jobs[j].Keep.Days / 7
		case timespec.Monthly:
			dst.Jobs[j].Keep.N = dst.Jobs[j].Keep.Days / 30
		}
	}
	return nil
}

func (v2 V2API) GetSystems(w http.ResponseWriter, req *http.Request) {
	targets, err := v2.Data.GetAllTargets(
		&db.TargetFilter{
			SkipUsed:   paramEquals(req, "unused", "t"),
			SkipUnused: paramEquals(req, "unused", "f"),
			SearchName: paramValue(req, "name", ""),
			ForPlugin:  paramValue(req, "plugin", ""),
			ExactMatch: paramEquals(req, "exact", "t"),
		},
	)
	if err != nil {
		bail(w, err)
		return
	}

	systems := make([]v2System, len(targets))
	for i, target := range targets {
		err := v2.copyTarget(&systems[i], target)
		if err != nil {
			bail(w, err)
			return
		}
	}

	JSON(w, systems)
	return
}

func (v2 V2API) GetSystem(w http.ResponseWriter, req *http.Request) {
	re := regexp.MustCompile(`/([a-fA-F0-9-]+)$`)
	id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

	target, err := v2.Data.GetTarget(id)
	if err != nil {
		bail(w, err)
		return
	}

	if target == nil {
		w.WriteHeader(404)
		return
	}

	var system v2System
	err = v2.copyTarget(&system, target)
	if err != nil {
		bail(w, err)
		return
	}

	archives, err := v2.Data.GetAllArchives(
		&db.ArchiveFilter{
			ForTarget:  target.UUID.String(),
			WithStatus: []string{"valid"},
		},
	)
	if err != nil {
		bail(w, err)
		return
	}

	system.Archives = make([]v2SystemArchive, len(archives))
	for i, archive := range archives {
		system.Archives[i].Schedule = archive.Job
		system.Archives[i].TakenAt = archive.TakenAt.Time().Unix()
		system.Archives[i].UUID = archive.UUID
		system.Archives[i].Expiry = (int)((archive.ExpiresAt.Time().Unix() - archive.TakenAt.Time().Unix()) / 86400)
		system.Archives[i].Notes = archive.Notes
		system.Archives[i].Size = -1 // FIXME

		tasks, err := v2.Data.GetAllTasks(
			&db.TaskFilter{
				ForArchive: archive.UUID.String(),
			},
		)
		if err != nil {
			bail(w, err)
			return
		}

		if len(tasks) == 1 {
			system.Archives[i].TaskUUID = tasks[0].UUID
			system.Archives[i].OK = tasks[0].OK
		} else if len(tasks) > 1 {
			bail(w, fmt.Errorf("multiple tasks associated with archive UUID %s", archive.UUID))
			return
		}
	}

	failed, err := v2.Data.GetAllTasks(
		&db.TaskFilter{
			ForTarget:    target.UUID.String(),
			ForOp:        "backup",
			ForStatus:    "failed",
			OnlyRelevant: true,
		},
	)
	if err != nil {
		bail(w, err)
		return
	}

	system.FailedTasks = make([]v2SystemTask, len(failed))
	for i, task := range failed {
		system.FailedTasks[i].UUID = task.UUID
		system.FailedTasks[i].Type = task.Op
		system.FailedTasks[i].Owner = task.Owner
		system.FailedTasks[i].StartedAt = task.StartedAt.Time().Unix()
		system.FailedTasks[i].OK = task.OK
		system.FailedTasks[i].Notes = task.Notes
	}

	JSON(w, system)
	return
}

func (v2 V2API) PostSystem(w http.ResponseWriter, req *http.Request) {
}

func (v2 V2API) PutSystem(w http.ResponseWriter, req *http.Request) {
}

type v2PatchAnnotation struct {
	Type        string `json:"type"`
	UUID        string `json:"uuid"`
	Disposition string `json:"disposition"`
	Notes       string `json:"notes"`
	Clear       string `json:"clear"`
}

func (v2 V2API) PatchSystem(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		w.WriteHeader(400)
		return
	}

	var params struct {
		Annotations []v2PatchAnnotation `json:"annotations"`
	}

	if err := json.NewDecoder(req.Body).Decode(&params); err != nil && err != io.EOF {
		bailWithError(w, ClientErrorf("bad JSON payload: %s", err))
		return
	}

	re := regexp.MustCompile(`^/v2/systems/([a-fA-F0-9-]+)`)
	id := uuid.Parse(re.FindStringSubmatch(req.URL.Path)[1])

	target, err := v2.Data.GetTarget(id)
	if err != nil {
		bail(w, err)
		return
	}

	for _, ann := range params.Annotations {
		switch ann.Type {
		case "task":
			err = v2.Data.AnnotateTargetTask(
				target.UUID,
				ann.UUID,
				&db.TaskAnnotation{
					Disposition: ann.Disposition,
					Notes:       ann.Notes,
					Clear:       ann.Clear,
				},
			)
			if err != nil {
				bail(w, err)
				return
			}

		case "archive":
			err = v2.Data.AnnotateTargetArchive(
				target.UUID,
				ann.UUID,
				ann.Notes,
			)
			if err != nil {
				bail(w, err)
				return
			}

		default:
			bailWithError(w, ClientErrorf("invalid system annotation type '%s'", ann.Type))
			return
		}
	}

	_ = v2.Data.MarkTasksIrrelevant()

	JSONLiteral(w, `{"ok":"annotated"}`)
	return
}

func (v2 V2API) DeleteSystem(w http.ResponseWriter, req *http.Request) {
}
