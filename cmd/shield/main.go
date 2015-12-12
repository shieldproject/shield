package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
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

func main() {
	options := Options{
		Shield:   getopt.StringLong("shield", 'H', "", "SHIELD target to run command against, i.e. http://shield.my.domain:8080"),
		Used:     getopt.BoolLong("used", 0, "Only show things that are in-use by something else"),
		Unused:   getopt.BoolLong("unused", 0, "Only show things that are not used by something else"),
		Paused:   getopt.BoolLong("paused", 0, "Only show jobs that are paused"),
		Unpaused: getopt.BoolLong("unpaused", 0, "Only show jobs that are unpaused"),
		All:      getopt.BoolLong("all", 'a', "Show all the things"),

		Debug: getopt.BoolLong("debug", 'D', "Enable debugging"),

		Status:    getopt.StringLong("status", 'S', "running", "Only show tasks with the given status (one of 'pending', 'running', 'canceled', 'done' or 'failed')"),
		Target:    getopt.StringLong("target", 't', "", "Only show things for the target with this UUID"),
		Store:     getopt.StringLong("store", 's', "", "Only show things for the store with this UUID"),
		Schedule:  getopt.StringLong("schedule", 'w', "", "Only show things for the schedule with this UUID"),
		Retention: getopt.StringLong("policy", 'p', "", "Only show things for the retention policy with this UUID"),
		Plugin:    getopt.StringLong("plugin", 'P', "", "Only show things for the given target or store plugin"),
		After:     getopt.StringLong("after", 'A', "", "Only show archives that were taken after the given date, in YYYYMMDD format."),
		Before:    getopt.StringLong("before", 'B', "", "Only show archives that were taken before the given date, in YYYYMMDD format."),
		To:        getopt.StringLong("to", 'T', "", "Restore the archive in question to a different target, specified by UUID"),
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

	if *options.Shield != "" {
		os.Setenv("SHIELD_TARGET", *options.Shield)
	}

	c := NewCommand().With(options)

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
		targets, err := GetTargets(TargetFilter{
			Plugin: *opts.Plugin,
			Unused: MaybeBools(*opts.Unused, *opts.Used),
		})

		if err != nil {
			return fmt.Errorf("failed to retrieve targets from SHIELD: %s", err)
		}

		t := tui.NewTable("UUID", "Target Name", "Description", "Plugin", "Endpoint", "SHIELD Agent")
		for _, target := range targets {
			t.Row(target.UUID, target.Name, target.Summary, target.Plugin, target.Endpoint, target.Agent)
		}
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("ls targets", "list targets")

	c.Dispatch("show target", func(opts Options, args []string) error {
		require(len(args) == 1, "shield show target <UUID>")
		id := uuid.Parse(args[0])

		target, err := GetTarget(id)
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
		userInput := bufio.NewReader(os.Stdin)
		in := tui.NewForm()

		fmt.Println("Target name")
		name, err := userInput.ReadString('\n')
		if err != nil {
			return fmt.Errorf("ERROR: Could not read input: %s", err)
		}
		u["name"] = strings.TrimSpace(name)

		fmt.Println("Target summary")
		summary, err := userInput.ReadString('\n')
		if err != nil {
			return fmt.Errorf("ERROR: Could not read input: %s", err)
		}
		u["summary"] = strings.TrimSpace(summary)

		fmt.Println("Target plugin")
		plugin, err := userInput.ReadString('\n')
		if err != nil {
			return fmt.Errorf("ERROR: Could not read input: %s", err)
		}
		u["plugin"] = strings.TrimSpace(plugin)

		fmt.Println("Target endpoint")
		endpoint, err := userInput.ReadString('\n')
		if err != nil {
			return fmt.Errorf("ERROR: Could not read input: %s", err)
		}
		u["endpoint"] = strings.TrimSpace(endpoint)

		fmt.Println("Target agent")
		agent, err := userInput.ReadString('\n')
		if err != nil {
			return fmt.Errorf("ERROR: Could not read input: %s", err)
		}
		u["agent"] = strings.TrimSpace(agent)

		content, err := json.Marshal(u)
		if err != nil {
			return fmt.Errorf("ERROR: Could not marshal into JSON\nmapped input:%v\nerror:%s", u, err)
		}
		t, err := CreateTarget(string(content))
		if err != nil {
			return fmt.Errorf("ERROR: Could not create new target: %s", err)
		}

		fmt.Printf("Created new target.\n")
		return c.Invoke("show", "target", t.UUID)
	})
	c.Alias("new target", "create target")
	c.Alias("create new target", "create target")
	c.Alias("make target", "create target")
	c.Alias("c t", "create target")

	c.Dispatch("edit target", func(opts Options, args []string) error {
		require(len(args) == 1, "shield edit target <UUID>")
		id := uuid.Parse(args[0])

		t, err := GetTarget(id)
		if err != nil {
			return fmt.Errorf("ERROR: Could not retrieve target '%s': %s", id, err)
		}
		t, err = UpdateTarget(id, invokeEditor(`{
  "name":     "`+t.Name+`",
  "summary":  "`+t.Summary+`",
  "plugin":   "`+t.Plugin+`",
  "endpoint": "`+t.Endpoint+`",
  "agent":    "`+t.Agent+`"
}`))
		if err != nil {
			return fmt.Errorf("ERROR: Could not update target '%s': %s", id, err)
		}
		fmt.Printf("Updated target.\n")
		return c.Invoke("show", "target", t.UUID)
	})
	c.Alias("update target", "edit target")

	c.Dispatch("delete target", func(opts Options, args []string) error {
		require(len(args) == 1, "shield delete target <UUID>")
		id := uuid.Parse(args[0])

		err := DeleteTarget(id)
		if err != nil {
			return fmt.Errorf("ERROR: Could not delete target '%s': %s", id, err)
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
		schedules, err := GetSchedules(ScheduleFilter{
			Unused: MaybeBools(*opts.Unused, *opts.Used),
		})
		if err != nil {
			return fmt.Errorf("failed to retrieve schedules from SHIELD: %s", err)
		}
		t := tui.NewTable("UUID", "Name", "Description", "Frequency / Interval (UTC)")
		for _, schedule := range schedules {
			t.Row(schedule.UUID, schedule.Name, schedule.Summary, schedule.When)
		}
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("ls schedules", "list schedules")

	c.Dispatch("show schedule", func(opts Options, args []string) error {
		require(len(args) == 1, "shield show schedule <UUID>")
		id := uuid.Parse(args[0])

		schedule, err := GetSchedule(id)
		if err != nil {
			return err
		}

		t := tui.NewReport()
		t.Add("UUID", schedule.UUID)
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
		userInput := bufio.NewReader(os.Stdin)
		u := make(map[string]interface{})

		fmt.Println("Schedule name")
		name, err := userInput.ReadString('\n')
		if err != nil {
			return fmt.Errorf("ERROR: Could not read input: %s", err)
		}
		u["name"] = strings.TrimSpace(name)

		fmt.Println("Schedule summary")
		summary, err := userInput.ReadString('\n')
		if err != nil {
			return fmt.Errorf("ERROR: Could not read input: %s", err)
		}
		u["summary"] = strings.TrimSpace(summary)

		fmt.Println("When to run schedule (eg daily at 4:00)")
		when, err := userInput.ReadString('\n')
		if err != nil {
			return fmt.Errorf("ERROR: Could not read input: %s", err)
		}
		u["when"] = strings.TrimSpace(when)

		content, err := json.Marshal(u)
		if err != nil {
			return fmt.Errorf("ERROR: Could not marshal into JSON\nmapped input:%v\nerror:%s", u, err)
		}
		s, err := CreateSchedule(string(content))

		if err != nil {
			return fmt.Errorf("ERROR: Could not create new schedule: %s", err)
		}
		fmt.Printf("Created new schedule.\n")
		return c.Invoke("show", "schedule", s.UUID)
	})
	c.Alias("new schedule", "create schedule")
	c.Alias("create new schedule", "create schedule")
	c.Alias("make schedule", "create schedule")
	c.Alias("c s", "create schedule")

	c.Dispatch("edit schedule", func(opts Options, args []string) error {
		require(len(args) == 1, "shield edit schedule <UUID>")
		id := uuid.Parse(args[0])

		s, err := GetSchedule(id)
		if err != nil {
			return fmt.Errorf("ERROR: Could not retrieve schedule '%s': %s", id, err)
		}

		s, err = UpdateSchedule(id, invokeEditor(`{
  "name":    "`+s.Name+`",
  "summary": "`+s.Summary+`",
  "when":    "`+s.When+`"
}`))
		if err != nil {
			return fmt.Errorf("ERROR: Could not update schedule '%s': %s", id, err)
		}
		fmt.Printf("Updated schedule.\n")
		return c.Invoke("show", "schedule", s.UUID)
	})
	c.Alias("update schedule", "edit schedule")

	c.Dispatch("delete schedule", func(opts Options, args []string) error {
		require(len(args) == 1, "shield delete schedule <UUID>")
		id := uuid.Parse(args[0])

		err := DeleteSchedule(id)
		if err != nil {
			return fmt.Errorf("ERROR: Cannot delete schedule '%s': %s", id, err)
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
		policies, err := GetRetentionPolicies(RetentionPoliciesFilter{
			Unused: MaybeBools(*opts.Unused, *opts.Used),
		})
		if err != nil {
			return fmt.Errorf("failed to retrieve retention policies from SHIELD: %s", err)
		}
		t := tui.NewTable("UUID", "Name", "Description", "Expires in")
		for _, policy := range policies {
			t.Row(policy.UUID, policy.Name, policy.Summary, fmt.Sprintf("%d days", policy.Expires/86400))
		}
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("ls retention policies", "list retention policies")
	c.Alias("list policies", "list retention policies")
	c.Alias("ls policies", "list policies")

	c.Dispatch("show retention policy", func(opts Options, args []string) error {
		require(len(args) == 1, "shield show retention policy <UUID>")
		id := uuid.Parse(args[0])

		policy, err := GetRetentionPolicy(id)
		if err != nil {
			return err
		}

		t := tui.NewReport()
		t.Add("UUID", policy.UUID)
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
		userInput := bufio.NewReader(os.Stdin)
		u := make(map[string]interface{})

		fmt.Println("Policy name")
		name, err := userInput.ReadString('\n')
		if err != nil {
			return fmt.Errorf("ERROR: Could not read input: %s", err)
		}
		u["name"] = strings.TrimSpace(name)

		fmt.Println("Policy summary")
		summary, err := userInput.ReadString('\n')
		if err != nil {
			return fmt.Errorf("ERROR: Could not read input: %s", err)
		}
		u["summary"] = strings.TrimSpace(summary)

		fmt.Println("Policy expiration in seconds (protip: there are 86400 sec per day)")
		expire_string, err := userInput.ReadString('\n')
		if err != nil {
			return fmt.Errorf("ERROR: Could not read input: %s", err)
		}
		expire_string = strings.TrimSpace(expire_string)
		expires, err := strconv.Atoi(expire_string)
		if err != nil {
			return fmt.Errorf("ERROR: Could not convert input '%s' to integer.", expire_string)
		}
		u["expires"] = expires

		content, err := json.Marshal(u)
		if err != nil {
			return fmt.Errorf("ERROR: Could not marshal into JSON\nmapped input:%v\nerror:%s", u, err)
		}
		p, err := CreateRetentionPolicy(string(content))

		if err != nil {
			return fmt.Errorf("ERROR: Could not create new retention policy: %s", err)
		}
		fmt.Printf("Created new retention policy.\n")
		return c.Invoke("show", "retention", "policy", p.UUID)
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
		require(len(args) == 1, "shield edit retention policy <UUID>")
		id := uuid.Parse(args[0])

		p, err := GetRetentionPolicy(id)
		if err != nil {
			return fmt.Errorf("ERROR: Cannot retrieve policy '%s': %s", id, err)
		}
		p, err = UpdateRetentionPolicy(id, invokeEditor(`{
  "name":     "`+p.Name+`",
  "summary":  "`+p.Summary+`",
  "expires":  `+fmt.Sprintf("%d", p.Expires)+`
}`))
		if err != nil {
			return fmt.Errorf("ERROR: Cannot update policy '%s': %s", id, err)
		}
		fmt.Printf("Updated policy.\n")
		return c.Invoke("show", "retention", "policy", p.UUID)
	})
	c.Alias("update retention policy", "edit retention policy")
	c.Alias("edit policy", "edit retention policy")
	c.Alias("update policy", "edit policy")

	c.Dispatch("delete retention policy", func(opts Options, args []string) error {
		require(len(args) == 1, "shield delete retention policy <UUID>")
		id := uuid.Parse(args[0])

		err := DeleteRetentionPolicy(id)
		if err != nil {
			return fmt.Errorf("ERROR: Cannot delete retention policy '%s': %s", id, err)
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
		stores, err := GetStores(StoreFilter{
			Plugin: *opts.Plugin,
			Unused: MaybeBools(*opts.Unused, *opts.Used),
		})
		if err != nil {
			return fmt.Errorf("ERROR: Could not fetch list of stores: %s", err)
		}
		t := tui.NewTable("UUID", "Name", "Description", "Plugin", "Endpoint")
		for _, store := range stores {
			t.Row(store.UUID, store.Name, store.Summary, store.Plugin, store.Endpoint)
		}
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("ls stores", "list stores")

	c.Dispatch("show store", func(opts Options, args []string) error {
		require(len(args) == 1, "shield show store <UUID>")
		id := uuid.Parse(args[0])

		store, err := GetStore(id)
		if err != nil {
			return err
		}

		t := tui.NewReport()
		t.Add("UUID", store.UUID)
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
		userInput := bufio.NewReader(os.Stdin)
		u := make(map[string]interface{})

		fmt.Println("Store name")
		name, err := userInput.ReadString('\n')
		if err != nil {
			return fmt.Errorf("ERROR: Could not read input: %s", err)
		}
		u["name"] = strings.TrimSpace(name)

		fmt.Println("Store summary")
		summary, err := userInput.ReadString('\n')
		if err != nil {
			return fmt.Errorf("ERROR: Could not read input: %s", err)
		}
		u["summary"] = strings.TrimSpace(summary)

		fmt.Println("Plugin name")
		plugin, err := userInput.ReadString('\n')
		if err != nil {
			return fmt.Errorf("ERROR: Could not read input: %s", err)
		}
		u["plugin"] = strings.TrimSpace(plugin)

		fmt.Println("Endpoint JSON")
		endpoint, err := userInput.ReadString('\n')
		if err != nil {
			return fmt.Errorf("ERROR: Could not read input: %s", err)
		}
		u["endpoint"] = strings.TrimSpace(endpoint)

		content, err := json.Marshal(u)
		if err != nil {
			return fmt.Errorf("ERROR: Could not marshal into JSON\nmapped input:%v\nerror:%s", u, err)
		}
		s, err := CreateStore(string(content))

		if err != nil {
			return fmt.Errorf("ERROR: Could not create new store: %s", err)
		}
		fmt.Printf("Created new store.\n")

		return c.Invoke("show", "store", s.UUID)
	})
	c.Alias("new store", "create store")
	c.Alias("create new store", "create store")
	c.Alias("make store", "create store")
	c.Alias("c st", "create store")

	c.Dispatch("edit store", func(opts Options, args []string) error {
		require(len(args) == 1, "shield delete store <UUID>")
		id := uuid.Parse(args[0])

		s, err := GetStore(id)
		if err != nil {
			return fmt.Errorf("ERROR: Cannot retrieve store '%s': %s", id, err)
		}

		s, err = UpdateStore(id, invokeEditor(`{
  "name":     "`+s.Name+`",
  "summary":  "`+s.Summary+`",
  "plugin":   "`+s.Plugin+`",
  "endpoint": "`+s.Endpoint+`"
}`))
		if err != nil {
			return fmt.Errorf("ERROR: Cannot update store '%s': %s", id, err)
		}

		fmt.Printf("Updated store.\n")
		return c.Invoke("show", "store", s.UUID)
	})
	c.Alias("update store", "edit store")

	c.Dispatch("delete store", func(opts Options, args []string) error {
		require(len(args) == 1, "shield delete store <UUID>")
		id := uuid.Parse(args[0])

		err := DeleteStore(id)
		if err != nil {
			return fmt.Errorf("ERROR: Cannot delete store '%s': %s", id, err)
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
		jobs, err := GetJobs(JobFilter{
			Paused:    MaybeBools(*opts.Unpaused, *opts.Paused),
			Target:    *opts.Target,
			Store:     *opts.Store,
			Schedule:  *opts.Schedule,
			Retention: *opts.Retention,
		})
		if err != nil {
			return fmt.Errorf("\nERROR: Unexpected arguments following command: %v\n", err)
		}
		t := tui.NewTable("UUID", "P?", "Name", "Description", "Retention Policy", "Schedule", "Target", "Agent")
		for _, job := range jobs {
			t.Row(job.UUID, BoolString(job.Paused), job.Name, job.Summary,
				job.RetentionName, job.ScheduleName, job.TargetEndpoint, job.Agent)
		}
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("ls jobs", "list jobs")
	c.Alias("ls j", "list jobs")

	c.Dispatch("show job", func(opts Options, args []string) error {
		require(len(args) == 1, "shield show job <UUID>")
		id := uuid.Parse(args[0])

		job, err := GetJob(id)
		if err != nil {
			return nil
		}

		t := tui.NewReport()
		t.Add("UUID", job.UUID)
		t.Add("Name", job.Name)
		t.Add("Paused", BoolString(job.Paused))
		t.Break()

		t.Add("Retention Policy", job.RetentionName)
		t.Add("Retention UUID", job.RetentionUUID)
		t.Add("Expires in", fmt.Sprintf("%d days", job.Expiry/86400))
		t.Break()

		t.Add("Schedule Policy", job.ScheduleName)
		t.Add("Schedule UUID", job.ScheduleUUID)
		t.Break()

		t.Add("Target", job.TargetPlugin)
		t.Add("Target UUID", job.TargetUUID)
		t.Add("Target Endpoint", job.TargetEndpoint)
		t.Add("SHIELD Agent", job.Agent)
		t.Break()

		t.Add("Store", job.StorePlugin)
		t.Add("Store UUID", job.StoreUUID)
		t.Add("Store Endpoint", job.StoreEndpoint)
		t.Break()

		t.Add("Store", job.StorePlugin)
		t.Add("Store UUID", job.StoreUUID)
		t.Add("Store Endpoint", job.StoreEndpoint)

		t.Add("Notes", job.Summary)

		t.Output(os.Stdout)
		return nil
	})
	c.Alias("view job", "show job")
	c.Alias("display job", "show job")
	c.Alias("list job", "show job")
	c.Alias("ls job", "show job")

	c.Dispatch("create job", func(opts Options, args []string) error {
		userInput := bufio.NewReader(os.Stdin)
		u := make(map[string]interface{})

		fmt.Println("Job name")
		name, err := userInput.ReadString('\n')
		if err != nil {
			return fmt.Errorf("ERROR: Could not read input: %s", err)
		}
		u["name"] = strings.TrimSpace(name)

		fmt.Println("Job summary")
		summary, err := userInput.ReadString('\n')
		if err != nil {
			return fmt.Errorf("ERROR: Could not read input: %s", err)
		}
		u["summary"] = strings.TrimSpace(summary)

		fmt.Println("Store UUID")
		store, err := userInput.ReadString('\n')
		if err != nil {
			return fmt.Errorf("ERROR: Could not read input: %s", err)
		}
		u["store"] = strings.TrimSpace(store)

		fmt.Println("Target UUID")
		target, err := userInput.ReadString('\n')
		if err != nil {
			return fmt.Errorf("ERROR: Could not read input: %s", err)
		}
		u["target"] = strings.TrimSpace(target)

		fmt.Println("Policy UUID")
		policy, err := userInput.ReadString('\n')
		if err != nil {
			return fmt.Errorf("ERROR: Could not read input: %s", err)
		}
		u["retention"] = strings.TrimSpace(policy)

		fmt.Println("Schedule UUID")
		schedule, err := userInput.ReadString('\n')
		if err != nil {
			return fmt.Errorf("ERROR: Could not read input: %s", err)
		}
		u["schedule"] = strings.TrimSpace(schedule)

		fmt.Println("Should the job be paused on creation? (Y/n)")
		paused, err := userInput.ReadString('\n')
		if err != nil {
			return fmt.Errorf("ERROR: Could not read input: %s", err)
		}
		paused = strings.TrimSpace(paused)
		pause := false
		if paused == "Y" || paused == "yes" || paused == "y" {
			pause = true
		}
		u["paused"] = pause

		content, err := json.Marshal(u)
		if err != nil {
			return fmt.Errorf("ERROR: Could not marshal into JSON\nmapped input:%v\nerror:%s", u, err)
		}

		job, err := CreateJob(string(content))
		if err != nil {
			return fmt.Errorf("ERROR: Could not create new job: %s", err)
		}

		return c.Invoke("show", "job", job.UUID)
	})
	c.Alias("new job", "create job")
	c.Alias("create new job", "create job")
	c.Alias("make job", "create job")
	c.Alias("c j", "create job")

	c.Dispatch("edit job", func(opts Options, args []string) error {
		require(len(args) == 1, "shield edit job <UUID>")
		id := uuid.Parse(args[0])

		j, err := GetJob(id)
		if err != nil {
			return fmt.Errorf("ERROR: Could not retrieve job '%s': %s", id, err)
		}
		paused := "false"
		if j.Paused {
			paused = "true"
		}

		content := invokeEditor(`{
  "name"      : "` + j.Name + `",
  "summary"   : "` + j.Summary + `",

  "store"     : "` + j.StoreUUID + `",
  "target"    : "` + j.TargetUUID + `",
  "retention" : "` + j.RetentionUUID + `",
  "schedule"  : "` + j.ScheduleUUID + `",
  "paused"    : ` + paused + `
}`)

		j, err = UpdateJob(id, content)
		if err != nil {
			return fmt.Errorf("ERROR: Could not update job '%s': %s", id, err)
		}

		fmt.Printf("Updated job.\n")
		return c.Invoke("show", "job", j.UUID)
	})
	c.Alias("update job", "edit job")

	c.Dispatch("delete job", func(opts Options, args []string) error {
		require(len(args) == 1, "shield delete job <UUID>")
		id := uuid.Parse(args[0])

		err := DeleteJob(id)
		if err != nil {
			return fmt.Errorf("ERROR: Cannot delete job '%s': %s", id, err)
		}
		fmt.Printf("Deleted job '%s'\n", id)
		return nil
	})
	c.Alias("remove job", "delete job")
	c.Alias("rm job", "delete job")

	c.Dispatch("pause job", func(opts Options, args []string) error {
		require(len(args) == 1, "shield pause job <UUID>")
		id := uuid.Parse(args[0])

		err := PauseJob(id)
		if err != nil {
			return fmt.Errorf("ERROR: Could not pause job '%s': %s", id, err)
		}
		fmt.Printf("Successfully paused job '%s'\n", id)
		return nil
	})

	c.Dispatch("unpause job", func(opts Options, args []string) error {
		require(len(args) == 1, "shield unpause job <UUID>")
		id := uuid.Parse(args[0])

		err := UnpauseJob(id)
		if err != nil {
			return fmt.Errorf("ERROR: Could not unpause job '%s': %s", id, err)
		}
		fmt.Printf("Successfully unpaused job '%s'\n", id)
		return nil
	})

	c.Dispatch("run job", func(opts Options, args []string) error {
		require(len(args) == 1, "shield run job <UUID>")
		id := uuid.Parse(args[0])

		err := RunJob(id, fmt.Sprintf(`{"owner":"%s@%s"}`, os.Getenv("USER"), os.Getenv("HOSTNAME")))
		if err != nil {
			return err
		}
		fmt.Printf("scheduled immediate run of job %s\n", id)
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
		if *options.Status == "all" || *options.All {
			*options.Status = ""
		}
		tasks, err := GetTasks(TaskFilter{
			Status: *options.Status,
		})
		if err != nil {
			return fmt.Errorf("\nERROR: Could not fetch list of tasks: %s\n", err)
		}
		t := tui.NewTable("UUID", "Owner", "Type", "Status", "Started", "Stopped")
		for _, task := range tasks {
			started := "(pending)"
			if !task.StartedAt.IsZero() {
				started = task.StartedAt.Format(time.RFC1123Z)
			}

			stopped := "(running)"
			if !task.StoppedAt.IsZero() {
				stopped = task.StoppedAt.Format(time.RFC1123Z)
			}

			t.Row(task.UUID, task.Owner, task.Op, task.Status, started, stopped)
		}
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("ls tasks", "list tasks")

	c.Dispatch("show task", func(opts Options, args []string) error {
		require(len(args) == 1, "shield show task <UUID>")
		id := uuid.Parse(args[0])

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
		require(len(args) == 1, "shield cancel task <UUID>")
		id := uuid.Parse(args[0])

		err := CancelTask(id)
		if err != nil {
			return fmt.Errorf("ERROR: could not cancel task '%s'", id)
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
		archives, err := GetArchives(ArchiveFilter{
			Target: *options.Target,
			Store:  *options.Store,
			Before: *options.Before,
			After:  *options.After,
		})
		if err != nil {
			return fmt.Errorf("ERROR: Could not fetch list of archives: %s", err)
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

		t := tui.NewTable("UUID", "Target Type", "Target Name", "Store Type", "Store Name", "Taken at", "Expires at", "Notes")
		for _, archive := range archives {
			if *opts.Target != "" && archive.TargetUUID != *opts.Target {
				continue
			}
			if *opts.Store != "" && archive.StoreUUID != *opts.Store {
				continue
			}

			t.Row(archive.UUID,
				archive.TargetPlugin, target[archive.TargetUUID].Name,
				archive.StorePlugin, store[archive.StoreUUID].Name,
				archive.TakenAt.Format(time.RFC1123Z),
				archive.ExpiresAt.Format(time.RFC1123Z),
				archive.Notes)
		}
		t.Output(os.Stdout)
		return nil
	})
	c.Alias("ls archives", "list archives")

	c.Dispatch("show archive", func(opts Options, args []string) error {
		require(len(args) == 1, "shield show archive <UUID>")
		id := uuid.Parse(args[0])

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
		require(len(args) == 1, "USAGE: shield restore archive <UUID>")
		id := uuid.Parse(args[0])

		targetJSON := "{}"
		toTargetJSONmsg := ""
		if opts.Target != nil {
			targetJSON = *opts.Target
			toTargetJSONmsg = fmt.Sprintf("to target '%s'", targetJSON)
		}

		err := RestoreArchive(id, targetJSON)
		if err != nil {
			return fmt.Errorf("ERROR: Cannot restore archive '%s': '%s'", id, err)
		}
		fmt.Printf("Restoring archive '%s' %s\n", id, toTargetJSONmsg)
		return nil
	})

	c.Dispatch("delete archive", func(opts Options, args []string) error {
		require(len(args) == 1, "USAGE: shield delete archive <UUID>")
		id := uuid.Parse(args[0])

		err := DeleteArchive(id)
		if err != nil {
			return fmt.Errorf("ERROR: Cannot delete archive '%s': %s", id, err)
		}
		fmt.Printf("Deleted archive '%s'\n", id)
		return nil
	})
	c.Alias("remove archive", "delete archive")
	c.Alias("rm archive", "delete archive")

	/**************************************************************************/

	if err, ok := c.Execute(command...); ok {
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
}

func invokeEditor(content string) string {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	tmpDir := os.TempDir()
	tmpFile, tmpFileErr := ioutil.TempFile(tmpDir, "tempFilePrefix")
	if tmpFileErr != nil {
		fmt.Fprintln(os.Stderr, "ERROR: Could not create temporary editor file:\n", tmpFileErr)
	}
	if content != "" {
		err := ioutil.WriteFile(tmpFile.Name(), []byte(content), 600)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERROR: Could not write initial content to editor file:\n", err)
			os.Exit(1)
		}
	}

	path, err := exec.LookPath(editor)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Could not find editor `%s` in path:\n%s", editor, err)
		os.Exit(1)
	}
	fmt.Printf("%s is available at %s\nCalling it with file %s \n", editor, path, tmpFile.Name())

	cmd := exec.Command(path, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		fmt.Printf("Start failed: %s", err)
	}
	fmt.Printf("Waiting for editor to finish.\n")
	err = cmd.Wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Editor `%s` exited with error:\n%s", editor, err)
		os.Exit(1)
	}

	new_content, err := ioutil.ReadFile(tmpFile.Name())

	return string(new_content)
}
