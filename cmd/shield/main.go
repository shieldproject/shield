package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/pborman/getopt"
	"github.com/pborman/uuid"

	. "github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

func require(good bool, msg string) {
	if !good {
		fmt.Fprintf(os.Stderr, "USAGE: %s ...\n", msg)
		os.Exit(1)
	}
}

func readall(in io.Reader) (string, error) {
	b, err := ioutil.ReadAll(in)
	return string(b), err
}

var (
	debug = false
)

func main() {
	options := Options{
		Shield:   getopt.StringLong("shield", 'H', "", "SHIELD target to run command against, i.e. http://shield.my.domain:8080"),
		Used:     getopt.BoolLong("used", 0, "Only show things that are in-use by something else"),
		Unused:   getopt.BoolLong("unused", 0, "Only show things that are not used by something else"),
		Paused:   getopt.BoolLong("paused", 0, "Only show jobs that are paused"),
		Unpaused: getopt.BoolLong("unpaused", 0, "Only show jobs that are unpaused"),
		All:      getopt.BoolLong("all", 'a', "Show all the things"),

		Debug: getopt.BoolLong("debug", 'D', "Enable debugging"),
		Trace: getopt.BoolLong("trace", 'T', "Enable trace mode"),
		Raw:   getopt.BoolLong("raw", 0, "Operate in RAW mode, reading and writing only JSON"),

		Status:    getopt.StringLong("status", 'S', "", "Only show archives/tasks with the given status"),
		Target:    getopt.StringLong("target", 't', "", "Only show things for the target with this UUID"),
		Store:     getopt.StringLong("store", 's', "", "Only show things for the store with this UUID"),
		Schedule:  getopt.StringLong("schedule", 'w', "", "Only show things for the schedule with this UUID"),
		Retention: getopt.StringLong("policy", 'p', "", "Only show things for the retention policy with this UUID"),
		Plugin:    getopt.StringLong("plugin", 'P', "", "Only show things for the given target or store plugin"),
		After:     getopt.StringLong("after", 'A', "", "Only show archives that were taken after the given date, in YYYYMMDD format."),
		Before:    getopt.StringLong("before", 'B', "", "Only show archives that were taken before the given date, in YYYYMMDD format."),
		To:        getopt.StringLong("to", 0, "", "Restore the archive in question to a different target, specified by UUID"),
	}

	OK := func(f string, l ...interface{}) {
		if *options.Raw {
			RawJSON(map[string]string{"ok": fmt.Sprintf(f, l...)})
			return
		}
		fmt.Printf("%s\n", tui.Green(fmt.Sprintf(f, l...)))
	}

	MSG := func(f string, l ...interface{}) {
		if !*options.Raw {
			fmt.Printf("\n%s\n", tui.Green(fmt.Sprintf(f, l...)))
		}
	}

	var command []string
	var opts = getopt.CommandLine
	args := os.Args
	for {
		opts.Parse(args)
		if opts.NArgs() == 0 {
			break
		}
		command = append(command, opts.Arg(0))
		args = opts.Args()
	}

	debug = *options.Debug
	DEBUG("shield cli starting up")
	if *options.Shield != "" {
		DEBUG("setting SHIELD_TARGET to '%s'", *options.Shield)
		os.Setenv("SHIELD_TARGET", *options.Shield)
	} else {
		DEBUG("SHIELD_TARGET is currently set to '%s'", *options.Shield)
	}

	if *options.Trace {
		DEBUG("enabling TRACE output")
		os.Setenv("SHIELD_TRACE", "1")
	}

	c := NewCommand().With(options)

	/*
	    ######  ########    ###    ######## ##     ##  ######
	   ##    ##    ##      ## ##      ##    ##     ## ##    ##
	   ##          ##     ##   ##     ##    ##     ## ##
	    ######     ##    ##     ##    ##    ##     ##  ######
	         ##    ##    #########    ##    ##     ##       ##
	   ##    ##    ##    ##     ##    ##    ##     ## ##    ##
	    ######     ##    ##     ##    ##     #######   ######
	*/

	c.Dispatch("status", func(opts Options, args []string) error {
		status, err := GetStatus()
		if err != nil {
			return err
		}

		if *opts.Raw {
			return RawJSON(map[string]string{
				"name":    status.Name,
				"version": status.Version,
			})
		}

		t := tui.NewReport()
		t.Add("Name", status.Name)
		t.Add("Version", status.Version)
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("stat", "status")

	/*
	   ########    ###    ########   ######   ######## ########
	      ##      ## ##   ##     ## ##    ##  ##          ##
	      ##     ##   ##  ##     ## ##        ##          ##
	      ##    ##     ## ########  ##   #### ######      ##
	      ##    ######### ##   ##   ##    ##  ##          ##
	      ##    ##     ## ##    ##  ##    ##  ##          ##
	      ##    ##     ## ##     ##  ######   ########    ##
	*/

	c.Dispatch("list targets", func(opts Options, args []string) error {
		DEBUG("running 'list targets' command")
		DEBUG("  for plugin: '%s'", *opts.Plugin)
		DEBUG("  show unused? %s", *opts.Unused)
		DEBUG("  show in-use? %s", *opts.Used)

		targets, err := GetTargets(TargetFilter{
			Name:   strings.Join(args, " "),
			Plugin: *opts.Plugin,
			Unused: MaybeBools(*opts.Unused, *opts.Used),
		})

		if err != nil {
			return err
		}

		if *opts.Raw {
			return RawJSON(targets)
		}

		t := tui.NewTable("Name", "Summary", "Plugin", "Remote IP", "Configuration")
		for _, target := range targets {
			t.Row(target, target.Name, target.Summary, target.Plugin, target.Agent, target.Endpoint)
		}
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("ls targets", "list targets")

	c.Dispatch("show target", func(opts Options, args []string) error {
		DEBUG("running 'show target' command")

		target, _, err := FindTarget(strings.Join(args, " "), *opts.Raw)
		if err != nil {
			return err
		}

		if *opts.Raw {
			return RawJSON(target)
		}

		t := tui.NewReport()
		t.Add("Name", target.Name)
		t.Add("Summary", target.Summary)
		t.Break()

		t.Add("Plugin", target.Plugin)
		t.Add("Configuration", target.Endpoint)
		t.Add("Remote IP", target.Agent)
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("view target", "show target")
	c.Alias("display target", "show target")
	c.Alias("list target", "show target")
	c.Alias("ls target", "show target")

	c.Dispatch("create target", func(opts Options, args []string) error {
		DEBUG("running 'create target' command")

		var err error
		var content string
		if *opts.Raw {
			content, err = readall(os.Stdin)
			if err != nil {
				return err
			}

		} else {
			in := tui.NewForm()
			in.NewField("Target Name", "name", "", "", tui.FieldIsRequired)
			in.NewField("Summary", "summary", "", "", tui.FieldIsOptional)
			in.NewField("Plugin Name", "plugin", "", "", tui.FieldIsRequired)
			in.NewField("Configuration", "endpoint", "", "", tui.FieldIsRequired)
			in.NewField("Remote IP:port", "agent", "", "", tui.FieldIsRequired)
			err := in.Show()
			if err != nil {
				return err
			}

			if !in.Confirm("Really create this target?") {
				return fmt.Errorf("Canceling...")
			}

			content, err = in.BuildContent()
			if err != nil {
				return err
			}
		}

		DEBUG("JSON:\n  %s\n", content)
		t, err := CreateTarget(content)
		if err != nil {
			return err
		}

		MSG("Created new target")
		return c.Execute("show", "target", t.UUID)
	})
	c.Alias("new target", "create target")
	c.Alias("create new target", "create target")
	c.Alias("make target", "create target")
	c.Alias("c t", "create target")

	c.Dispatch("edit target", func(opts Options, args []string) error {
		DEBUG("running 'edit target' command")

		t, id, err := FindTarget(strings.Join(args, " "), *opts.Raw)
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
			in.NewField("Target Name", "name", t.Name, "", tui.FieldIsRequired)
			in.NewField("Summary", "summary", t.Summary, "", tui.FieldIsOptional)
			in.NewField("Plugin Name", "plugin", t.Plugin, "", tui.FieldIsRequired)
			in.NewField("Configuration", "endpoint", t.Endpoint, "", tui.FieldIsRequired)
			in.NewField("Remote IP:port", "agent", t.Agent, "", tui.FieldIsRequired)

			if err := in.Show(); err != nil {
				return err
			}

			if !in.Confirm("Save these changes?") {
				return fmt.Errorf("Canceling...")
			}

			content, err = in.BuildContent()
			if err != nil {
				return err
			}
		}

		DEBUG("JSON:\n  %s\n", content)
		t, err = UpdateTarget(id, content)
		if err != nil {
			return err
		}

		MSG("Updated target")
		return c.Execute("show", "target", t.UUID)
	})
	c.Alias("update target", "edit target")

	c.Dispatch("delete target", func(opts Options, args []string) error {
		DEBUG("running 'delete target' command")

		target, id, err := FindTarget(strings.Join(args, " "), *opts.Raw)
		if err != nil {
			return err
		}

		if !*opts.Raw {
			t := tui.NewReport()
			t.Add("Name", target.Name)
			t.Add("Summary", target.Summary)
			t.Break()

			t.Add("Plugin", target.Plugin)
			t.Add("Configuration", target.Endpoint)
			t.Add("Remote IP", target.Agent)
			t.Output(os.Stdout)

			if !tui.Confirm("Really delete this target?") {
				return fmt.Errorf("Cancelling...")
			}
		}

		if err := DeleteTarget(id); err != nil {
			return err
		}

		OK("Deleted target")
		return nil
	})
	c.Alias("remove target", "delete target")
	c.Alias("rm target", "delete target")

	/*
	    ######   ######  ##     ## ######## ########  ##     ## ##       ########
	   ##    ## ##    ## ##     ## ##       ##     ## ##     ## ##       ##
	   ##       ##       ##     ## ##       ##     ## ##     ## ##       ##
	    ######  ##       ######### ######   ##     ## ##     ## ##       ######
	         ## ##       ##     ## ##       ##     ## ##     ## ##       ##
	   ##    ## ##    ## ##     ## ##       ##     ## ##     ## ##       ##
	    ######   ######  ##     ## ######## ########   #######  ######## ########
	*/

	c.Dispatch("list schedules", func(opts Options, args []string) error {
		DEBUG("running 'list schedules' command")
		DEBUG("  show unused? %s", *opts.Unused)
		DEBUG("  show in-use? %s", *opts.Used)

		schedules, err := GetSchedules(ScheduleFilter{
			Name:   strings.Join(args, " "),
			Unused: MaybeBools(*opts.Unused, *opts.Used),
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
	})
	c.Alias("ls schedules", "list schedules")

	c.Dispatch("show schedule", func(opts Options, args []string) error {
		DEBUG("running 'show schedule' command")

		schedule, _, err := FindSchedule(strings.Join(args, " "), *opts.Raw)
		if err != nil {
			return err
		}

		if *opts.Raw {
			return RawJSON(schedule)
		}

		t := tui.NewReport()
		t.Add("Name", schedule.Name)
		t.Add("Summary", schedule.Summary)
		t.Add("Timespec", schedule.When)
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("view schedule", "show schedule")
	c.Alias("display schedule", "show schedule")
	c.Alias("list schedule", "show schedule")
	c.Alias("ls schedule", "show schedule")

	c.Dispatch("create schedule", func(opts Options, args []string) error {
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
				return fmt.Errorf("Canceling...")
			}

			content, err = in.BuildContent()
			if err != nil {
				return err
			}
		}

		DEBUG("JSON:\n  %s\n", content)
		s, err := CreateSchedule(content)
		if err != nil {
			return err
		}

		MSG("Created new schedule")
		return c.Execute("show", "schedule", s.UUID)
	})
	c.Alias("new schedule", "create schedule")
	c.Alias("create new schedule", "create schedule")
	c.Alias("make schedule", "create schedule")
	c.Alias("c s", "create schedule")

	c.Dispatch("edit schedule", func(opts Options, args []string) error {
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
				return fmt.Errorf("Canceling...")
			}

			content, err = in.BuildContent()
			if err != nil {
				return err
			}
		}

		DEBUG("JSON:\n  %s\n", content)
		s, err = UpdateSchedule(id, content)
		if err != nil {
			return err
		}

		MSG("Updated schedule")
		return c.Execute("show", "schedule", s.UUID)
	})
	c.Alias("update schedule", "edit schedule")

	c.Dispatch("delete schedule", func(opts Options, args []string) error {
		DEBUG("running 'delete schedule' command")

		schedule, id, err := FindSchedule(strings.Join(args, " "), *opts.Raw)
		if err != nil {
			return err
		}

		if !*opts.Raw {
			t := tui.NewReport()
			t.Add("Name", schedule.Name)
			t.Add("Summary", schedule.Summary)
			t.Add("Timespec", schedule.When)
			t.Output(os.Stdout)

			if !tui.Confirm("Really delete this schedule?") {
				return fmt.Errorf("Cancelling...")
			}
		}

		if err := DeleteSchedule(id); err != nil {
			return err
		}

		OK("Deleted schedule")
		return nil
	})
	c.Alias("remove schedule", "delete schedule")
	c.Alias("rm schedule", "delete schedule")

	/*
	   ########  ######## ######## ######## ##    ## ######## ####  #######  ##    ##
	   ##     ## ##          ##    ##       ###   ##    ##     ##  ##     ## ###   ##
	   ##     ## ##          ##    ##       ####  ##    ##     ##  ##     ## ####  ##
	   ########  ######      ##    ######   ## ## ##    ##     ##  ##     ## ## ## ##
	   ##   ##   ##          ##    ##       ##  ####    ##     ##  ##     ## ##  ####
	   ##    ##  ##          ##    ##       ##   ###    ##     ##  ##     ## ##   ###
	   ##     ## ########    ##    ######## ##    ##    ##    ####  #######  ##    ##
	*/

	c.Dispatch("list retention policies", func(opts Options, args []string) error {
		DEBUG("running 'list retention policies' command")
		DEBUG("  show unused? %s", *opts.Unused)
		DEBUG("  show in-use? %s", *opts.Used)

		policies, err := GetRetentionPolicies(RetentionPolicyFilter{
			Name:   strings.Join(args, " "),
			Unused: MaybeBools(*opts.Unused, *opts.Used),
		})
		if err != nil {
			return err
		}

		if *opts.Raw {
			return RawJSON(policies)
		}

		t := tui.NewTable("Name", "Summary", "Expires in")
		for _, policy := range policies {
			t.Row(policy, policy.Name, policy.Summary, fmt.Sprintf("%d days", policy.Expires/86400))
		}
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("ls retention policies", "list retention policies")
	c.Alias("list policies", "list retention policies")
	c.Alias("ls policies", "list policies")

	c.Dispatch("show retention policy", func(opts Options, args []string) error {
		DEBUG("running 'show retention policy' command")

		policy, _, err := FindRetentionPolicy(strings.Join(args, " "), *opts.Raw)
		if err != nil {
			return err
		}

		if *opts.Raw {
			return RawJSON(policy)
		}

		t := tui.NewReport()
		t.Add("Name", policy.Name)
		t.Add("Summary", policy.Summary)
		t.Add("Expiration", fmt.Sprintf("%d days", policy.Expires/86400))
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("view retention policy", "show retention policy")
	c.Alias("display retention policy", "show retention policy")
	c.Alias("list retention policy", "show retention policy")
	c.Alias("show policy", "show retention policy")
	c.Alias("view policy", "show policy")
	c.Alias("display policy", "show policy")
	c.Alias("list policy", "show policy")

	c.Dispatch("create retention policy", func(opts Options, args []string) error {
		DEBUG("running 'create retention policy' command")

		var err error
		var content string
		if *opts.Raw {
			content, err = readall(os.Stdin)
			if err != nil {
				return err
			}

		} else {
			in := tui.NewForm()
			in.NewField("Policy Name", "name", "", "", tui.FieldIsRequired)
			in.NewField("Summary", "summary", "", "", tui.FieldIsOptional)
			in.NewField("Retention Timeframe, in days", "expires", "", "", FieldIsRetentionTimeframe)

			if err := in.Show(); err != nil {
				return err
			}

			if !in.Confirm("Really create this retention policy?") {
				return fmt.Errorf("Canceling...")
			}

			content, err = in.BuildContent()
			if err != nil {
				return err
			}
		}

		DEBUG("JSON:\n  %s\n", content)
		p, err := CreateRetentionPolicy(content)

		if err != nil {
			return err
		}

		MSG("Created new retention policy")
		return c.Execute("show", "retention", "policy", p.UUID)
	})
	c.Alias("new retention policy", "create retention policy")
	c.Alias("create new retention policy", "create retention policy")
	c.Alias("make retention policy", "create retention policy")
	c.Alias("create policy", "create retention policy")
	c.Alias("new policy", "create policy")
	c.Alias("create new policy", "create policy")
	c.Alias("make policy", "create policy")
	c.Alias("c p", "create policy")

	c.Dispatch("edit retention policy", func(opts Options, args []string) error {
		DEBUG("running 'edit retention policy' command")

		p, id, err := FindRetentionPolicy(strings.Join(args, " "), *opts.Raw)
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
			in.NewField("Policy Name", "name", p.Name, "", tui.FieldIsRequired)
			in.NewField("Summary", "summary", p.Summary, "", tui.FieldIsOptional)
			in.NewField("Retention Timeframe", "expires", p.Expires, fmt.Sprintf("%dd", p.Expires/86400), FieldIsRetentionTimeframe)

			if err = in.Show(); err != nil {
				return err
			}

			if !in.Confirm("Save these changes?") {
				return fmt.Errorf("Canceling...")
			}

			content, err = in.BuildContent()
			if err != nil {
				return err
			}
		}

		DEBUG("JSON:\n  %s\n", content)
		p, err = UpdateRetentionPolicy(id, content)
		if err != nil {
			return err
		}

		MSG("Updated retention policy")
		return c.Execute("show", "retention", "policy", p.UUID)
	})
	c.Alias("update retention policy", "edit retention policy")
	c.Alias("edit policy", "edit retention policy")
	c.Alias("update policy", "edit policy")

	c.Dispatch("delete retention policy", func(opts Options, args []string) error {
		DEBUG("running 'delete retention policy' command")

		policy, id, err := FindRetentionPolicy(strings.Join(args, " "), *opts.Raw)
		if err != nil {
			return err
		}

		if !*opts.Raw {
			t := tui.NewReport()
			t.Add("Name", policy.Name)
			t.Add("Summary", policy.Summary)
			t.Add("Expiration", fmt.Sprintf("%d days", policy.Expires/86400))
			t.Output(os.Stdout)

			if !tui.Confirm("Really delete this retention policy?") {
				return fmt.Errorf("Cancelling...")
			}
		}

		if err := DeleteRetentionPolicy(id); err != nil {
			return err
		}

		OK("Deleted retention policy")
		return nil
	})
	c.Alias("remove retention policy", "delete retention policy")
	c.Alias("rm retention policy", "delete retention policy")
	c.Alias("delete policy", "delete retention policy")
	c.Alias("remove policy", "delete policy")
	c.Alias("rm policy", "delete policy")

	/*
	    ######  ########  #######  ########  ########
	   ##    ##    ##    ##     ## ##     ## ##
	   ##          ##    ##     ## ##     ## ##
	    ######     ##    ##     ## ########  ######
	         ##    ##    ##     ## ##   ##   ##
	   ##    ##    ##    ##     ## ##    ##  ##
	    ######     ##     #######  ##     ## ########
	*/

	c.Dispatch("list stores", func(opts Options, args []string) error {
		DEBUG("running 'list stores' command")
		DEBUG("  for plugin: '%s'", *opts.Plugin)
		DEBUG("  show unused? %s", *opts.Unused)
		DEBUG("  show in-use? %s", *opts.Used)

		stores, err := GetStores(StoreFilter{
			Name:   strings.Join(args, " "),
			Plugin: *opts.Plugin,
			Unused: MaybeBools(*opts.Unused, *opts.Used),
		})
		if err != nil {
			return err
		}

		if *opts.Raw {
			return RawJSON(stores)
		}

		t := tui.NewTable("Name", "Summary", "Plugin", "Configuration")
		for _, store := range stores {
			t.Row(store, store.Name, store.Summary, store.Plugin, store.Endpoint)
		}
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("ls stores", "list stores")

	c.Dispatch("show store", func(opts Options, args []string) error {
		DEBUG("running 'show store' command")

		store, _, err := FindStore(strings.Join(args, " "), *opts.Raw)
		if err != nil {
			return err
		}

		if *opts.Raw {
			return RawJSON(store)
		}

		t := tui.NewReport()
		t.Add("Name", store.Name)
		t.Add("Summary", store.Summary)
		t.Break()

		t.Add("Plugin", store.Plugin)
		t.Add("Configuration", store.Endpoint)
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("view store", "show store")
	c.Alias("display store", "show store")
	c.Alias("list store", "show store")
	c.Alias("ls store", "show store")

	c.Dispatch("create store", func(opts Options, args []string) error {
		DEBUG("running 'create store' command")

		var err error
		var content string
		if *opts.Raw {
			content, err = readall(os.Stdin)
			if err != nil {
				return err
			}

		} else {
			in := tui.NewForm()
			in.NewField("Store Name", "name", "", "", tui.FieldIsRequired)
			in.NewField("Summary", "summary", "", "", tui.FieldIsOptional)
			in.NewField("Plugin Name", "plugin", "", "", tui.FieldIsRequired)
			in.NewField("Configuration (JSON)", "endpoint", "", "", tui.FieldIsRequired)

			if err := in.Show(); err != nil {
				return err
			}

			if !in.Confirm("Really create this archive store?") {
				return fmt.Errorf("Canceling...")
			}

			content, err = in.BuildContent()
			if err != nil {
				return err
			}
		}

		DEBUG("JSON:\n  %s\n", content)
		s, err := CreateStore(content)

		if err != nil {
			return err
		}

		MSG("Created new store")
		return c.Execute("show", "store", s.UUID)
	})
	c.Alias("new store", "create store")
	c.Alias("create new store", "create store")
	c.Alias("make store", "create store")
	c.Alias("c st", "create store")

	c.Dispatch("edit store", func(opts Options, args []string) error {
		DEBUG("running 'edit store' command")

		s, id, err := FindStore(strings.Join(args, " "), *opts.Raw)
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
			in.NewField("Store Name", "name", s.Name, "", tui.FieldIsRequired)
			in.NewField("Summary", "summary", s.Summary, "", tui.FieldIsOptional)
			in.NewField("Plugin Name", "plugin", s.Plugin, "", tui.FieldIsRequired)
			in.NewField("Configuration (JSON)", "endpoint", s.Endpoint, "", tui.FieldIsRequired)

			err = in.Show()
			if err != nil {
				return err
			}

			if !in.Confirm("Save these changes?") {
				return fmt.Errorf("Canceling...")
			}

			content, err = in.BuildContent()
			if err != nil {
				return err
			}
		}

		DEBUG("JSON:\n  %s\n", content)
		s, err = UpdateStore(id, content)
		if err != nil {
			return err
		}

		MSG("Updated store")
		return c.Execute("show", "store", s.UUID)
	})
	c.Alias("update store", "edit store")

	c.Dispatch("delete store", func(opts Options, args []string) error {
		DEBUG("running 'delete store' command")

		store, id, err := FindStore(strings.Join(args, " "), *opts.Raw)
		if err != nil {
			return err
		}

		if !*opts.Raw {
			t := tui.NewReport()
			t.Add("Name", store.Name)
			t.Add("Summary", store.Summary)
			t.Break()

			t.Add("Plugin", store.Plugin)
			t.Add("Configuration", store.Endpoint)
			t.Output(os.Stdout)

			if !tui.Confirm("Really delete this store?") {
				return fmt.Errorf("Cancelling...")
			}
		}

		if err := DeleteStore(id); err != nil {
			return err
		}

		OK("Deleted store")
		return nil
	})
	c.Alias("remove store", "delete store")
	c.Alias("rm store", "delete store")

	/*
	         ##  #######  ########
	         ## ##     ## ##     ##
	         ## ##     ## ##     ##
	         ## ##     ## ########
	   ##    ## ##     ## ##     ##
	   ##    ## ##     ## ##     ##
	    ######   #######  ########
	*/

	c.Dispatch("list jobs", func(opts Options, args []string) error {
		DEBUG("running 'list jobs' command")
		DEBUG("  for target:      '%s'", *opts.Target)
		DEBUG("  for store:       '%s'", *opts.Store)
		DEBUG("  for schedule:    '%s'", *opts.Schedule)
		DEBUG("  for ret. policy: '%s'", *opts.Retention)
		DEBUG("  show paused?      %s", *opts.Paused)
		DEBUG("  show unpaused?    %s", *opts.Unpaused)

		jobs, err := GetJobs(JobFilter{
			Name:      strings.Join(args, " "),
			Paused:    MaybeBools(*opts.Unpaused, *opts.Paused),
			Target:    *opts.Target,
			Store:     *opts.Store,
			Schedule:  *opts.Schedule,
			Retention: *opts.Retention,
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
				job.RetentionName, job.ScheduleName, job.Agent, job.TargetEndpoint)
		}
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("ls jobs", "list jobs")
	c.Alias("ls j", "list jobs")

	c.Dispatch("show job", func(opts Options, args []string) error {
		DEBUG("running 'show job' command")

		job, _, err := FindJob(strings.Join(args, " "), *opts.Raw)
		if err != nil {
			return nil
		}

		if *opts.Raw {
			return RawJSON(job)
		}

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
		return nil
	})
	c.Alias("view job", "show job")
	c.Alias("display job", "show job")
	c.Alias("list job", "show job")
	c.Alias("ls job", "show job")

	c.Dispatch("create job", func(opts Options, args []string) error {
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

			in.NewField("Paused?", "paused", "yes", "", tui.FieldIsBoolean)
			err := in.Show()
			if err != nil {
				return err
			}

			if !in.Confirm("Really create this backup job?") {
				return fmt.Errorf("Canceling...")
			}

			content, err = in.BuildContent()
			if err != nil {
				return err
			}
		}

		DEBUG("JSON:\n  %s\n", content)
		job, err := CreateJob(content)
		if err != nil {
			return err
		}

		MSG("Created new job")
		return c.Execute("show", "job", job.UUID)
	})
	c.Alias("new job", "create job")
	c.Alias("create new job", "create job")
	c.Alias("make job", "create job")
	c.Alias("c j", "create job")

	c.Dispatch("edit job", func(opts Options, args []string) error {
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

			paused := "no"
			if j.Paused {
				paused = "yes"
			}
			in := tui.NewForm()
			in.NewField("Job Name", "name", j.Name, "", tui.FieldIsRequired)
			in.NewField("Summary", "summary", j.Summary, "", tui.FieldIsOptional)
			in.NewField("Store", "store", j.StoreUUID, j.StoreName, FieldIsStoreUUID)
			in.NewField("Target", "target", j.TargetUUID, j.TargetName, FieldIsTargetUUID)
			in.NewField("Retention Policy", "retention", j.RetentionUUID, fmt.Sprintf("%s - %dd", j.RetentionName, j.Expiry/86400), FieldIsRetentionPolicyUUID)
			in.NewField("Schedule", "schedule", j.ScheduleUUID, fmt.Sprintf("%s - %s", j.ScheduleName, j.ScheduleWhen), FieldIsScheduleUUID)
			in.NewField("Paused?", "paused", paused, "", tui.FieldIsBoolean)

			if err = in.Show(); err != nil {
				return err
			}

			if !in.Confirm("Save these changes?") {
				return fmt.Errorf("Canceling...")
			}

			content, err = in.BuildContent()
			if err != nil {
				return err
			}
		}

		DEBUG("JSON:\n  %s\n", content)
		j, err = UpdateJob(id, content)
		if err != nil {
			return err
		}

		MSG("Updated job")
		return c.Execute("show", "job", j.UUID)
	})
	c.Alias("update job", "edit job")

	c.Dispatch("delete job", func(opts Options, args []string) error {
		DEBUG("running 'delete job' command")

		job, id, err := FindJob(strings.Join(args, " "), *opts.Raw)
		if err != nil {
			return err
		}

		if !*opts.Raw {
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

			if !tui.Confirm("Really delete this backup job?") {
				return fmt.Errorf("Cancelling...")
			}
		}

		if err := DeleteJob(id); err != nil {
			return err
		}

		OK("Deleted job")
		return nil
	})
	c.Alias("remove job", "delete job")
	c.Alias("rm job", "delete job")

	c.Dispatch("pause job", func(opts Options, args []string) error {
		DEBUG("running 'pause job' command")

		_, id, err := FindJob(strings.Join(args, " "), *opts.Raw)
		if err != nil {
			return err
		}
		if err := PauseJob(id); err != nil {
			return err
		}

		return nil
	})

	c.Dispatch("unpause job", func(opts Options, args []string) error {
		DEBUG("running 'unpause job' command")

		_, id, err := FindJob(strings.Join(args, " "), *opts.Raw)
		if err != nil {
			return err
		}
		if err := UnpauseJob(id); err != nil {
			return err
		}

		OK("Unpaused job")
		return nil
	})

	c.Dispatch("run job", func(opts Options, args []string) error {
		DEBUG("running 'run job' command")

		_, id, err := FindJob(strings.Join(args, " "), *opts.Raw)
		if err != nil {
			return err
		}
		if err := RunJob(id, fmt.Sprintf(`{"owner":"%s@%s"}`,
			os.Getenv("USER"), os.Getenv("HOSTNAME"))); err != nil {
			return err
		}

		OK("Scheduled immediate run of job")
		return nil
	})

	/*
	   ########    ###     ######  ##    ##
	      ##      ## ##   ##    ## ##   ##
	      ##     ##   ##  ##       ##  ##
	      ##    ##     ##  ######  #####
	      ##    #########       ## ##  ##
	      ##    ##     ## ##    ## ##   ##
	      ##    ##     ##  ######  ##    ##
	*/

	c.Dispatch("list tasks", func(opts Options, args []string) error {
		DEBUG("running 'list tasks' command")

		if *options.Status == "" {
			*options.Status = "running"
		}
		if *options.Status == "all" || *options.All {
			*options.Status = ""
		}
		DEBUG("  for status: '%s'", *opts.Status)

		tasks, err := GetTasks(TaskFilter{
			Status: *options.Status,
		})
		if err != nil {
			return err
		}

		if *opts.Raw {
			return RawJSON(tasks)
		}

		job := map[string]Job{}
		jobs, _ := GetJobs(JobFilter{})
		for _, j := range jobs {
			job[j.UUID] = j
		}

		t := tui.NewTable("UUID", "Owner", "Type", "Remote IP", "Status", "Started", "Stopped")
		for _, task := range tasks {
			started := "(pending)"
			if !task.StartedAt.IsZero() {
				started = task.StartedAt.Format(time.RFC1123Z)
			}

			stopped := "(running)"
			if !task.StoppedAt.IsZero() {
				stopped = task.StoppedAt.Format(time.RFC1123Z)
			}

			t.Row(task, task.UUID, task.Owner, task.Op, job[task.JobUUID].Agent, task.Status, started, stopped)
		}
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("ls tasks", "list tasks")

	c.Dispatch("show task", func(opts Options, args []string) error {
		DEBUG("running 'show task' command")

		require(len(args) == 1, "shield show task <UUID>")
		id := uuid.Parse(args[0])
		DEBUG("  task UUID = '%s'", id)

		task, err := GetTask(id)
		if err != nil {
			return err
		}

		if *opts.Raw {
			return RawJSON(task)
		}

		t := tui.NewReport()
		t.Add("UUID", task.UUID)
		t.Add("Owner", task.Owner)
		t.Add("Type", task.Op)
		t.Add("Status", task.Status)
		t.Break()

		started := "(pending)"
		if !task.StartedAt.IsZero() {
			started = task.StartedAt.Format(time.RFC1123Z)
		}
		stopped := "(running)"
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
		return nil
	})
	c.Alias("view task", "show task")
	c.Alias("display task", "show task")
	c.Alias("list task", "show task")
	c.Alias("ls task", "show task")

	c.Dispatch("cancel task", func(opts Options, args []string) error {
		DEBUG("running 'cancel task' command")

		require(len(args) == 1, "shield cancel task <UUID>")
		id := uuid.Parse(args[0])
		DEBUG("  task UUID = '%s'", id)

		err := CancelTask(id)
		if err != nil {
			return err
		}

		OK("Cancelled task '%s'\n", id)
		return nil
	})
	c.Alias("stop task", "cancel task")

	/*
	      ###    ########   ######  ##     ## #### ##     ## ########
	     ## ##   ##     ## ##    ## ##     ##  ##  ##     ## ##
	    ##   ##  ##     ## ##       ##     ##  ##  ##     ## ##
	   ##     ## ########  ##       #########  ##  ##     ## ######
	   ######### ##   ##   ##       ##     ##  ##   ##   ##  ##
	   ##     ## ##    ##  ##    ## ##     ##  ##    ## ##   ##
	   ##     ## ##     ##  ######  ##     ## ####    ###    ########
	*/

	c.Dispatch("list archives", func(opts Options, args []string) error {
		DEBUG("running 'list archives' command")

		if *options.Status == "" {
			*options.Status = "valid"
		}
		if *options.Status == "all" || *options.All {
			*options.Status = ""
		}
		DEBUG("  for status: '%s'", *opts.Status)

		archives, err := GetArchives(ArchiveFilter{
			Target: *options.Target,
			Store:  *options.Store,
			Before: *options.Before,
			After:  *options.After,
			Status: *options.Status,
		})
		if err != nil {
			return err
		}

		if *opts.Raw {
			return RawJSON(archives)
		}

		// Map out the target names, for prettier output
		target := map[string]Target{}
		targets, _ := GetTargets(TargetFilter{})
		for _, t := range targets {
			target[t.UUID] = t
		}

		// Map out the store names, for prettier output
		store := map[string]Store{}
		stores, _ := GetStores(StoreFilter{})
		for _, s := range stores {
			store[s.UUID] = s
		}

		t := tui.NewTable("UUID", "Target", "Remote IP", "Store", "Taken at", "Expires at", "Status", "Notes")
		for _, archive := range archives {
			if *opts.Target != "" && archive.TargetUUID != *opts.Target {
				continue
			}
			if *opts.Store != "" && archive.StoreUUID != *opts.Store {
				continue
			}

			t.Row(archive, archive.UUID,
				fmt.Sprintf("%s (%s)", target[archive.TargetUUID].Name, archive.TargetPlugin),
				target[archive.TargetUUID].Agent,
				fmt.Sprintf("%s (%s)", store[archive.StoreUUID].Name, archive.StorePlugin),
				archive.TakenAt.Format(time.RFC1123Z),
				archive.ExpiresAt.Format(time.RFC1123Z),
				archive.Status, archive.Notes)
		}
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("ls archives", "list archives")

	c.Dispatch("show archive", func(opts Options, args []string) error {
		DEBUG("running 'show archive' command")

		require(len(args) == 1, "shield show archive <UUID>")
		id := uuid.Parse(args[0])
		DEBUG("  archive UUID = '%s'", id)

		archive, err := GetArchive(id)
		if err != nil {
			return err
		}

		if *opts.Raw {
			return RawJSON(archive)
		}

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
		return nil
	})
	c.Alias("view archive", "show archive")
	c.Alias("display archive", "show archive")
	c.Alias("list archive", "show archive")
	c.Alias("ls archive", "show archive")

	c.Dispatch("restore archive", func(opts Options, args []string) error {
		DEBUG("running 'restore archive' command")

		require(len(args) == 1, "USAGE: shield restore archive <UUID>")
		id := uuid.Parse(args[0])
		DEBUG("  archive UUID = '%s'", id)

		targetJSON := "{}"
		toTargetJSONmsg := ""
		if opts.Target != nil && *opts.Target != "" {
			targetJSON = *opts.Target
			toTargetJSONmsg = fmt.Sprintf("to target '%s'", targetJSON)
		}

		err := RestoreArchive(id, targetJSON)
		if err != nil {
			return err
		}

		OK("Scheduled immedate restore of archive '%s' %s", id, toTargetJSONmsg)
		return nil
	})

	c.Dispatch("delete archive", func(opts Options, args []string) error {
		DEBUG("running 'delete archive' command")

		require(len(args) == 1, "USAGE: shield delete archive <UUID>")
		id := uuid.Parse(args[0])
		DEBUG("  archive UUID = '%s'", id)

		archive, err := GetArchive(id)
		if err != nil {
			return err
		}

		if !*opts.Raw {
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

			if !tui.Confirm("Really delete this archive?") {
				return fmt.Errorf("Cancelling...")
			}
		}

		if err := DeleteArchive(id); err != nil {
			return err
		}

		OK("Deleted archive")
		return nil
	})
	c.Alias("remove archive", "delete archive")
	c.Alias("rm archive", "delete archive")

	/**************************************************************************/

	if err := c.Execute(command...); err != nil {
		if *options.Raw {
			_ = RawJSON(map[string]string{"error": err.Error()})
		} else {
			fmt.Fprintf(os.Stderr, "%s\n", tui.Red(err))
		}
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}
