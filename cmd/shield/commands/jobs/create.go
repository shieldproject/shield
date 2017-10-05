package jobs

import (
	"fmt"
	"os"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Create - Create a new backup job
var Create = &commands.Command{
	Summary: "Create a new backup job",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{commands.UpdateIfExistsFlag},
		JSONInput: `{
			"name":"TestJob",
			"paused":true,
			"retention":"18a446c4-c068-4c09-886c-cb77b6a85274",
			"schedule":"daily 4am",
			"store":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf",
			"summary":"A Test Job",
			"target":"84751f04-2be2-428d-b6a3-2022c63bf6ee"
			"tenant":"5c839605-856f-4a1d-97cd-e5f4019c08af"
		}`,
		JSONOutput: `{
			"uuid":"f6623a6f-8dce-46b2-a293-5525bc3a3588",
			"name":"TestJob",
			"summary":"A Test Job",
			"retention_name":"AnotherPolicy",
			"retention_uuid":"18a446c4-c068-4c09-886c-cb77b6a85274",
			"expiry":31536000,
			"schedule":"daily 4am",
			"paused":true,
			"store_uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf",
			"store_name":"AnotherStore",
			"store_plugin":"s3",
			"store_endpoint":"{\"endpoint\":\"schmendpoint\"}",
			"target_uuid":"84751f04-2be2-428d-b6a3-2022c63bf6ee",
			"target_name":"TestTarget",
			"target_plugin":"postgres",
			"target_endpoint":"{\"endpoint\":\"schmendpoint\"}",
			"agent":"127.0.0.1:1234"
			"tenant_uuid":"5c839605-856f-4a1d-97cd-e5f4019c08af"
			"tenant_name":"Engineering"
		}`,
	},
	RunFn: cliCreateJob,
	Group: commands.JobsGroup,
}

func cliCreateJob(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'create job' command")

	var err error
	var content string
	if *opts.Raw {
		content, err = internal.ReadAll(os.Stdin)
		if err != nil {
			return err
		}

	} else {
		in := tui.NewForm()
		in.NewField("Job Name", "name", "", "", tui.FieldIsRequired)
		in.NewField("Summary", "summary", "", "", tui.FieldIsOptional)

		in.NewField("Store", "store", "", "", internal.FieldIsStoreUUID)
		in.NewField("Target", "target", "", "", internal.FieldIsTargetUUID)
		in.NewField("Retention Policy", "retention", "", "", internal.FieldIsRetentionPolicyUUID)
		in.NewField("Schedule Timespec (e.g. daily 4am)", "schedule", "", "", tui.FieldIsRequired)

		in.NewField("Paused?", "paused", "no", "", tui.FieldIsBoolean)

		err := in.Show()
		if err != nil {
			return err
		}

		if !in.Confirm("Really create this backup job?") {
			return internal.ErrCanceled
		}

		if opts.APIVersion == 1 {
			log.DEBUG("Creating schedule against v1 API")
			//We need to create a schedule for this
			//Also, we need to give the created schedule to the form.
			scheduleField := in.GetField("schedule")
			jobNameField := in.GetField("name")

			sched := &api.Schedule{
				Name:    uuid.New(),
				Summary: fmt.Sprintf("Created by v8 CLI for job '%s`", jobNameField.Value.(string)),
				When:    scheduleField.Value.(string),
			}

			schedUUID, err := sched.Create()
			if err != nil {
				return err
			}

			scheduleField.Value = schedUUID
		}

		content, err = in.BuildContent()
		if err != nil {
			return err
		}
	}

	log.DEBUG("JSON:\n  %s\n", content)

	if *opts.UpdateIfExists {
		t, id, err := internal.FindJob(content, true)
		if err != nil {
			return err
		}
		if id != nil {
			t, err = api.UpdateJob(id, content)
			if err != nil {
				return err
			}
			commands.MSG("Updated existing job")
			return cliGetJob(opts, t.UUID)
		}
	}

	job, err := api.CreateJob(content)
	if err != nil {
		return err
	}

	commands.MSG("Created new job")
	return cliGetJob(opts, job.UUID)
}
