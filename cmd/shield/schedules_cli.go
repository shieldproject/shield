package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pborman/uuid"
	"github.com/spf13/cobra"

	. "github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

var (

	//== Applicable actions for Schedules

	createScheduleCmd = &cobra.Command{
		Use:   "schedule",
		Short: "Creates a new schedule",
		Long:  "Create a new schedule with ...",
	} // FIXME

	updateScheduleCmd = &cobra.Command{
		Use:   "schedule",
		Short: "Update the Schedules",
	}
)

func init() {

	// Hookup functions to the subcommands
	createScheduleCmd.Run = processCreateScheduleRequest
	updateScheduleCmd.Run = processUpdateScheduleRequest

	// Add the subcommands to the base actions
	createCmd.AddCommand(createScheduleCmd)
	updateCmd.AddCommand(updateScheduleCmd)
}

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
		"name":     "Empty Schedule",
		"summary":  "Late for a very important date",
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

func DeleteScheduleByUUID(u string) error {
	err := DeleteSchedule(uuid.Parse(u))
	if err != nil {
		return fmt.Errorf("ERROR: Cannot delete schedule '%s': %s", u, err)
	}
	fmt.Fprintf(os.Stdout, "Deleted schedule '%s'\n", u)
	return nil
}

func processCreateScheduleRequest(cmd *cobra.Command, args []string) {

	// Validate Request
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "\nERROR: Unexpected arguments following command: %v\n", args)
		//FIXME  show help
		os.Exit(1)
	}

	// Invoke editor
	content := invokeEditor(`{
	"name":     "",
	"summary":  "",
	"when":    ""
}`)

	fmt.Println("Got the following content:\n\n", content)

	data, err := CreateSchedule(content)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not fetch list of schedules:\n", err)
		os.Exit(1)
	}

	// Print
	output, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render list of schedules:\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output[:]))

	return
}

func processUpdateScheduleRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprint(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	requested_UUID := uuid.Parse(args[0])

	original_data, err := GetSchedule(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not show schedule:\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(original_data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render schedule:\n", err)
	}

	fmt.Println("Got the following original schedule:\n\n", string(data))

	// Invoke editor
	content := invokeEditor(string(data))

	fmt.Println("Got the following edited schedule:\n\n", content)

	update_data, err := UpdateSchedule(requested_UUID, content)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not update schedules:\n", err)
		os.Exit(1)
	}
	// Print
	output, err := json.MarshalIndent(update_data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render schedule:\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output[:]))

	return
}
