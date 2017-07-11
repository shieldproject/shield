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
func cliListJobs(opts Options, args []string, help bool) error {
	if help {
		HelpListMacro("job", "jobs")
		FlagHelp("Show only jobs using the specified target", true, "-t", "--target=value")
		FlagHelp("Show only jobs using the specified store", true, "-s", "--store=value")
		FlagHelp("Show only jobs using the specified schedule", true, "-w", "--schedule=value")
		FlagHelp("Show only jobs using the specified retention policy", true, "-p", "--policy=value")
		FlagHelp("Show only jobs which are in the paused state", true, "--paused")
		FlagHelp("Show only jobs which are NOT in the paused state", true, "--unpaused")
		JSONHelp(`[{"uuid":"f6623a6f-8dce-46b2-a293-5525bc3a3588","name":"TestJob","summary":"A Test Job","retention_name":"AnotherPolicy","retention_uuid":"18a446c4-c068-4c09-886c-cb77b6a85274","expiry":31536000,"schedule_name":"AnotherSched","schedule_uuid":"9a58a3fa-7457-431c-b094-e201b42b5c7b","schedule_when":"daily 4am","paused":true,"store_uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf","store_name":"AnotherStore","store_plugin":"s3","store_endpoint":"{\"endpoint\":\"schmendpoint\"}","target_uuid":"84751f04-2be2-428d-b6a3-2022c63bf6ee","target_name":"TestTarget","target_plugin":"postgres","target_endpoint":"{\"endpoint\":\"schmendpoint\"}","agent":"127.0.0.1:1234"}]`)
		return nil
	}

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
func cliGetJob(opts Options, args []string, help bool) error {
	if help {
		HelpShowMacro("job", "jobs")
		JSONHelp(`{"uuid":"f6623a6f-8dce-46b2-a293-5525bc3a3588","name":"TestJob","summary":"A Test Job","retention_name":"AnotherPolicy","retention_uuid":"18a446c4-c068-4c09-886c-cb77b6a85274","expiry":31536000,"schedule_name":"AnotherSched","schedule_uuid":"9a58a3fa-7457-431c-b094-e201b42b5c7b","schedule_when":"daily 4am","paused":true,"store_uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf","store_name":"AnotherStore","store_plugin":"s3","store_endpoint":"{\"endpoint\":\"schmendpoint\"}","target_uuid":"84751f04-2be2-428d-b6a3-2022c63bf6ee","target_name":"TestTarget","target_plugin":"postgres","target_endpoint":"{\"endpoint\":\"schmendpoint\"}","agent":"127.0.0.1:1234"}`)
		return nil
	}

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
func cliCreateJob(opts Options, args []string, help bool) error {
	if help {
		HelpCreateMacro("job", "jobs")
		InputHelp(`{"name":"TestJob","paused":true,"retention":"18a446c4-c068-4c09-886c-cb77b6a85274","schedule":"9a58a3fa-7457-431c-b094-e201b42b5c7b","store":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf","summary":"A Test Job","target":"84751f04-2be2-428d-b6a3-2022c63bf6ee"}`)
		JSONHelp(`{"uuid":"f6623a6f-8dce-46b2-a293-5525bc3a3588","name":"TestJob","summary":"A Test Job","retention_name":"AnotherPolicy","retention_uuid":"18a446c4-c068-4c09-886c-cb77b6a85274","expiry":31536000,"schedule_name":"AnotherSched","schedule_uuid":"9a58a3fa-7457-431c-b094-e201b42b5c7b","schedule_when":"daily 4am","paused":true,"store_uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf","store_name":"AnotherStore","store_plugin":"s3","store_endpoint":"{\"endpoint\":\"schmendpoint\"}","target_uuid":"84751f04-2be2-428d-b6a3-2022c63bf6ee","target_name":"TestTarget","target_plugin":"postgres","target_endpoint":"{\"endpoint\":\"schmendpoint\"}","agent":"127.0.0.1:1234"}`)
		return nil
	}

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
			return cliGetJob(opts, []string{t.UUID}, false)
		}
	}

	job, err := api.CreateJob(content)
	if err != nil {
		return err
	}

	MSG("Created new job")
	return cliGetJob(opts, []string{job.UUID}, false)
}

//Modify an existing backup job
func cliEditJob(opts Options, args []string, help bool) error {
	if help {
		HelpEditMacro("job", "jobs")

		return nil
	}

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
	return cliGetJob(opts, []string{j.UUID}, false)
}

//Delete a backup job
func cliDeleteJob(opts Options, args []string, help bool) error {
	if help {
		HelpDeleteMacro("job", "jobs")
		InputHelp(`{"name":"AnotherJob","retention":"18a446c4-c068-4c09-886c-cb77b6a85274","schedule":"9a58a3fa-7457-431c-b094-e201b42b5c7b","store":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf","summary":"A Test Job","target":"84751f04-2be2-428d-b6a3-2022c63bf6ee"}`)
		JSONHelp(`{"uuid":"f6623a6f-8dce-46b2-a293-5525bc3a3588","name":"AnotherJob","summary":"A Test Job","retention_name":"AnotherPolicy","retention_uuid":"18a446c4-c068-4c09-886c-cb77b6a85274","expiry":31536000,"schedule_name":"AnotherSched","schedule_uuid":"9a58a3fa-7457-431c-b094-e201b42b5c7b","schedule_when":"daily 4am","paused":true,"store_uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf","store_name":"AnotherStore","store_plugin":"s3","store_endpoint":"{\"endpoint\":\"schmendpoint\"}","target_uuid":"84751f04-2be2-428d-b6a3-2022c63bf6ee","target_name":"TestTarget","target_plugin":"postgres","target_endpoint":"{\"endpoint\":\"schmendpoint\"}","agent":"127.0.0.1:1234"}`)
		return nil
	}

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
func cliPauseJob(opts Options, args []string, help bool) error {
	if help {
		FlagHelp(`A string partially matching the name of a job to pause
				or a UUID exactly matching the UUID of a job to pause.
				Not setting this value explicitly will default it to the empty string.`,
			false, "<job>")
		HelpKMacro()
		return nil
	}

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
func cliUnpauseJob(opts Options, args []string, help bool) error {
	if help {
		FlagHelp(`A string partially matching the name of a job to unpause
				or a UUID exactly matching the UUID of a job to unpause.
				Not setting this value explicitly will default it to the empty string.`,
			false, "<job>")
		HelpKMacro()
		return nil
	}

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
func cliRunJob(opts Options, args []string, help bool) error {
	if help {
		MessageHelp("Note: If raw mode is specified and the targeted SHIELD backend does not support handing back the task uuid, the task_uuid in the JSON will be the empty string")
		FlagHelp(`A string partially matching the name of a job to run
				or a UUID exactly matching the UUID of a job to run.
				Not setting this value explicitly will default it to the empty string.`,
			false, "<job>")
		JSONHelp(`{"ok":"Scheduled immediate run of job","task_uuid":"143e5494-63c4-4e05-9051-8b3015eae061"}`)
		HelpKMacro()
		return nil
	}

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
