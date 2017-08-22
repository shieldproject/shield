package jobs

import (
	"fmt"
	"os"
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Edit - Modify an existing backup job
var Edit = &commands.Command{
	Summary: "Modify an existing backup job",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{commands.JobNameFlag},
		JSONInput: `{
			"name":"AnotherJob",
			"retention":"18a446c4-c068-4c09-886c-cb77b6a85274",
			"schedule":"9a58a3fa-7457-431c-b094-e201b42b5c7b",
			"store":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf",
			"summary":"A Test Job",
			"target":"84751f04-2be2-428d-b6a3-2022c63bf6ee"
		}`,
		JSONOutput: `{
			"uuid":"f6623a6f-8dce-46b2-a293-5525bc3a3588",
			"name":"AnotherJob",
			"summary":"A Test Job",
			"retention_name":"AnotherPolicy",
			"retention_uuid":"18a446c4-c068-4c09-886c-cb77b6a85274",
			"expiry":31536000,
			"schedule_name":"AnotherSched",
			"schedule_uuid":"9a58a3fa-7457-431c-b094-e201b42b5c7b",
			"schedule_when":"daily 4am",
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
		}`,
	},
	RunFn: cliEditJob,
	Group: commands.JobsGroup,
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

		in := tui.NewForm()
		in.NewField("Job Name", "name", j.Name, "", tui.FieldIsRequired)
		in.NewField("Summary", "summary", j.Summary, "", tui.FieldIsOptional)
		in.NewField("Store", "store", j.StoreUUID, j.StoreName, internal.FieldIsStoreUUID)
		in.NewField("Target", "target", j.TargetUUID, j.TargetName, internal.FieldIsTargetUUID)
		in.NewField("Retention Policy", "retention", j.RetentionUUID, fmt.Sprintf("%s - %dd", j.RetentionName, j.Expiry/86400), internal.FieldIsRetentionPolicyUUID)
		in.NewField("Schedule", "schedule", j.ScheduleUUID, fmt.Sprintf("%s - %s", j.ScheduleName, j.ScheduleWhen), internal.FieldIsScheduleUUID)

		if err = in.Show(); err != nil {
			return err
		}

		if !in.Confirm("Save these changes?") {
			return internal.ErrCanceled
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
	return cliGetJob(opts, j.UUID)
}
