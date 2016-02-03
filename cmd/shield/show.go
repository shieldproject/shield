package main

import (
	"fmt"
	"os"
	"time"

	. "github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

func ShowTarget(target Target) {
	t := tui.NewReport()
	t.Add("Name", target.Name)
	t.Add("Summary", target.Summary)
	t.Break()

	t.Add("Plugin", target.Plugin)
	t.Add("Configuration", target.Endpoint)
	t.Add("Remote IP", target.Agent)
	t.Output(os.Stdout)
}

func ShowStore(store Store) {
	t := tui.NewReport()
	t.Add("Name", store.Name)
	t.Add("Summary", store.Summary)
	t.Break()

	t.Add("Plugin", store.Plugin)
	t.Add("Configuration", store.Endpoint)
	t.Output(os.Stdout)
}

func ShowSchedule(schedule Schedule) {
	t := tui.NewReport()
	t.Add("Name", schedule.Name)
	t.Add("Summary", schedule.Summary)
	t.Add("Timespec", schedule.When)
	t.Output(os.Stdout)
}

func ShowRetentionPolicy(policy RetentionPolicy) {
	t := tui.NewReport()
	t.Add("Name", policy.Name)
	t.Add("Summary", policy.Summary)
	t.Add("Expiration", fmt.Sprintf("%d days", policy.Expires/86400))
	t.Output(os.Stdout)
}

func ShowJob(job Job) {
	t := tui.NewReport()
	t.Add("Name", job.Name)
	t.Add("Paused", BoolString(job.Paused))
	t.Break()

	t.Add("Retention Policy", job.RetentionName)
	t.Add("Expires in", fmt.Sprintf("%d days", job.Expiry/86400))
	t.Break()

	t.Add("Schedule Policy", job.ScheduleName)
	t.Break()

	t.Add("Target", job.TargetPlugin)
	t.Add("Target Endpoint", job.TargetEndpoint)
	t.Add("Remote IP", job.Agent)
	t.Break()

	t.Add("Store", job.StorePlugin)
	t.Add("Store Endpoint", job.StoreEndpoint)
	t.Break()

	t.Add("Notes", job.Summary)

	t.Output(os.Stdout)
}

func ShowTask(task Task) {
	t := tui.NewReport()
	t.Add("UUID", task.UUID)
	t.Add("Owner", task.Owner)
	t.Add("Type", task.Op)
	t.Add("Status", task.Status)
	t.Break()

	started := "(pending)"
	stopped := "(not yet started)"
	if !task.StartedAt.IsZero() {
		stopped = "(running)"
		started = task.StartedAt.Format(time.RFC1123Z)
	}
	if !task.StoppedAt.IsZero() {
		stopped = task.StoppedAt.Format(time.RFC1123Z)
	}
	t.Add("Started at", started)
	t.Add("Stopped at", stopped)
	t.Break()

	t.Add("Job UUID", task.JobUUID)
	t.Add("Archive UUID", task.ArchiveUUID)
	t.Break()

	t.Add("Log", task.Log)
	t.Output(os.Stdout)
}

func ShowArchive(archive Archive) {
	t := tui.NewReport()
	t.Add("UUID", archive.UUID)
	t.Add("Backup Key", archive.StoreKey)
	t.Add("Target", fmt.Sprintf("%s %s", archive.TargetPlugin, archive.TargetEndpoint))
	t.Add("Store", fmt.Sprintf("%s %s", archive.StorePlugin, archive.StoreEndpoint))
	t.Break()

	t.Add("Taken at", archive.TakenAt.Format(time.RFC1123Z))
	t.Add("Expires at", archive.ExpiresAt.Format(time.RFC1123Z))
	t.Add("Notes", archive.Notes)

	t.Output(os.Stdout)
}
