package main

import (
	"fmt"
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

		t := tui.NewTable("UUID", "Target Name", "Summary", "Plugin", "Target Agent IP", "Endpoint")
		for _, target := range targets {
			t.Row(target, target.UUID, target.Name, target.Summary, target.Plugin, target.Agent, target.Endpoint)
		}
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("ls targets", "list targets")

	c.Dispatch("show target", func(opts Options, args []string) error {
		DEBUG("running 'show target' command")

		target, _, err := FindTarget(args...)
		if err != nil {
			return err
		}

		t := tui.NewReport()
		t.Add("UUID", target.UUID)
		t.Add("Name", target.Name)
		t.Add("Summary", target.Summary)
		t.Break()

		t.Add("Plugin", target.Plugin)
		t.Add("Endpoint", target.Endpoint)
		t.Add("SHIELD Agent", target.Agent)
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("view target", "show target")
	c.Alias("display target", "show target")
	c.Alias("list target", "show target")
	c.Alias("ls target", "show target")

	c.Dispatch("create target", func(opts Options, args []string) error {
		DEBUG("running 'create target' command")

		in := tui.NewForm()
		in.NewField("Target name", "name", "", "", tui.FieldIsRequired)
		in.NewField("Target summary", "summary", "", "", tui.FieldIsOptional)
		in.NewField("Target plugin", "plugin", "", "", tui.FieldIsRequired)
		in.NewField("Target endpoint", "endpoint", "", "", tui.FieldIsRequired)
		in.NewField("Target agent", "agent", "", "", tui.FieldIsRequired)
		err := in.Show()
		if err != nil {
			return err
		}

		if !tui.Confirm("Really create this target?") {
			return fmt.Errorf("Canceling...")
		}

		content, err := in.BuildContent()
		if err != nil {
			return err
		}

		DEBUG("JSON:\n  %s\n", content)
		t, err := CreateTarget(content)
		if err != nil {
			return err
		}

		fmt.Printf("Created new target.\n")
		return c.Execute("show", "target", t.UUID)
	})
	c.Alias("new target", "create target")
	c.Alias("create new target", "create target")
	c.Alias("make target", "create target")
	c.Alias("c t", "create target")

	c.Dispatch("edit target", func(opts Options, args []string) error {
		DEBUG("running 'edit target' command")

		t, id, err := FindTarget(args...)
		if err != nil {
			return err
		}

		in := tui.NewForm()
		in.NewField("Target name", "name", t.Name, "", tui.FieldIsRequired)
		in.NewField("Target summary", "summary", t.Summary, "", tui.FieldIsOptional)
		in.NewField("Target plugin", "plugin", t.Plugin, "", tui.FieldIsRequired)
		in.NewField("Target endpoint", "endpoint", t.Endpoint, "", tui.FieldIsRequired)
		in.NewField("Target agent", "agent", t.Agent, "", tui.FieldIsRequired)

		if err := in.Show(); err != nil {
			return err
		}

		content, err := in.BuildContent()
		if err != nil {
			return err
		}

		DEBUG("JSON:\n  %s\n", content)
		t, err = UpdateTarget(id, content)
		if err != nil {
			return err
		}
		fmt.Printf("Updated target.\n")
		return c.Execute("show", "target", t.UUID)
	})
	c.Alias("update target", "edit target")

	c.Dispatch("delete target", func(opts Options, args []string) error {
		DEBUG("running 'delete target' command")

		_, id, err := FindTarget(args...)
		if err != nil {
			return err
		}
		if err := DeleteTarget(id); err != nil {
			return err
		}
		fmt.Printf("Deleted target '%s'\n", id)
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
		t := tui.NewTable("UUID", "Name", "Summary", "Frequency / Interval (UTC)")
		for _, schedule := range schedules {
			t.Row(schedule, schedule.UUID, schedule.Name, schedule.Summary, schedule.When)
		}
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("ls schedules", "list schedules")

	c.Dispatch("show schedule", func(opts Options, args []string) error {
		DEBUG("running 'show schedule' command")

		schedule, _, err := FindSchedule(args...)
		if err != nil {
			return err
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

		in := tui.NewForm()
		in.NewField("Schedule name", "name", "", "", tui.FieldIsRequired)
		in.NewField("Schedule summary", "summary", "", "", tui.FieldIsOptional)
		in.NewField("When to run schedule (eg daily at 4:00)", "when", "", "", tui.FieldIsRequired)

		if err := in.Show(); err != nil {
			return err
		}

		if !tui.Confirm("Really create this schedule?") {
			return fmt.Errorf("Canceling...")
		}

		content, err := in.BuildContent()
		if err != nil {
			return err
		}

		DEBUG("JSON:\n  %s\n", content)
		s, err := CreateSchedule(content)
		if err != nil {
			return err
		}

		fmt.Printf("Created new schedule.\n")
		return c.Execute("show", "schedule", s.UUID)
	})
	c.Alias("new schedule", "create schedule")
	c.Alias("create new schedule", "create schedule")
	c.Alias("make schedule", "create schedule")
	c.Alias("c s", "create schedule")

	c.Dispatch("edit schedule", func(opts Options, args []string) error {
		DEBUG("running 'edit schedule' command")

		s, id, err := FindSchedule(args...)
		if err != nil {
			return err
		}

		in := tui.NewForm()
		in.NewField("Schedule name", "name", s.Name, "", tui.FieldIsRequired)
		in.NewField("Schedule summary", "summary", s.Summary, "", tui.FieldIsOptional)
		in.NewField("When to run schedule (eg daily at 4:00)", "when", s.When, "", tui.FieldIsRequired)

		if err = in.Show(); err != nil {
			return err
		}

		content, err := in.BuildContent()
		if err != nil {
			return err
		}

		DEBUG("JSON:\n  %s\n", content)
		s, err = UpdateSchedule(id, content)
		if err != nil {
			return err
		}
		fmt.Printf("Updated schedule.\n")
		return c.Execute("show", "schedule", s.UUID)
	})
	c.Alias("update schedule", "edit schedule")

	c.Dispatch("delete schedule", func(opts Options, args []string) error {
		DEBUG("running 'delete schedule' command")

		_, id, err := FindSchedule(args...)
		if err != nil {
			return err
		}

		if err := DeleteSchedule(id); err != nil {
			return err
		}

		fmt.Printf("Deleted schedule '%s'\n", id)
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
		t := tui.NewTable("UUID", "Name", "Summary", "Expires in")
		for _, policy := range policies {
			t.Row(policy, policy.UUID, policy.Name, policy.Summary, fmt.Sprintf("%d days", policy.Expires/86400))
		}
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("ls retention policies", "list retention policies")
	c.Alias("list policies", "list retention policies")
	c.Alias("ls policies", "list policies")

	c.Dispatch("show retention policy", func(opts Options, args []string) error {
		DEBUG("running 'show retention policy' command")

		policy, _, err := FindRetentionPolicy(args...)
		if err != nil {
			return err
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

		in := tui.NewForm()
		in.NewField("Policy name", "name", "", "", tui.FieldIsRequired)
		in.NewField("Policy summary", "summary", "", "", tui.FieldIsOptional)
		in.NewField("Retention Timeframe (in days)", "expires", "", "", FieldIsRetentionTimeframe)

		if err := in.Show(); err != nil {
			return err
		}

		if !tui.Confirm("Really create this retention policy?") {
			return fmt.Errorf("Canceling...")
		}

		content, err := in.BuildContent()
		if err != nil {
			return err
		}

		DEBUG("JSON:\n  %s\n", content)
		p, err := CreateRetentionPolicy(content)

		if err != nil {
			return err
		}
		fmt.Printf("Created new retention policy.\n")
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

		p, id, err := FindRetentionPolicy(args...)
		if err != nil {
			return err
		}

		in := tui.NewForm()
		in.NewField("Policy name", "name", p.Name, "", tui.FieldIsRequired)
		in.NewField("Policy summary", "summary", p.Summary, "", tui.FieldIsOptional)
		in.NewField("Retention Timeframe", "expires", p.Expires, fmt.Sprintf("%dd", p.Expires / 86400), FieldIsRetentionTimeframe)

		if err = in.Show(); err != nil {
			return err
		}

		content, err := in.BuildContent()
		if err != nil {
			return err
		}

		DEBUG("JSON:\n  %s\n", content)
		p, err = UpdateRetentionPolicy(id, content)
		if err != nil {
			return err
		}
		fmt.Printf("Updated policy.\n")
		return c.Execute("show", "retention", "policy", p.UUID)
	})
	c.Alias("update retention policy", "edit retention policy")
	c.Alias("edit policy", "edit retention policy")
	c.Alias("update policy", "edit policy")

	c.Dispatch("delete retention policy", func(opts Options, args []string) error {
		DEBUG("running 'delete retention policy' command")

		_, id, err := FindRetentionPolicy(args...)
		if err != nil {
			return err
		}

		if err := DeleteRetentionPolicy(id); err != nil {
			return err
		}
		fmt.Printf("Deleted retention policy '%s'\n", id)
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
		t := tui.NewTable("UUID", "Name", "Summary", "Plugin", "Endpoint")
		for _, store := range stores {
			t.Row(store, store.UUID, store.Name, store.Summary, store.Plugin, store.Endpoint)
		}
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("ls stores", "list stores")

	c.Dispatch("show store", func(opts Options, args []string) error {
		DEBUG("running 'show store' command")

		store, _, err := FindStore(args...)
		if err != nil {
			return err
		}

		t := tui.NewReport()
		t.Add("Name", store.Name)
		t.Add("Summary", store.Summary)
		t.Break()

		t.Add("Plugin", store.Plugin)
		t.Add("Endpoint", store.Endpoint)
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("view store", "show store")
	c.Alias("display store", "show store")
	c.Alias("list store", "show store")
	c.Alias("ls store", "show store")

	c.Dispatch("create store", func(opts Options, args []string) error {
		DEBUG("running 'create store' command")

		in := tui.NewForm()
		in.NewField("Store name", "name", "", "", tui.FieldIsRequired)
		in.NewField("Store summary", "summary", "", "", tui.FieldIsOptional)
		in.NewField("Plugin name", "plugin", "", "", tui.FieldIsRequired)
		in.NewField("Endpoint (JSON)", "endpoint", "", "", tui.FieldIsRequired)

		if err := in.Show(); err != nil {
			return err
		}

		if !tui.Confirm("Really create this archive store?") {
			return fmt.Errorf("Canceling...")
		}

		content, err := in.BuildContent()
		if err != nil {
			return err
		}

		DEBUG("JSON:\n  %s\n", content)
		s, err := CreateStore(content)

		if err != nil {
			return err
		}
		fmt.Printf("Created new store.\n")

		return c.Execute("show", "store", s.UUID)
	})
	c.Alias("new store", "create store")
	c.Alias("create new store", "create store")
	c.Alias("make store", "create store")
	c.Alias("c st", "create store")

	c.Dispatch("edit store", func(opts Options, args []string) error {
		DEBUG("running 'edit store' command")

		s, id, err := FindStore(args...)
		if err != nil {
			return err
		}

		in := tui.NewForm()
		in.NewField("Store name", "name", s.Name, "", tui.FieldIsRequired)
		in.NewField("Store summary", "summary", s.Summary, "", tui.FieldIsOptional)
		in.NewField("Plugin name", "plugin", s.Plugin, "", tui.FieldIsRequired)
		in.NewField("Endpoint (JSON)", "endpoint", s.Endpoint, "", tui.FieldIsRequired)

		err = in.Show()
		if err != nil {
			return err
		}

		content, err := in.BuildContent()
		if err != nil {
			return err
		}

		DEBUG("JSON:\n  %s\n", content)
		s, err = UpdateStore(id, content)
		if err != nil {
			return err
		}
		fmt.Printf("Updated store.\n")
		return c.Execute("show", "store", s.UUID)
	})
	c.Alias("update store", "edit store")

	c.Dispatch("delete store", func(opts Options, args []string) error {
		DEBUG("running 'delete store' command")

		_, id, err := FindStore(args...)
		if err != nil {
			return err
		}

		if err := DeleteStore(id); err != nil {
			return err
		}
		fmt.Printf("Deleted store '%s'\n", id)
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
		t := tui.NewTable("Name", "P?", "Summary", "Retention Policy", "Schedule", "Target Agent IP", "Target")
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

		job, _, err := FindJob(args...)
		if err != nil {
			return nil
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
		t.Add("SHIELD Agent", job.Agent)
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

		in := tui.NewForm()
		in.NewField("Job name", "name", "", "", tui.FieldIsRequired)
		in.NewField("Job summary", "summary", "", "", tui.FieldIsOptional)

		in.NewField("Store", "store", "", "", FieldIsStoreUUID)
		in.NewField("Target", "target", "", "", FieldIsTargetUUID)
		in.NewField("Retention Policy", "retention", "", "", FieldIsRetentionPolicyUUID)
		in.NewField("Schedule", "schedule", "", "", FieldIsScheduleUUID)

		in.NewField("Paused? [Y/n]", "paused", "yes", "", tui.FieldIsBoolean)
		err := in.Show()
		if err != nil {
			return err
		}

		if !tui.Confirm("Really create this backup job?") {
			return fmt.Errorf("Canceling...")
		}

		content, err := in.BuildContent()
		if err != nil {
			return err
		}

		DEBUG("JSON:\n  %s\n", content)
		job, err := CreateJob(content)
		if err != nil {
			return err
		}

		return c.Execute("show", "job", job.UUID)
	})
	c.Alias("new job", "create job")
	c.Alias("create new job", "create job")
	c.Alias("make job", "create job")
	c.Alias("c j", "create job")

	c.Dispatch("edit job", func(opts Options, args []string) error {
		DEBUG("running 'edit job' command")

		j, id, err := FindJob(args...)
		if err != nil {
			return err
		}
		paused := "no"
		if j.Paused {
			paused = "yes"
		}

		in := tui.NewForm()
		in.NewField("Job name", "name", j.Name, "", tui.FieldIsRequired)
		in.NewField("Job summary", "summary", j.Summary, "", tui.FieldIsOptional)
		in.NewField("Store", "store", j.StoreUUID, j.StoreName, FieldIsStoreUUID)
		in.NewField("Target", "target", j.TargetUUID, j.TargetName, FieldIsTargetUUID)
		in.NewField("Retention Policy", "retention", j.RetentionUUID, fmt.Sprintf("%s - %dd", j.RetentionName, j.Expiry / 86400), FieldIsRetentionPolicyUUID)
		in.NewField("Schedule", "schedule", j.ScheduleUUID, fmt.Sprintf("%s - %s", j.ScheduleName, j.ScheduleWhen), FieldIsScheduleUUID)

		in.NewField("Should the job be paused? (Y/n)", "paused", paused, "", tui.FieldIsBoolean)
		err = in.Show()
		if err != nil {
			return err
		}

		content, err := in.BuildContent()
		if err != nil {
			return err
		}

		DEBUG("JSON:\n  %s\n", content)
		j, err = UpdateJob(id, content)
		if err != nil {
			return err
		}

		fmt.Printf("Updated job.\n")
		return c.Execute("show", "job", j.UUID)
	})
	c.Alias("update job", "edit job")

	c.Dispatch("delete job", func(opts Options, args []string) error {
		DEBUG("running 'delete job' command")

		_, id, err := FindJob(args...)
		if err != nil {
			return err
		}
		if err := DeleteJob(id); err != nil {
			return err
		}
		fmt.Printf("Deleted job '%s'\n", id)
		return nil
	})
	c.Alias("remove job", "delete job")
	c.Alias("rm job", "delete job")

	c.Dispatch("pause job", func(opts Options, args []string) error {
		DEBUG("running 'pause job' command")

		_, id, err := FindJob(args...)
		if err != nil {
			return err
		}
		if err := PauseJob(id); err != nil {
			return err
		}
		fmt.Printf("Paused job\n")
		return nil
	})

	c.Dispatch("unpause job", func(opts Options, args []string) error {
		DEBUG("running 'unpause job' command")

		_, id, err := FindJob(args...)
		if err != nil {
			return err
		}
		if err := UnpauseJob(id); err != nil {
			return err
		}
		fmt.Printf("Unpaused job\n")
		return nil
	})

	c.Dispatch("run job", func(opts Options, args []string) error {
		DEBUG("running 'run job' command")

		_, id, err := FindJob(args...)
		if err != nil {
			return err
		}
		if err := RunJob(id, fmt.Sprintf(`{"owner":"%s@%s"}`,
			os.Getenv("USER"), os.Getenv("HOSTNAME"))); err != nil {
			return err
		}
		fmt.Printf("Scheduled immediate run of job\n")
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

		job := map[string]Job{}
		jobs, _ := GetJobs(JobFilter{})
		for _, j := range jobs {
			job[j.UUID] = j
		}

		t := tui.NewTable("UUID", "Owner", "Type", "Target Agent IP", "Status", "Started", "Stopped")
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
		fmt.Printf("Successfully cancelled task '%s'\n", id)
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

		t := tui.NewTable("UUID", "Target Type", "Target Name", "Target Agent IP", "Store Type", "Store Name", "Taken at", "Expires at", "Status", "Notes")
		for _, archive := range archives {
			if *opts.Target != "" && archive.TargetUUID != *opts.Target {
				continue
			}
			if *opts.Store != "" && archive.StoreUUID != *opts.Store {
				continue
			}

			t.Row(archive, archive.UUID,
				archive.TargetPlugin, target[archive.TargetUUID].Name, target[archive.TargetUUID].Agent,
				archive.StorePlugin, store[archive.StoreUUID].Name,
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

		t := tui.NewReport()
		t.Add("UUID", archive.UUID)
		t.Add("Backup Key", archive.StoreKey)
		t.Break()

		t.Add("Target", archive.TargetPlugin)
		t.Add("Target UUID", archive.TargetUUID)
		t.Add("Target Endpoint", archive.TargetEndpoint)
		t.Break()

		t.Add("Store", archive.StorePlugin)
		t.Add("Store UUID", archive.StoreUUID)
		t.Add("Store Endpoint", archive.StoreEndpoint)
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
		fmt.Printf("Restoring archive '%s' %s\n", id, toTargetJSONmsg)
		return nil
	})

	c.Dispatch("delete archive", func(opts Options, args []string) error {
		DEBUG("running 'delete archive' command")

		require(len(args) == 1, "USAGE: shield delete archive <UUID>")
		id := uuid.Parse(args[0])
		DEBUG("  archive UUID = '%s'", id)

		err := DeleteArchive(id)
		if err != nil {
			return err
		}
		fmt.Printf("Deleted archive '%s'\n", id)
		return nil
	})
	c.Alias("remove archive", "delete archive")
	c.Alias("rm archive", "delete archive")

	/**************************************************************************/

	if err := c.Execute(command...); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}
