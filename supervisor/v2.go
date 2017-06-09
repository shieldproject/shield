package supervisor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		dst.Jobs[j].OK = true /* FIXME */
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
		system.Archives[i].Size = -1

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
			system.Archives[i].Notes = tasks[0].Notes
		} else if len(tasks) > 1 {
			bail(w, fmt.Errorf("multiple tasks associated with archive UUID %s", archive.UUID))
			return
		}
	}

	failed, err := v2.Data.GetAllTasks(
		&db.TaskFilter{
			ForTarget: target.UUID.String(),
			ForOp:     "backup",
			ForStatus: "failed",
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

		default:
			bailWithError(w, ClientErrorf("invalid system annotation type '%s'", ann.Type))
			return
		}
	}

	JSONLiteral(w, `{"ok":"annotated"}`)
	return
}

func (v2 V2API) DeleteSystem(w http.ResponseWriter, req *http.Request) {
}
