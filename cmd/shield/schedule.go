package main

import (
	"os"
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

//List available backup schedules
func cliListSchedules(args ...string) error {
	DEBUG("running 'list schedules' command")
	DEBUG("  show unused? %v", *opts.Unused)
	DEBUG("  show in-use? %v", *opts.Used)
	if *opts.Raw {
		DEBUG(" fuzzy search? %v", api.MaybeBools(*opts.Fuzzy, *opts.Raw).Yes)
	}

	schedules, err := api.GetSchedules(api.ScheduleFilter{
		Name:       strings.Join(args, " "),
		Unused:     api.MaybeBools(*opts.Unused, *opts.Used),
		ExactMatch: api.Opposite(api.MaybeBools(*opts.Fuzzy, *opts.Raw)),
	})
	if err != nil {
		return err
	}

	if *opts.Raw {
		return RawJSON(schedules)
	}

	t := tui.NewTable("Name", "Summary", "Frequency / Interval (UTC)")
	for _, schedule := range schedules {
		t.Row(schedule, schedule.Name, schedule.Summary, schedule.When)
	}
	t.Output(os.Stdout)
	return nil
}

//Print detailed information about a specific backup schedule
func cliGetSchedule(args ...string) error {
	DEBUG("running 'show schedule' command")

	schedule, _, err := FindSchedule(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if *opts.Raw {
		return RawJSON(schedule)
	}
	if *opts.ShowUUID {
		return RawUUID(schedule.UUID)
	}

	ShowSchedule(schedule)
	return nil
}

//Create a new backup schedule
func cliCreateSchedule(args ...string) error {
	DEBUG("running 'create schedule' command")

	var err error
	var content string
	if *opts.Raw {
		content, err = readall(os.Stdin)
		if err != nil {
			return err
		}

	} else {
		in := tui.NewForm()
		in.NewField("Schedule Name", "name", "", "", tui.FieldIsRequired)
		in.NewField("Summary", "summary", "", "", tui.FieldIsOptional)
		in.NewField("Time Spec (i.e. 'daily 4am')", "when", "", "", tui.FieldIsRequired)

		if err := in.Show(); err != nil {
			return err
		}

		if !in.Confirm("Really create this schedule?") {
			return errCanceled
		}

		content, err = in.BuildContent()
		if err != nil {
			return err
		}
	}

	DEBUG("JSON:\n  %s\n", content)

	if *opts.UpdateIfExists {
		t, id, err := FindSchedule(content, true)
		if err != nil {
			return err
		}
		if id != nil {
			t, err = api.UpdateSchedule(id, content)
			if err != nil {
				return err
			}
			MSG("Updated existing schedule")
			return cliGetSchedule(t.UUID)
		}
	}

	s, err := api.CreateSchedule(content)
	if err != nil {
		return err
	}

	MSG("Created new schedule")
	return cliGetSchedule(s.UUID)
}

//Modify an existing backup schedule
func cliEditSchedule(args ...string) error {
	DEBUG("running 'edit schedule' command")

	s, id, err := FindSchedule(strings.Join(args, " "), *opts.Raw)
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
		in.NewField("Schedule Name", "name", s.Name, "", tui.FieldIsRequired)
		in.NewField("Summary", "summary", s.Summary, "", tui.FieldIsOptional)
		in.NewField("Time Spec (i.e. 'daily 4am')", "when", s.When, "", tui.FieldIsRequired)

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
	s, err = api.UpdateSchedule(id, content)
	if err != nil {
		return err
	}

	MSG("Updated schedule")
	return cliGetSchedule(s.UUID)
}

//Delete a backup schedule
func cliDeleteSchedule(args ...string) error {
	DEBUG("running 'delete schedule' command")

	schedule, id, err := FindSchedule(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if !*opts.Raw {
		ShowSchedule(schedule)
		if !tui.Confirm("Really delete this schedule?") {
			return errCanceled
		}
	}

	if err := api.DeleteSchedule(id); err != nil {
		return err
	}

	OK("Deleted schedule")
	return nil
}
