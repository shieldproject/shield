package main

import (
	//"encoding/json"
	"fmt"
	"os"

	"github.com/pborman/uuid"

	. "github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

type ListJobOptions struct {
	Unpaused  bool
	Paused    bool
	Target    string
	Store     string
	Schedule  string
	Retention string
	UUID      string
}

func ListJobs(opts ListJobOptions) error {
	jobs, err := GetJobs(JobFilter{
		Paused:    MaybeBools(opts.Unpaused, opts.Paused),
		Target:    opts.Target,
		Store:     opts.Store,
		Schedule:  opts.Schedule,
		Retention: opts.Retention,
	})
	if err != nil {
		return fmt.Errorf("\nERROR: Unexpected arguments following command: %v\n", err)
	}
	t := tui.NewTable("UUID", "P?", "Name", "Description", "Retention Policy", "Schedule", "Target", "Agent")
	for _, job := range jobs {
		paused := "-"
		if job.Paused {
			paused = "Y"
		}

		if len(opts.UUID) > 0 && opts.UUID == job.UUID {
			t.Row(job.UUID, paused, job.Name, job.Summary,
				job.RetentionName, job.ScheduleName, job.TargetEndpoint, job.Agent)
			break
		} else if len(opts.UUID) > 0 && opts.UUID != job.UUID {
			continue
		}
		t.Row(job.UUID, paused, job.Name, job.Summary,
			job.RetentionName, job.ScheduleName, job.TargetEndpoint, job.Agent)
	}
	t.Output(os.Stdout)
	return nil
}

func PauseUnpauseJob(p bool, u string) error {
	if p {
		err := PauseJob(uuid.Parse(u))
		if err != nil {
			return fmt.Errorf("ERROR: Could not pause job '%s': %s", u, err)
		}
		fmt.Fprintf(os.Stdout, "Successfully paused job '%s'\n", u)
	} else {
		err := UnpauseJob(uuid.Parse(u))
		if err != nil {
			return fmt.Errorf("ERROR: Could not unpause job '%s': %s", u, err)
		}
		fmt.Fprintf(os.Stdout, "Successfully unpaused job '%s'\n", u)
	}
	return nil
}

func CreateNewJob() error {
	content := invokeEditor(`{
	"name"      : "Empty Job",
	"summary"   : "a short description",

	"store"     : "StoreUUID",
	"target"    : "TargetUUID",
	"retention" : "PolicyUUID",
	"schedule"  : "ScheduleUUID",
	"paused"    : false
	}`)

	newJob, err := CreateJob(content)
	fmt.Printf("newJob: %v\n", newJob)
	if err != nil {
		return fmt.Errorf("ERROR: Could not create new job: %s", err)
	}

	fmt.Fprintf(os.Stdout, "Created new job.\n")
	t := tui.NewTable("UUID", "P?", "Name", "Description", "Retention Policy", "Schedule", "Target", "Agent")
	paused := "-"
	if newJob.Paused {
		paused = "Y"
	}
	t.Row(newJob.UUID, paused, newJob.Name, newJob.Summary, newJob.RetentionName,
		newJob.ScheduleName, newJob.TargetEndpoint, newJob.Agent)
	t.Output(os.Stdout)
	return nil
}

func EditExstingJob(u string) error {
	j, err := GetJob(uuid.Parse(u))
	if err != nil {
		return fmt.Errorf("ERROR: Could not retrieve job '%s': %s", u, err)
	}
	paused := "false"
	if j.Paused {
		paused = "true"
	}

	content := invokeEditor(`{
		"name"      : "` + j.Name + `",
		"summary"   : "` + j.Summary + `",

		"store"     : "` + j.StoreUUID + `",
		"target"    : "` + j.TargetUUID + `",
		"retention" : "` + j.RetentionUUID + `",
		"schedule"  : "` + j.ScheduleUUID + `",
		"paused"    : ` + paused + `
		}`)

	j, err = UpdateJob(uuid.Parse(u), content)
	if err != nil {
		return fmt.Errorf("ERROR: Could not update job '%s': %s", u, err)
	}

	fmt.Fprintf(os.Stdout, "Updated job.\n")
	t := tui.NewTable("UUID", "P?", "Name", "Description", "Retention Policy", "Schedule", "Target", "Agent")
	paused = "-"
	if j.Paused {
		paused = "Y"
	}
	t.Row(j.UUID, paused, j.Name, j.Summary, j.RetentionName, j.ScheduleName, j.TargetEndpoint, j.Agent)
	t.Output(os.Stdout)
	return nil
}

func DeleteJobByUUID(u string) error {
	err := DeleteJob(uuid.Parse(u))
	if err != nil {
		return fmt.Errorf("ERROR: Cannot delete job '%s': %s", u, err)
	}
	fmt.Fprintf(os.Stdout, "Deleted job '%s'\n", u)
	return nil
}

/*
func processRunJobRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprint(os.Stderr, "\nERROR: Requires a single UUID\n", args)
		//FIXME  show help
		os.Exit(1)
	}

	requested_UUID := uuid.Parse(args[0])

	// FIXME when owner can be passed in or otherwise fetched
	content := "{\"owner\":\"anon\"}"

	err := RunJob(requested_UUID, content)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not run job:\n", err)
		os.Exit(1)
	}

	fmt.Println(requested_UUID, "scheduled")

	return
}
*/
