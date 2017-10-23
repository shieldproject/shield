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
	Flags:   commands.FlagList{commands.UpdateIfExistsFlag},
	RunFn:   cliCreateJob,
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
