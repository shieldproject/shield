package jobs

import (
	"encoding/json"
	"strings"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
)

//Run - Schedule an immediate run of a backup job
var Run = &commands.Command{
	Summary: "Schedule an immediate run of a backup job",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{commands.JobNameFlag},
		JSONOutput: `{
			"ok":"Scheduled immediate run of job",
			"task_uuid":"143e5494-63c4-4e05-9051-8b3015eae061"
		}`,
	},
	RunFn: cliRunJob,
	Group: commands.JobsGroup,
}

func cliRunJob(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'run job' command")

	_, id, err := internal.FindJob(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	var params = struct {
		Owner string `json:"owner"`
	}{
		Owner: commands.CurrentUser(),
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
		internal.RawJSON(map[string]interface{}{
			"ok":        "Scheduled immediate run of job",
			"task_uuid": taskUUID,
		})
	} else {
		commands.OK("Scheduled immediate run of job")
		if taskUUID != "" {
			ansi.Printf("To view task, type @B{shield task %s}\n", taskUUID)
		}
	}

	return nil
}
