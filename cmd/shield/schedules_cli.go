package main

import (
	//"encoding/json"
	"fmt"
	"os"

	"github.com/pborman/uuid"

	. "github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

type ListScheduleOptions struct {
	Unused bool
	Used   bool
	UUID   string
}

func ListSchedules(opts ListScheduleOptions) error {
	//FIXME: --(un)used not working?
	schedules, err := GetSchedules(ScheduleFilter{
		Unused: MaybeBools(opts.Unused, opts.Used),
	})
	if err != nil {
		return fmt.Errorf("failed to retrieve schedules from SHIELD: %s", err)
	}
	t := tui.NewTable("UUID", "Name", "Description", "Frequency / Interval (UTC)")
	for _, schedule := range schedules {
		//FIXME: implement with GetSchedule(UUID)
		if len(opts.UUID) > 0 && opts.UUID == schedule.UUID {
			t.Row(schedule.UUID, schedule.Name, schedule.Summary, schedule.When)
			break
		} else if len(opts.UUID) > 0 && opts.UUID != schedule.UUID {
			continue
		}
		t.Row(schedule.UUID, schedule.Name, schedule.Summary, schedule.When)
	}
	t.Output(os.Stdout)
	return nil
}

func CreateNewSchedule() error {
	content := invokeEditor(`{
		"name":    "Empty Schedule",
		"summary": "Late for a very important date",
		"when":    "daily at 4:00"
		}`)

	newSchedule, err := CreateSchedule(content)
	if err != nil {
		return fmt.Errorf("ERROR: Could not create new schedule: %s", err)
	}
	fmt.Fprintf(os.Stdout, "Created new schedule.\n")
	t := tui.NewTable("UUID", "Name", "Description", "Frequency / Interval (UTC)")
	t.Row(newSchedule.UUID, newSchedule.Name, newSchedule.Summary, newSchedule.When)
	t.Output(os.Stdout)

	return nil
}

func EditExstingSchedule(u string) error {
	s, err := GetSchedule(uuid.Parse(u))
	if err != nil {
		return fmt.Errorf("ERROR: Could not retrieve schedule '%s': %s", u, err)
	}

	content := invokeEditor(`{
		"name":    "` + s.Name + `",
		"summary": "` + s.Summary + `",
		"when":    "` + s.When + `"
		}`)

	s, err = UpdateSchedule(uuid.Parse(u), content)
	if err != nil {
		return fmt.Errorf("ERROR: Could not update schedule '%s': %s", u, err)
	}
	fmt.Fprintf(os.Stdout, "Updated schedule.\n")
	t := tui.NewTable("UUID", "Name", "Description", "Frequency / Interval (UTC)")
	t.Row(s.UUID, s.Name, s.Summary, s.When)
	t.Output(os.Stdout)
	return nil
}

func DeleteScheduleByUUID(u string) error {
	err := DeleteSchedule(uuid.Parse(u))
	if err != nil {
		return fmt.Errorf("ERROR: Cannot delete schedule '%s': %s", u, err)
	}
	fmt.Fprintf(os.Stdout, "Deleted schedule '%s'\n", u)
	return nil
}
