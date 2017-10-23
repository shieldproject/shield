package jobs

import (
	"fmt"
	"os"
	"strings"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Edit - Modify an existing backup job
var Edit = &commands.Command{
	Flags: commands.FlagList{commands.JobNameFlag},
	RunFn: cliEditJob,
}

func cliEditJob(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'edit job' command")

	j, id, err := internal.FindJob(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	var content string
	if *opts.Raw {
		content, err = internal.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
	} else {
		schedDefault := j.Schedule
		schedShowAs := j.Schedule

		if opts.APIVersion == 1 {
			schedDefault = j.ScheduleUUID
			schedShowAs = j.ScheduleWhen
		}

		in := tui.NewForm()
		in.NewField("Job Name", "name", j.Name, "", tui.FieldIsRequired)
		in.NewField("Summary", "summary", j.Summary, "", tui.FieldIsOptional)
		in.NewField("Store", "store", j.StoreUUID, j.StoreName, internal.FieldIsStoreUUID)
		in.NewField("Target", "target", j.TargetUUID, j.TargetName, internal.FieldIsTargetUUID)
		in.NewField("Retention Policy", "retention", j.RetentionUUID, fmt.Sprintf("%s - %dd", j.RetentionName, j.Expiry/86400), internal.FieldIsRetentionPolicyUUID)
		in.NewField("Schedule Timespec", "schedule", schedDefault, schedShowAs, tui.FieldIsRequired)

		if err = in.Show(); err != nil {
			return err
		}

		if !in.Confirm("Save these changes?") {
			return internal.ErrCanceled
		}

		if opts.APIVersion == 1 {
			scheduleField := in.GetField("schedule")
			if scheduleField.Value.(string) != j.ScheduleUUID {
				log.DEBUG("Creating schedule against v1 API")
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
		}

		content, err = in.BuildContent()
		if err != nil {
			return err
		}
	}

	log.DEBUG("JSON:\n  %s\n", content)
	j, err = api.UpdateJob(id, content)
	if err != nil {
		return err
	}

	commands.MSG("Updated job")

	if opts.APIVersion == 1 {
		maybeGCSchedule(j.ScheduleUUID)
	}

	return cliGetJob(opts, j.UUID)
}
