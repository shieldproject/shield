package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

//List available backup jobs
func cliListJobs(args ...string) error {
	DEBUG("running 'list jobs' command")
	DEBUG("  for target:      '%s'", *opts.Target)
	DEBUG("  for store:       '%s'", *opts.Store)
	DEBUG("  for schedule:    '%s'", *opts.Schedule)
	DEBUG("  for ret. policy: '%s'", *opts.Retention)
	DEBUG("  show paused?      %v", *opts.Paused)
	DEBUG("  show unpaused?    %v", *opts.Unpaused)
	if *opts.Raw {
		DEBUG(" fuzzy search? %v", api.MaybeBools(*opts.Fuzzy, *opts.Raw).Yes)
	}

	jobs, err := api.GetJobs(api.JobFilter{
		Name:       strings.Join(args, " "),
		Paused:     api.MaybeBools(*opts.Paused, *opts.Unpaused),
		Target:     *opts.Target,
		Store:      *opts.Store,
		Schedule:   *opts.Schedule,
		Retention:  *opts.Retention,
		ExactMatch: api.Opposite(api.MaybeBools(*opts.Fuzzy, *opts.Raw)),
	})
	if err != nil {
		return err
	}

	if *opts.Raw {
		return RawJSON(jobs)
	}

	t := tui.NewTable("Name", "P?", "Summary", "Retention Policy", "Schedule", "Remote IP", "Target")
	for _, job := range jobs {
		t.Row(job, job.Name, BoolString(job.Paused), job.Summary,
			job.RetentionName, job.ScheduleName, job.Agent, PrettyJSON(job.TargetEndpoint))
	}
	t.Output(os.Stdout)
	return nil
}

//Print detailed information about a specific backup job
func cliGetJob(args ...string) error {
	DEBUG("running 'show job' command")

	job, _, err := FindJob(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if *opts.Raw {
		return RawJSON(job)
	}
	if *opts.ShowUUID {
		return RawUUID(job.UUID)
	}

	ShowJob(job)
	return nil
}

//Create a new backup job
func cliCreateJob(args ...string) error {
	DEBUG("running 'create job' command")

	var err error
	var content string
	if *opts.Raw {
		content, err = readall(os.Stdin)
		if err != nil {
			return err
		}

	} else {
		in := tui.NewForm()
		in.NewField("Job Name", "name", "", "", tui.FieldIsRequired)
		in.NewField("Summary", "summary", "", "", tui.FieldIsOptional)

		in.NewField("Store", "store", "", "", FieldIsStoreUUID)
		in.NewField("Target", "target", "", "", FieldIsTargetUUID)
		in.NewField("Retention Policy", "retention", "", "", FieldIsRetentionPolicyUUID)
		in.NewField("Schedule", "schedule", "", "", FieldIsScheduleUUID)

		in.NewField("Paused?", "paused", "no", "", tui.FieldIsBoolean)
		err := in.Show()
		if err != nil {
			return err
		}

		if !in.Confirm("Really create this backup job?") {
			return errCanceled
		}

		content, err = in.BuildContent()
		if err != nil {
			return err
		}
	}

	DEBUG("JSON:\n  %s\n", content)

	if *opts.UpdateIfExists {
		t, id, err := FindJob(content, true)
		if err != nil {
			return err
		}
		if id != nil {
			t, err = api.UpdateJob(id, content)
			if err != nil {
				return err
			}
			MSG("Updated existing job")
			return cliGetJob(t.UUID)
		}
	}

	job, err := api.CreateJob(content)
	if err != nil {
		return err
	}

	MSG("Created new job")
	return cliGetJob(job.UUID)
}

//Modify an existing backup job
func cliEditJob(args ...string) error {
	DEBUG("running 'edit job' command")

	j, id, err := FindJob(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	var content string
	if *opts.Raw {
		content, err = readall(os.Stdin)
		if err != nil {
			return err
		}

	} else {

		in := tui.NewForm()
		in.NewField("Job Name", "name", j.Name, "", tui.FieldIsRequired)
		in.NewField("Summary", "summary", j.Summary, "", tui.FieldIsOptional)
		in.NewField("Store", "store", j.StoreUUID, j.StoreName, FieldIsStoreUUID)
		in.NewField("Target", "target", j.TargetUUID, j.TargetName, FieldIsTargetUUID)
		in.NewField("Retention Policy", "retention", j.RetentionUUID, fmt.Sprintf("%s - %dd", j.RetentionName, j.Expiry/86400), FieldIsRetentionPolicyUUID)
		in.NewField("Schedule", "schedule", j.ScheduleUUID, fmt.Sprintf("%s - %s", j.ScheduleName, j.ScheduleWhen), FieldIsScheduleUUID)

		if err = in.Show(); err != nil {
			return err
		}

		if !in.Confirm("Save these changes?") {
			return errCanceled
		}

		content, err = in.BuildContent()
		if err != nil {
			return err
		}
	}

	DEBUG("JSON:\n  %s\n", content)
	j, err = api.UpdateJob(id, content)
	if err != nil {
		return err
	}

	MSG("Updated job")
	return cliGetJob(j.UUID)
}

//Delete a backup job
func cliDeleteJob(args ...string) error {
	DEBUG("running 'delete job' command")

	job, id, err := FindJob(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if !*opts.Raw {
		ShowJob(job)
		if !tui.Confirm("Really delete this backup job?") {
			return errCanceled
		}
	}

	if err := api.DeleteJob(id); err != nil {
		return err
	}

	OK("Deleted job")
	return nil
}

//Pause a backup job
func cliPauseJob(args ...string) error {
	DEBUG("running 'pause job' command")

	_, id, err := FindJob(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}
	if err := api.PauseJob(id); err != nil {
		return err
	}

	return nil
}

//Unpause a backup job
func cliUnpauseJob(args ...string) error {
	DEBUG("running 'unpause job' command")

	_, id, err := FindJob(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}
	if err := api.UnpauseJob(id); err != nil {
		return err
	}

	OK("Unpaused job")
	return nil
}

//Schedule an immediate run of a backup job
func cliRunJob(args ...string) error {
	DEBUG("running 'run job' command")

	_, id, err := FindJob(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	var params = struct {
		Owner string `json:"owner"`
	}{
		Owner: CurrentUser(),
	}

	b, err := json.Marshal(params)
	if err != nil {
		return err
	}

	taskUUID, err := api.RunJob(id, string(b))
	if err != nil {
		return err
	}

	if *opts.Raw {
		RawJSON(map[string]interface{}{
			"ok":        "Scheduled immediate run of job",
			"task_uuid": taskUUID,
		})
	} else {
		OK("Scheduled immediate run of job")
		if taskUUID != "" {
			ansi.Printf("To view task, type @B{shield task %s}\n", taskUUID)
		}
	}

	return nil
}
