package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pborman/getopt"
	"github.com/pborman/uuid"
	"github.com/starkandwayne/goutils/ansi"

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

var Version = ""

func main() {
	options := Options{
		Shield:   getopt.StringLong("shield", 'H', "", "DEPRECATED - Previously required to point to a SHIELD backend to talk to. Now used to auto-vivify ~/.shield_config if necessary"),
		Used:     getopt.BoolLong("used", 0, "Only show things that are in-use by something else"),
		Unused:   getopt.BoolLong("unused", 0, "Only show things that are not used by something else"),
		Paused:   getopt.BoolLong("paused", 0, "Only show jobs that are paused"),
		Unpaused: getopt.BoolLong("unpaused", 0, "Only show jobs that are unpaused"),
		All:      getopt.BoolLong("all", 'a', "Show all the things"),

		Debug:             getopt.BoolLong("debug", 'D', "Enable debugging"),
		Trace:             getopt.BoolLong("trace", 'T', "Enable trace mode"),
		Raw:               getopt.BoolLong("raw", 0, "Operate in RAW mode, reading and writing only JSON"),
		Fuzzy:             getopt.BoolLong("fuzzy", 0, "In RAW mode, perform fuzzy (inexact) searching"),
		SkipSSLValidation: getopt.BoolLong("skip-ssl-validation", 'k', "Disable SSL Certificate Validation"),

		Status:    getopt.StringLong("status", 'S', "", "Only show archives/tasks with the given status"),
		Target:    getopt.StringLong("target", 't', "", "Only show things for the target with this UUID"),
		Store:     getopt.StringLong("store", 's', "", "Only show things for the store with this UUID"),
		Schedule:  getopt.StringLong("schedule", 'w', "", "Only show things for the schedule with this UUID"),
		Retention: getopt.StringLong("policy", 'p', "", "Only show things for the retention policy with this UUID"),
		Plugin:    getopt.StringLong("plugin", 'P', "", "Only show things for the given target or store plugin"),
		After:     getopt.StringLong("after", 'A', "", "Only show archives that were taken after the given date, in YYYYMMDD format."),
		Before:    getopt.StringLong("before", 'B', "", "Only show archives that were taken before the given date, in YYYYMMDD format."),
		To:        getopt.StringLong("to", 0, "", "Restore the archive in question to a different target, specified by UUID"),
		Limit:     getopt.StringLong("limit", 0, "", "Display only the X most recent tasks or archives"),

		Config:  getopt.StringLong("config", 'c', os.Getenv("HOME")+"/.shield_config", "Overrides ~/.shield_config as the SHIELD config file"),
		Version: getopt.BoolLong("version", 'v', "Display the SHIELD version"),
	}

	OK := func(f string, l ...interface{}) {
		if *options.Raw {
			RawJSON(map[string]string{"ok": fmt.Sprintf(f, l...)})
			return
		}
		ansi.Printf("@G{%s}\n", fmt.Sprintf(f, l...))
	}

	MSG := func(f string, l ...interface{}) {
		if !*options.Raw {
			ansi.Printf("\n@G{%s}\n", fmt.Sprintf(f, l...))
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

	if len(command) == 0 {
		command = []string{"help"}
	}

	debug = *options.Debug
	DEBUG("shield cli starting up")

	if *options.Trace {
		DEBUG("enabling TRACE output")
		os.Setenv("SHIELD_TRACE", "1")
	}

	if *options.SkipSSLValidation {
		os.Setenv("SHIELD_SKIP_SSL_VERIFY", "true")
	}

	if *options.Version {
		if Version == "" {
			fmt.Printf("shield cli (development)%s\n", Version)
		} else {
			fmt.Printf("shield cli v%s\n", Version)
		}
		os.Exit(0)
	}

	c := NewCommand().With(options)

	c.HelpGroup("INFO:")
	c.Dispatch("help", "Get detailed help with a specific command",
		func(opts Options, args []string, help bool) error {
			if len(args) == 0 {
				buf := bytes.Buffer{}
				getopt.PrintUsage(&buf)
				//Gets the usage line from the getopt usage output
				ansi.Fprintf(os.Stderr, strings.Split(buf.String(), "\n")[0]+"\n")
				ansi.Fprintf(os.Stderr, "For more help with a command, type @M{shield help <command>}\n")
				ansi.Fprintf(os.Stderr, "For a list of available commands, type @M{shield commands}\n")
				ansi.Fprintf(os.Stderr, "For a list of available flags, type @M{shield flags}\n")
				ansi.Fprintf(os.Stderr, "\n@R{The verbose, multi-word commands (such as `list schedules`) are now deprecated}\n"+
					"@R{in favor of, for example, the shorter `schedules`. Other long commands have had their}\n"+
					"@R{spaces replaced with dashes. Check `commands` for the new canonical names.}\n")
				return nil
			} else if args[0] == "help" {
				ansi.Fprintf(os.Stderr, "@R{This is getting a bit too meta, don't you think?}\n")
				return nil
			}

			// otherwise ...
			return c.Help(args...)
		})

	c.Alias("usage", "help")

	c.Dispatch("commands", "Show the list of available commands",
		func(opts Options, args []string, help bool) error {
			ansi.Fprintf(os.Stderr, "\n@R{NAME:}\n  shield\t\tCLI for interacting with the Shield API.\n")
			ansi.Fprintf(os.Stderr, "\n@R{USAGE:}\n  shield [options] <command>\n")
			ansi.Fprintf(os.Stderr, "\n@R{ENVIRONMENT VARIABLES:}\n")
			ansi.Fprintf(os.Stderr, "  SHIELD_TRACE\t\tset to 'true' for trace output.\n")
			ansi.Fprintf(os.Stderr, "  SHIELD_DEBUG\t\tset to 'true' for debug output.\n\n")
			ansi.Fprintf(os.Stderr, "@R{COMMANDS:}\n\n")
			ansi.Fprintf(os.Stderr, c.Usage())
			ansi.Fprintf(os.Stderr, "\n")
			return nil
		})

	c.Dispatch("flags", "Show the list of all command line flags",
		func(opts Options, args []string, help bool) error {
			getopt.PrintUsage(os.Stderr)
			return nil
		})
	c.Alias("options", "flags")

	/*
	    ######  ########    ###    ######## ##     ##  ######
	   ##    ##    ##      ## ##      ##    ##     ## ##    ##
	   ##          ##     ##   ##     ##    ##     ## ##
	    ######     ##    ##     ##    ##    ##     ##  ######
	         ##    ##    #########    ##    ##     ##       ##
	   ##    ##    ##    ##     ##    ##    ##     ## ##    ##
	    ######     ##    ##     ##    ##     #######   ######
	*/
	c.Dispatch("status", "Query the SHIELD backup server for its status and version info",
		func(opts Options, args []string, help bool) error {
			if help {
				FlagHelp("Outputs information as a JSON object", true, "--raw")
				JSONHelp(fmt.Sprintf("{\"name\":\"MyShield\",\"version\":\"%s\"}\n", Version))
				return nil
			}

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
			t.Add("API Version", status.Version)
			t.Output(os.Stdout)
			return nil
		})
	c.Alias("stat", "status")

	/*
	   ########     ###     ######  ##    ## ######## ##    ## ########   ######
	   ##     ##   ## ##   ##    ## ##   ##  ##       ###   ## ##     ## ##    ##
	   ##     ##  ##   ##  ##       ##  ##   ##       ####  ## ##     ## ##
	   ########  ##     ## ##       #####    ######   ## ## ## ##     ##  ######
	   ##     ## ######### ##       ##  ##   ##       ##  #### ##     ##       ##
	   ##     ## ##     ## ##    ## ##   ##  ##       ##   ### ##     ## ##    ##
	   ########  ##     ##  ######  ##    ## ######## ##    ## ########   ######
	*/
	c.HelpBreak()
	c.HelpGroup("BACKENDS:")
	c.Dispatch("backends", "List configured SHIELD backends",
		func(opts Options, args []string, help bool) error {
			if help {
				JSONHelp(`[{"name":"mybackend","uri":"https://10.244.2.2:443"}]`)
				FlagHelp("Outputs information as JSON object", true, "--raw")
				return nil
			}

			DEBUG("running 'backends' command")

			var indices []string
			for k, _ := range Cfg.Aliases {
				indices = append(indices, k)
			}
			sort.Strings(indices)

			if *opts.Raw {
				arr := []map[string]string{}
				for _, alias := range indices {
					arr = append(arr, map[string]string{"name": alias, "uri": Cfg.Aliases[alias]})
				}
				return RawJSON(arr)
			}

			t := tui.NewTable("Name", "Backend URI")
			for _, alias := range indices {
				be := map[string]string{"name": alias, "uri": Cfg.Aliases[alias]}
				t.Row(be, be["name"], be["uri"])
			}
			t.Output(os.Stdout)

			return nil
		})
	c.Alias("list backends", "backends")
	c.Alias("ls be", "backends")

	c.Dispatch("create-backend", "Create or modify a SHIELD backend",
		func(opts Options, args []string, help bool) error {
			if help {
				FlagHelp(`The name of the new backend`, false, "<name>")
				FlagHelp(`The address at which the new backend can be found`, false, "<uri>")

				return nil
			}

			DEBUG("running 'create backend' command")

			if len(args) != 2 {
				return fmt.Errorf("Invalid 'create backend' syntax: `shield backend <name> <uri>")
			}
			err := Cfg.AddBackend(args[1], args[0])
			if err != nil {
				return err
			}

			err = Cfg.UseBackend(args[0])
			if err != nil {
				return err
			}

			err = Cfg.Save()
			if err != nil {
				return err
			}

			ansi.Fprintf(os.Stdout, "Successfully created backend '@G{%s}', pointing to '@G{%s}'\n\n", args[0], args[1])
			DisplayBackend(Cfg)

			return nil
		})
	c.Alias("create backend", "create-backend")
	c.Alias("c be", "create-backend")
	c.Alias("update backend", "create-backend")
	c.Alias("update-backend", "create-backend")
	c.Alias("edit-backend", "create-backend")
	c.Alias("edit backend", "create-backend")

	c.Dispatch("backend", "Select a particular backend for use",
		func(opts Options, args []string, help bool) error {
			if help {
				FlagHelp(`The name of the backend to target`, false, "<name>")
				return nil
			}

			DEBUG("running 'backend' command")

			if len(args) == 0 {
				DisplayBackend(Cfg)
				return nil
			}

			if len(args) != 1 {
				return fmt.Errorf("Invalid 'backend' syntax: `shield backend <name>`")
			}
			err := Cfg.UseBackend(args[0])
			if err != nil {
				return err
			}
			Cfg.Save()

			DisplayBackend(Cfg)
			return nil
		})
	c.Alias("use backend", "backend")
	c.Alias("use-backend", "backend")

	/*
	   ########    ###    ########   ######   ######## ########
	      ##      ## ##   ##     ## ##    ##  ##          ##
	      ##     ##   ##  ##     ## ##        ##          ##
	      ##    ##     ## ########  ##   #### ######      ##
	      ##    ######### ##   ##   ##    ##  ##          ##
	      ##    ##     ## ##    ##  ##    ##  ##          ##
	      ##    ##     ## ##     ##  ######   ########    ##
	*/

	c.HelpBreak()
	c.HelpGroup("TARGETS:")
	c.Dispatch("targets", "List available backup targets",
		func(opts Options, args []string, help bool) error {
			if help {
				FlagHelp("Only show targets using the named target plugin", true, "-P", "--policy=value")
				HelpListMacro("target", "targets")
				JSONHelp(`[{"uuid":"8add3e57-95cd-4ec0-9144-4cd5c50cd392","name":"SampleTarget","summary":"A Sample Target","plugin":"postgres","endpoint":"{\"endpoint\":\"127.0.0.1:5432\"}","agent":"127.0.0.1:1234"}]`)
				return nil
			}

			DEBUG("running 'list targets' command")
			DEBUG("  for plugin: '%s'", *opts.Plugin)
			DEBUG("  show unused? %v", *opts.Unused)
			DEBUG("  show in-use? %v", *opts.Used)
			if *opts.Raw {
				DEBUG(" fuzzy search? %v", MaybeBools(*opts.Fuzzy, *opts.Raw).Yes)
			}

			targets, err := GetTargets(TargetFilter{
				Name:       strings.Join(args, " "),
				Plugin:     *opts.Plugin,
				Unused:     MaybeBools(*opts.Unused, *opts.Used),
				ExactMatch: Opposite(MaybeBools(*opts.Fuzzy, *opts.Raw)),
			})

			if err != nil {
				return err
			}

			if *opts.Raw {
				return RawJSON(targets)
			}

			t := tui.NewTable("Name", "Summary", "Plugin", "Remote IP", "Configuration")
			for _, target := range targets {
				t.Row(target, target.Name, target.Summary, target.Plugin, target.Agent, PrettyJSON(target.Endpoint))
			}
			t.Output(os.Stdout)
			return nil
		})
	c.Alias("list targets", "targets")
	c.Alias("ls targets", "targets")

	c.Dispatch("target", "Print detailed information about a specific backup target",
		func(opts Options, args []string, help bool) error {
			if help {
				JSONHelp(`{"uuid":"8add3e57-95cd-4ec0-9144-4cd5c50cd392","name":"SampleTarget","summary":"A Sample Target","plugin":"postgres","endpoint":"{\"endpoint\":\"127.0.0.1:5432\"}","agent":"127.0.0.1:1234"}`)
				HelpShowMacro("target", "targets")
				return nil
			}

			DEBUG("running 'show target' command")

			target, _, err := FindTarget(strings.Join(args, " "), *opts.Raw)
			if err != nil {
				return err
			}

			if *opts.Raw {
				return RawJSON(target)
			}

			ShowTarget(target)
			return nil
		})
	c.Alias("show target", "target")
	c.Alias("view target", "target")
	c.Alias("display target", "target")
	c.Alias("list target", "target")
	c.Alias("ls target", "target")

	c.Dispatch("create-target", "Create a new backup target",
		func(opts Options, args []string, help bool) error {
			if help {
				InputHelp(`{"agent":"127.0.0.1:1234","endpoint":"{\"endpoint\":\"schmendpoint\"}","name":"TestTarget","plugin":"postgres","summary":"A Test Target"}`)
				JSONHelp(`{"uuid":"77398f3e-2a31-4f20-b3f7-49d3f0998712","name":"TestTarget","summary":"A Test Target","plugin":"postgres","endpoint":"{\"endpoint\":\"schmendpoint\"}","agent":"127.0.0.1:1234"}`)
				HelpCreateMacro("target", "targets")
				return nil
			}

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
				in.NewField("Plugin Name", "plugin", "", "", FieldIsPluginName)
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
			return c.Execute("target", t.UUID)
		})
	c.Alias("create target", "create-target")
	c.Alias("new target", "create-target")
	c.Alias("create new target", "create-target")
	c.Alias("make target", "create-target")
	c.Alias("c t", "create-target")
	c.Alias("add target", "create-target")

	c.Dispatch("edit-target", "Modify an existing backup target",
		func(opts Options, args []string, help bool) error {
			if help {
				InputHelp(`{"agent":"127.0.0.1:1234","endpoint":"{\"endpoint\":\"newschmendpoint\"}","name":"NewTargetName","plugin":"postgres","summary":"Some Target"}`)
				JSONHelp(`{"uuid":"8add3e57-95cd-4ec0-9144-4cd5c50cd392","name":"SomeTarget","summary":"Just this target, you know?","plugin":"postgres","endpoint":"{\"endpoint\":\"schmendpoint\"}","agent":"127.0.0.1:1234"}`)
				HelpEditMacro("target", "targets")
				return nil
			}

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
				in.NewField("Plugin Name", "plugin", t.Plugin, "", FieldIsPluginName)
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
			return c.Execute("target", t.UUID)
		})
	c.Alias("edit target", "edit-target")
	c.Alias("update target", "edit-target")

	c.Dispatch("delete-target", "Delete a backup target",
		func(opts Options, args []string, help bool) error {
			if help {
				HelpDeleteMacro("target", "targets")
				return nil
			}

			DEBUG("running 'delete target' command")

			target, id, err := FindTarget(strings.Join(args, " "), *opts.Raw)
			if err != nil {
				return err
			}

			if !*opts.Raw {
				ShowTarget(target)
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
	c.Alias("delete target", "delete-target")
	c.Alias("remove target", "delete-target")
	c.Alias("rm target", "delete-target")

	/*
	    ######   ######  ##     ## ######## ########  ##     ## ##       ########
	   ##    ## ##    ## ##     ## ##       ##     ## ##     ## ##       ##
	   ##       ##       ##     ## ##       ##     ## ##     ## ##       ##
	    ######  ##       ######### ######   ##     ## ##     ## ##       ######
	         ## ##       ##     ## ##       ##     ## ##     ## ##       ##
	   ##    ## ##    ## ##     ## ##       ##     ## ##     ## ##       ##
	    ######   ######  ##     ## ######## ########   #######  ######## ########
	*/

	c.HelpBreak()
	c.HelpGroup("SCHEDULES:")
	c.Dispatch("schedules", "List available backup schedules",
		func(opts Options, args []string, help bool) error {
			if help {
				HelpListMacro("schedule", "schedules")
				JSONHelp(`[{"uuid":"86ff3fec-76c5-48c4-880d-c37563033613","name":"TestSched","summary":"A Test Schedule","when":"daily 4am"}]`)
				return nil
			}

			DEBUG("running 'list schedules' command")
			DEBUG("  show unused? %v", *opts.Unused)
			DEBUG("  show in-use? %v", *opts.Used)
			if *opts.Raw {
				DEBUG(" fuzzy search? %v", MaybeBools(*opts.Fuzzy, *opts.Raw).Yes)
			}

			schedules, err := GetSchedules(ScheduleFilter{
				Name:       strings.Join(args, " "),
				Unused:     MaybeBools(*opts.Unused, *opts.Used),
				ExactMatch: Opposite(MaybeBools(*opts.Fuzzy, *opts.Raw)),
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
	c.Alias("list schedules", "schedules")
	c.Alias("ls schedules", "schedules")

	c.Dispatch("schedule", "Print detailed information about a specific backup schedule",
		func(opts Options, args []string, help bool) error {
			if help {
				HelpShowMacro("schedule", "schedules")
				JSONHelp(`{"uuid":"9a58a3fa-7457-431c-b094-e201b42b5c7b","name":"TestSched","summary":"A Test Schedule","when":"daily 4am"}`)
				return nil
			}

			DEBUG("running 'show schedule' command")

			schedule, _, err := FindSchedule(strings.Join(args, " "), *opts.Raw)
			if err != nil {
				return err
			}

			if *opts.Raw {
				return RawJSON(schedule)
			}

			ShowSchedule(schedule)
			return nil
		})
	c.Alias("show schedule", "schedule")
	c.Alias("view schedule", "schedule")
	c.Alias("display schedule", "schedule")
	c.Alias("list schedule", "schedule")
	c.Alias("ls schedule", "schedule")

	c.Dispatch("create-schedule", "Create a new backup schedule",
		func(opts Options, args []string, help bool) error {
			if help {
				InputHelp(`{"name":"TestSched","summary":"A Test Schedule","when":"daily 4am"}`)
				JSONHelp(`{"uuid":"9a58a3fa-7457-431c-b094-e201b42b5c7b","name":"TestSched","summary":"A Test Schedule","when":"daily 4am"}`)
				HelpCreateMacro("schedule", "schedules")
				return nil
			}

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
			return c.Execute("schedule", s.UUID)
		})
	c.Alias("create schedule", "create-schedule")
	c.Alias("new schedule", "create-schedule")
	c.Alias("create new schedule", "create-schedule")
	c.Alias("make schedule", "create-schedule")
	c.Alias("c s", "create-schedule")

	c.Dispatch("edit-schedule", "Modify an existing backup schedule",
		func(opts Options, args []string, help bool) error {
			if help {
				InputHelp(`{"name":"AnotherSched","summary":"A Test Schedule","when":"daily 4am"}`)
				HelpEditMacro("schedule", "schedules")
				JSONHelp(`{"uuid":"9a58a3fa-7457-431c-b094-e201b42b5c7b","name":"AnotherSched","summary":"A Test Schedule","when":"daily 4am"}`)
				return nil
			}

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
			return c.Execute("schedule", s.UUID)
		})
	c.Alias("edit schedule", "edit-schedule")
	c.Alias("update schedule", "edit-schedule")

	c.Dispatch("delete-schedule", "Delete a backup schedule",
		func(opts Options, args []string, help bool) error {
			if help {
				HelpDeleteMacro("schedule", "schedules")
				return nil
			}

			DEBUG("running 'delete schedule' command")

			schedule, id, err := FindSchedule(strings.Join(args, " "), *opts.Raw)
			if err != nil {
				return err
			}

			if !*opts.Raw {
				ShowSchedule(schedule)
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
	c.Alias("delete schedule", "delete-schedule")
	c.Alias("remove schedule", "delete-schedule")
	c.Alias("rm schedule", "delete-schedule")

	/*
	   ########  ######## ######## ######## ##    ## ######## ####  #######  ##    ##
	   ##     ## ##          ##    ##       ###   ##    ##     ##  ##     ## ###   ##
	   ##     ## ##          ##    ##       ####  ##    ##     ##  ##     ## ####  ##
	   ########  ######      ##    ######   ## ## ##    ##     ##  ##     ## ## ## ##
	   ##   ##   ##          ##    ##       ##  ####    ##     ##  ##     ## ##  ####
	   ##    ##  ##          ##    ##       ##   ###    ##     ##  ##     ## ##   ###
	   ##     ## ########    ##    ######## ##    ##    ##    ####  #######  ##    ##
	*/

	c.HelpBreak()
	c.HelpGroup("POLICIES:")
	c.Dispatch("policies", "List available retention policies",
		func(opts Options, args []string, help bool) error {
			if help {
				HelpListMacro("policy", "policies")
				JSONHelp(`[{"uuid":"8c6f894f-9c27-475f-ad5a-8c0db37926ec","name":"apolicy","summary":"a policy","expires":5616000}]`)
				return nil
			}

			DEBUG("running 'list retention policies' command")
			DEBUG("  show unused? %v", *opts.Unused)
			DEBUG("  show in-use? %v", *opts.Used)
			if *opts.Raw {
				DEBUG(" fuzzy search? %v", MaybeBools(*opts.Fuzzy, *opts.Raw).Yes)
			}

			policies, err := GetRetentionPolicies(RetentionPolicyFilter{
				Name:       strings.Join(args, " "),
				Unused:     MaybeBools(*opts.Unused, *opts.Used),
				ExactMatch: Opposite(MaybeBools(*opts.Fuzzy, *opts.Raw)),
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
	c.Alias("list retention policies", "policies")
	c.Alias("ls retention policies", "policies")
	c.Alias("list policies", "policies")
	c.Alias("ls policies", "policies")

	c.Dispatch("policy", "Print detailed information about a specific retention policy",
		func(opts Options, args []string, help bool) error {
			if help {
				HelpShowMacro("policy", "policies")
				JSONHelp(`{"uuid":"8c6f894f-9c27-475f-ad5a-8c0db37926ec","name":"apolicy","summary":"a policy","expires":5616000}`)
				return nil
			}

			DEBUG("running 'show retention policy' command")

			policy, _, err := FindRetentionPolicy(strings.Join(args, " "), *opts.Raw)
			if err != nil {
				return err
			}

			if *opts.Raw {
				return RawJSON(policy)
			}

			ShowRetentionPolicy(policy)
			return nil
		})
	c.Alias("show retention policy", "policy")
	c.Alias("view retention policy", "policy")
	c.Alias("display retention policy", "policy")
	c.Alias("list retention policy", "policy")
	c.Alias("show policy", "policy")
	c.Alias("view policy", "policy")
	c.Alias("display policy", "policy")
	c.Alias("list policy", "policy")

	c.Dispatch("create-policy", "Create a new retention policy",
		func(opts Options, args []string, help bool) error {
			if help {
				InputHelp(`{"expires":31536000,"name":"TestPolicy","summary":"A Test Policy"}`)
				JSONHelp(`{"uuid":"18a446c4-c068-4c09-886c-cb77b6a85274","name":"TestPolicy","summary":"A Test Policy","expires":31536000}`)
				HelpCreateMacro("policy", "policies")
				return nil
			}

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
			return c.Execute("policy", p.UUID)
		})
	c.Alias("create retention policy", "create-policy")
	c.Alias("new retention policy", "create-policy")
	c.Alias("create new retention policy", "create-policy")
	c.Alias("make retention policy", "create-policy")
	c.Alias("create policy", "create-policy")
	c.Alias("new policy", "create-policy")
	c.Alias("create new policy", "create-policy")
	c.Alias("make policy", "create-policy")
	c.Alias("c p", "create-policy")

	c.Dispatch("edit-policy", "Modify an existing retention policy",
		func(opts Options, args []string, help bool) error {
			if help {
				HelpEditMacro("policy", "policies")
				InputHelp(`{"expires":31536000,"name":"AnotherPolicy","summary":"A Test Policy"}`)
				JSONHelp(`{"uuid":"18a446c4-c068-4c09-886c-cb77b6a85274","name":"AnotherPolicy","summary":"A Test Policy","expires":31536000}`)
				return nil
			}

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
				in.NewField("Retention Timeframe, in days", "expires", p.Expires/86400, "", FieldIsRetentionTimeframe)

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
			return c.Execute("policy", p.UUID)
		})
	c.Alias("edit retention policy", "edit-policy")
	c.Alias("update retention policy", "edit-policy")
	c.Alias("edit policy", "edit-policy")
	c.Alias("update policy", "edit-policy")

	c.Dispatch("delete-policy", "Delete a retention policy",
		func(opts Options, args []string, help bool) error {
			if help {
				HelpDeleteMacro("policy", "policies")
				return nil
			}

			DEBUG("running 'delete retention policy' command")

			policy, id, err := FindRetentionPolicy(strings.Join(args, " "), *opts.Raw)
			if err != nil {
				return err
			}

			if !*opts.Raw {
				ShowRetentionPolicy(policy)
				if !tui.Confirm("Really delete this retention policy?") {
					return fmt.Errorf("Cancelling...")
				}
			}

			if err := DeleteRetentionPolicy(id); err != nil {
				return err
			}

			OK("Deleted policy")
			return nil
		})
	c.Alias("delete retention policy", "delete-policy")
	c.Alias("remove retention policy", "delete-policy")
	c.Alias("rm retention policy", "delete-policy")
	c.Alias("delete policy", "delete-policy")
	c.Alias("remove policy", "delete-policy")
	c.Alias("rm policy", "delete-policy")

	/*
	    ######  ########  #######  ########  ########
	   ##    ##    ##    ##     ## ##     ## ##
	   ##          ##    ##     ## ##     ## ##
	    ######     ##    ##     ## ########  ######
	         ##    ##    ##     ## ##   ##   ##
	   ##    ##    ##    ##     ## ##    ##  ##
	    ######     ##     #######  ##     ## ########
	*/

	c.HelpBreak()
	c.HelpGroup("STORES:")
	c.Dispatch("stores", "List available archive stores",
		func(opts Options, args []string, help bool) error {
			if help {
				HelpListMacro("store", "stores")
				FlagHelp("Only show stores using the named store plugin", true, "-P", "--policy=value")
				JSONHelp(`[{"uuid":"6e83bfb7-7ae1-4f0f-88a8-84f0fe4bae20","name":"test store","summary":"a test store named \"test store\"","plugin":"s3","endpoint":"{ \"endpoint\": \"doesntmatter\" }"}]`)
				return nil
			}

			DEBUG("running 'list stores' command")
			DEBUG("  for plugin: '%s'", *opts.Plugin)
			DEBUG("  show unused? %v", *opts.Unused)
			DEBUG("  show in-use? %v", *opts.Used)
			if *opts.Raw {
				DEBUG(" fuzzy search? %v", MaybeBools(*opts.Fuzzy, *opts.Raw).Yes)
			}

			stores, err := GetStores(StoreFilter{
				Name:       strings.Join(args, " "),
				Plugin:     *opts.Plugin,
				Unused:     MaybeBools(*opts.Unused, *opts.Used),
				ExactMatch: Opposite(MaybeBools(*opts.Fuzzy, *opts.Raw)),
			})
			if err != nil {
				return err
			}

			if *opts.Raw {
				return RawJSON(stores)
			}

			t := tui.NewTable("Name", "Summary", "Plugin", "Configuration")
			for _, store := range stores {
				t.Row(store, store.Name, store.Summary, store.Plugin, PrettyJSON(store.Endpoint))
			}
			t.Output(os.Stdout)
			return nil
		})
	c.Alias("list stores", "stores")
	c.Alias("ls stores", "stores")

	c.Dispatch("store", "Print detailed information about a specific archive store",
		func(opts Options, args []string, help bool) error {
			if help {
				JSONHelp(`{"uuid":"6e83bfb7-7ae1-4f0f-88a8-84f0fe4bae20","name":"test store","summary":"a test store named \"test store\"","plugin":"s3","endpoint":"{ \"endpoint\": \"doesntmatter\" }"}`)
				HelpShowMacro("store", "stores")
				return nil
			}

			DEBUG("running 'show store' command")

			store, _, err := FindStore(strings.Join(args, " "), *opts.Raw)
			if err != nil {
				return err
			}

			if *opts.Raw {
				return RawJSON(store)
			}

			ShowStore(store)
			return nil
		})
	c.Alias("show store", "store")
	c.Alias("view store", "store")
	c.Alias("display store", "store")
	c.Alias("list store", "store")
	c.Alias("ls store", "store")

	c.Dispatch("create-store", "Create a new archive store",
		func(opts Options, args []string, help bool) error {
			if help {
				HelpCreateMacro("store", "stores")
				InputHelp(`{"endpoint":"{\"endpoint\":\"schmendpoint\"}","name":"TestStore","plugin":"s3","summary":"A Test Store"}`)
				JSONHelp(`{"uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf","name":"TestStore","summary":"A Test Store","plugin":"s3","endpoint":"{\"endpoint\":\"schmendpoint\"}"}`)
				return nil
			}

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
				in.NewField("Plugin Name", "plugin", "", "", FieldIsPluginName)
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
			return c.Execute("store", s.UUID)
		})
	c.Alias("create store", "create-store")
	c.Alias("new store", "create-store")
	c.Alias("create new store", "create-store")
	c.Alias("make store", "create-store")
	c.Alias("c st", "create-store")

	c.Dispatch("edit-store", "Modify an existing archive store",
		func(opts Options, args []string, help bool) error {
			if help {
				HelpEditMacro("store", "stores")
				InputHelp(`{"endpoint":"{\"endpoint\":\"schmendpoint\"}","name":"AnotherStore","plugin":"s3","summary":"A Test Store"}`)
				JSONHelp(`{"uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf","name":"AnotherStore","summary":"A Test Store","plugin":"s3","endpoint":"{\"endpoint\":\"schmendpoint\"}"}`)
				return nil
			}

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
				in.NewField("Plugin Name", "plugin", s.Plugin, "", FieldIsPluginName)
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
			return c.Execute("store", s.UUID)
		})
	c.Alias("edit store", "edit-store")
	c.Alias("update store", "edit-store")

	c.Dispatch("delete-store", "Delete an archive store",
		func(opts Options, args []string, help bool) error {
			if help {
				HelpDeleteMacro("store", "stores")
				return nil
			}

			DEBUG("running 'delete store' command")

			store, id, err := FindStore(strings.Join(args, " "), *opts.Raw)
			if err != nil {
				return err
			}

			if !*opts.Raw {
				ShowStore(store)
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
	c.Alias("delete store", "delete-store")
	c.Alias("remove store", "delete-store")
	c.Alias("rm store", "delete-store")

	/*
	         ##  #######  ########
	         ## ##     ## ##     ##
	         ## ##     ## ##     ##
	         ## ##     ## ########
	   ##    ## ##     ## ##     ##
	   ##    ## ##     ## ##     ##
	    ######   #######  ########
	*/

	c.HelpBreak()
	c.HelpGroup("JOBS:")
	c.Dispatch("jobs", "List available backup jobs",
		func(opts Options, args []string, help bool) error {
			if help {
				HelpListMacro("job", "jobs")
				FlagHelp("Show only jobs using the specified target", true, "-t", "--target=value")
				FlagHelp("Show only jobs using the specified store", true, "-s", "--store=value")
				FlagHelp("Show only jobs using the specified schedule", true, "-w", "--schedule=value")
				FlagHelp("Show only jobs using the specified retention policy", true, "-p", "--policy=value")
				FlagHelp("Show only jobs which are in the paused state", true, "--paused")
				FlagHelp("Show only jobs which are NOT in the paused state", true, "--unpaused")
				JSONHelp(`[{"uuid":"f6623a6f-8dce-46b2-a293-5525bc3a3588","name":"TestJob","summary":"A Test Job","retention_name":"AnotherPolicy","retention_uuid":"18a446c4-c068-4c09-886c-cb77b6a85274","expiry":31536000,"schedule_name":"AnotherSched","schedule_uuid":"9a58a3fa-7457-431c-b094-e201b42b5c7b","schedule_when":"daily 4am","paused":true,"store_uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf","store_name":"AnotherStore","store_plugin":"s3","store_endpoint":"{\"endpoint\":\"schmendpoint\"}","target_uuid":"84751f04-2be2-428d-b6a3-2022c63bf6ee","target_name":"TestTarget","target_plugin":"postgres","target_endpoint":"{\"endpoint\":\"schmendpoint\"}","agent":"127.0.0.1:1234"}]`)
				return nil
			}

			DEBUG("running 'list jobs' command")
			DEBUG("  for target:      '%s'", *opts.Target)
			DEBUG("  for store:       '%s'", *opts.Store)
			DEBUG("  for schedule:    '%s'", *opts.Schedule)
			DEBUG("  for ret. policy: '%s'", *opts.Retention)
			DEBUG("  show paused?      %v", *opts.Paused)
			DEBUG("  show unpaused?    %v", *opts.Unpaused)
			if *opts.Raw {
				DEBUG(" fuzzy search? %v", MaybeBools(*opts.Fuzzy, *opts.Raw).Yes)
			}

			jobs, err := GetJobs(JobFilter{
				Name:       strings.Join(args, " "),
				Paused:     MaybeBools(*opts.Paused, *opts.Unpaused),
				Target:     *opts.Target,
				Store:      *opts.Store,
				Schedule:   *opts.Schedule,
				Retention:  *opts.Retention,
				ExactMatch: Opposite(MaybeBools(*opts.Fuzzy, *opts.Raw)),
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
					job.RetentionName, job.ScheduleName, job.Agent, PrettyJSON(job.TargetEndpoint))
			}
			t.Output(os.Stdout)
			return nil
		})
	c.Alias("list jobs", "jobs")
	c.Alias("ls jobs", "jobs")
	c.Alias("ls j", "jobs")

	c.Dispatch("job", "Print detailed information about a specific backup job",
		func(opts Options, args []string, help bool) error {
			if help {
				HelpShowMacro("job", "jobs")
				JSONHelp(`{"uuid":"f6623a6f-8dce-46b2-a293-5525bc3a3588","name":"TestJob","summary":"A Test Job","retention_name":"AnotherPolicy","retention_uuid":"18a446c4-c068-4c09-886c-cb77b6a85274","expiry":31536000,"schedule_name":"AnotherSched","schedule_uuid":"9a58a3fa-7457-431c-b094-e201b42b5c7b","schedule_when":"daily 4am","paused":true,"store_uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf","store_name":"AnotherStore","store_plugin":"s3","store_endpoint":"{\"endpoint\":\"schmendpoint\"}","target_uuid":"84751f04-2be2-428d-b6a3-2022c63bf6ee","target_name":"TestTarget","target_plugin":"postgres","target_endpoint":"{\"endpoint\":\"schmendpoint\"}","agent":"127.0.0.1:1234"}`)
				return nil
			}

			DEBUG("running 'show job' command")

			job, _, err := FindJob(strings.Join(args, " "), *opts.Raw)
			if err != nil {
				return err
			}

			if *opts.Raw {
				return RawJSON(job)
			}

			ShowJob(job)
			return nil
		})
	c.Alias("show job", "job")
	c.Alias("view job", "job")
	c.Alias("display job", "job")
	c.Alias("list job", "job")
	c.Alias("ls job", "job")

	c.Dispatch("create-job", "Create a new backup job",
		func(opts Options, args []string, help bool) error {
			if help {
				HelpCreateMacro("job", "jobs")
				InputHelp(`{"name":"TestJob","paused":true,"retention":"18a446c4-c068-4c09-886c-cb77b6a85274","schedule":"9a58a3fa-7457-431c-b094-e201b42b5c7b","store":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf","summary":"A Test Job","target":"84751f04-2be2-428d-b6a3-2022c63bf6ee"}`)
				JSONHelp(`{"uuid":"f6623a6f-8dce-46b2-a293-5525bc3a3588","name":"TestJob","summary":"A Test Job","retention_name":"AnotherPolicy","retention_uuid":"18a446c4-c068-4c09-886c-cb77b6a85274","expiry":31536000,"schedule_name":"AnotherSched","schedule_uuid":"9a58a3fa-7457-431c-b094-e201b42b5c7b","schedule_when":"daily 4am","paused":true,"store_uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf","store_name":"AnotherStore","store_plugin":"s3","store_endpoint":"{\"endpoint\":\"schmendpoint\"}","target_uuid":"84751f04-2be2-428d-b6a3-2022c63bf6ee","target_name":"TestTarget","target_plugin":"postgres","target_endpoint":"{\"endpoint\":\"schmendpoint\"}","agent":"127.0.0.1:1234"}`)
				return nil
			}

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

				in.NewField("Paused?", "paused", "no", "", tui.FieldIsBoolean)
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
			return c.Execute("job", job.UUID)
		})
	c.Alias("create job", "create-job")
	c.Alias("new job", "create-job")
	c.Alias("create new job", "create-job")
	c.Alias("make job", "create-job")
	c.Alias("c j", "create-job")

	c.Dispatch("edit-job", "Modify an existing backup job",
		func(opts Options, args []string, help bool) error {
			if help {
				HelpEditMacro("job", "jobs")

				return nil
			}

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

				in := tui.NewForm()
				in.NewField("Job Name", "name", j.Name, "", tui.FieldIsRequired)
				in.NewField("Summary", "summary", j.Summary, "", tui.FieldIsOptional)
				in.NewField("Store", "store", j.StoreUUID, j.StoreName, FieldIsStoreUUID)
				in.NewField("Target", "target", j.TargetUUID, j.TargetName, FieldIsTargetUUID)
				in.NewField("Retention Policy", "retention", j.RetentionUUID, fmt.Sprintf("%s - %dd", j.RetentionName, j.Expiry/86400), FieldIsRetentionPolicyUUID)
				in.NewField("Schedule", "schedule", j.ScheduleUUID, fmt.Sprintf("%s - %s", j.ScheduleName, j.ScheduleWhen), FieldIsScheduleUUID)

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
			return c.Execute("job", j.UUID)
		})
	c.Alias("edit job", "edit-job")
	c.Alias("update job", "edit-job")

	c.Dispatch("delete-job", "Delete a backup job",
		func(opts Options, args []string, help bool) error {
			if help {
				HelpDeleteMacro("job", "jobs")
				InputHelp(`{"name":"AnotherJob","retention":"18a446c4-c068-4c09-886c-cb77b6a85274","schedule":"9a58a3fa-7457-431c-b094-e201b42b5c7b","store":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf","summary":"A Test Job","target":"84751f04-2be2-428d-b6a3-2022c63bf6ee"}`)
				JSONHelp(`{"uuid":"f6623a6f-8dce-46b2-a293-5525bc3a3588","name":"AnotherJob","summary":"A Test Job","retention_name":"AnotherPolicy","retention_uuid":"18a446c4-c068-4c09-886c-cb77b6a85274","expiry":31536000,"schedule_name":"AnotherSched","schedule_uuid":"9a58a3fa-7457-431c-b094-e201b42b5c7b","schedule_when":"daily 4am","paused":true,"store_uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf","store_name":"AnotherStore","store_plugin":"s3","store_endpoint":"{\"endpoint\":\"schmendpoint\"}","target_uuid":"84751f04-2be2-428d-b6a3-2022c63bf6ee","target_name":"TestTarget","target_plugin":"postgres","target_endpoint":"{\"endpoint\":\"schmendpoint\"}","agent":"127.0.0.1:1234"}`)
				return nil
			}

			DEBUG("running 'delete job' command")

			job, id, err := FindJob(strings.Join(args, " "), *opts.Raw)
			if err != nil {
				return err
			}

			if !*opts.Raw {
				ShowJob(job)
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
	c.Alias("delete job", "delete-job")
	c.Alias("remove job", "delete-job")
	c.Alias("rm job", "delete-job")

	c.Dispatch("pause", "Pause a backup job",
		func(opts Options, args []string, help bool) error {
			if help {
				FlagHelp(`A string partially matching the name of a job to pause 
				or a UUID exactly matching the UUID of a job to pause.
				Not setting this value explicitly will default it to the empty string.`,
					false, "<job>")
				HelpKMacro()
				return nil
			}

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
	c.Alias("pause job", "pause")

	c.Dispatch("unpause", "Unpause a backup job",
		func(opts Options, args []string, help bool) error {
			if help {
				FlagHelp(`A string partially matching the name of a job to unpause 
				or a UUID exactly matching the UUID of a job to unpause.
				Not setting this value explicitly will default it to the empty string.`,
					false, "<job>")
				HelpKMacro()
				return nil
			}

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
	c.Alias("unpause job", "unpause")

	c.Dispatch("run", "Schedule an immediate run of a backup job",
		func(opts Options, args []string, help bool) error {
			if help {

				MessageHelp("Note: If raw mode is specified and the targeted SHIELD backend does not support handing back the task uuid, the task_uuid in the JSON will be the empty string")
				FlagHelp(`A string partially matching the name of a job to run 
				or a UUID exactly matching the UUID of a job to run.
				Not setting this value explicitly will default it to the empty string.`,
					false, "<job>")
				JSONHelp(`{"ok":"Scheduled immediate run of job","task_uuid":"143e5494-63c4-4e05-9051-8b3015eae061"}`)
				HelpKMacro()
				return nil
			}

			DEBUG("running 'run job' command")

			_, id, err := FindJob(strings.Join(args, " "), *opts.Raw)
			if err != nil {
				return err
			}

			var params = struct {
				Owner string `json:"owner"`
			}{
				Owner: CurrentUser(),
			}

			b, err := json.Marshal(params)
			if err != nil {
				return err
			}

			taskUUID, err := RunJob(id, string(b))
			if err != nil {
				return err
			}

			if *opts.Raw {
				RawJSON(map[string]interface{}{
					"ok":        "Scheduled immediate run of job",
					"task_uuid": taskUUID,
				})
			} else {
				OK("Scheduled immediate run of job")
				if taskUUID != "" {
					ansi.Printf("To view task, type @B{shield task %s}\n", taskUUID)
				}
			}

			return nil
		})
	c.Alias("run job", "run")

	/*
	   ########    ###     ######  ##    ##
	      ##      ## ##   ##    ## ##   ##
	      ##     ##   ##  ##       ##  ##
	      ##    ##     ##  ######  #####
	      ##    #########       ## ##  ##
	      ##    ##     ## ##    ## ##   ##
	      ##    ##     ##  ######  ##    ##
	*/

	c.HelpBreak()
	c.HelpGroup("TASKS:")
	c.Dispatch("tasks", "List available tasks",
		func(opts Options, args []string, help bool) error {
			if help {
				FlagHelp(`Only show tasks with the specified status
									Valid values are one of ['all', 'running', 'pending', 'cancelled']
									If not explicitly set, it defaults to 'running'`,
					true, "-S", "--status=value")
				FlagHelp(`Show all tasks, regardless of state`, true, "-a", "--all")
				FlagHelp("Returns information as a JSON object", true, "--raw")
				FlagHelp("Show only the <value> most recent tasks", true, "--limit=value")
				HelpKMacro()
				JSONHelp(`[{"uuid":"0e3736f3-6905-40ba-9adc-06641a282ff4","owner":"system","type":"backup","job_uuid":"9b39b2ed-04dc-4de4-9ee8-265a3f9000e8","archive_uuid":"2a4147ea-84a6-40fc-8028-143efabcc49d","status":"done","started_at":"2016-05-17 11:00:01","stopped_at":"2016-05-17 11:00:02","timeout_at":"","log":"This is where I would put my plugin output if I had one"}]`)
				return nil
			}

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
				Limit:  *options.Limit,
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
				stopped := "(not yet started)"
				if !task.StartedAt.IsZero() {
					stopped = "(running)"
					started = task.StartedAt.Format(time.RFC1123Z)
				}

				if !task.StoppedAt.IsZero() {
					stopped = task.StoppedAt.Format(time.RFC1123Z)
				}

				t.Row(task, task.UUID, task.Owner, task.Op, job[task.JobUUID].Agent, task.Status, started, stopped)
			}
			t.Output(os.Stdout)
			return nil
		})
	c.Alias("list tasks", "tasks")
	c.Alias("ls tasks", "tasks")

	c.Dispatch("task", "Print detailed information about a specific task",
		func(opts Options, args []string, help bool) error {
			if help {
				FlagHelp("The ID number of the task to show info about", false, "<id>")
				HelpKMacro()
				FlagHelp("Returns information as a JSON object", true, "--raw")
				JSONHelp(`{"uuid":"b40ae708-6215-4932-90fb-fe580fac7196","owner":"system","type":"backup","job_uuid":"9b39b2ed-04dc-4de4-9ee8-265a3f9000e8","archive_uuid":"62792b22-c89e-4d69-b874-69a5f056a9ef","status":"done","started_at":"2016-05-18 11:00:01","stopped_at":"2016-05-18 11:00:02","timeout_at":"","log":"This is where I would put my plugin output if I had one"}`)
				return nil
			}

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

			ShowTask(task)
			return nil
		})
	c.Alias("show task", "task")
	c.Alias("view task", "task")
	c.Alias("display task", "task")
	c.Alias("list task", "task")
	c.Alias("ls task", "task")

	c.Dispatch("cancel-task", "Cancel a running or pending task",
		func(opts Options, args []string, help bool) error {
			if help {
				FlagHelp(`Outputs the result as a JSON object.
				The cli will not prompt for confirmation in raw mode.`, true, "--raw")
				HelpKMacro()
				JSONHelp(`{"ok":"Cancelled task '81746508-bd18-46a8-842e-97911d4b23a3'\n"}`)
				return nil
			}

			DEBUG("running 'cancel task' command")

			require(len(args) == 1, "shield cancel task <UUID>")
			id := uuid.Parse(args[0])
			DEBUG("  task UUID = '%s'", id)

			task, err := GetTask(id)
			if err != nil {
				return err
			}

			if !*opts.Raw {
				ShowTask(task)
				if !tui.Confirm("Really cancel this task?") {
					return fmt.Errorf("Cancelling...")
				}
			}

			if err := CancelTask(id); err != nil {
				return err
			}

			OK("Cancelled task '%s'\n", id)
			return nil
		})
	c.Alias("cancel task", "cancel-task")
	c.Alias("stop task", "cancel-task")

	/*
	      ###    ########   ######  ##     ## #### ##     ## ########
	     ## ##   ##     ## ##    ## ##     ##  ##  ##     ## ##
	    ##   ##  ##     ## ##       ##     ##  ##  ##     ## ##
	   ##     ## ########  ##       #########  ##  ##     ## ######
	   ######### ##   ##   ##       ##     ##  ##   ##   ##  ##
	   ##     ## ##    ##  ##    ## ##     ##  ##    ## ##   ##
	   ##     ## ##     ##  ######  ##     ## ####    ###    ########
	*/

	c.HelpBreak()
	c.HelpGroup("ARCHIVES:")
	c.Dispatch("archives", "List available backup archives",
		func(opts Options, args []string, help bool) error {
			if help {
				HelpListMacro("archive", "archives")
				FlagHelp(`Only show archives with the specified state of validity.
									Accepted values are one of ['all', 'valid']
									If not explicitly set, it defaults to 'valid'`,
					true, "-S", "--status=value")
				FlagHelp("Show only archives created from the specified target", true, "-t", "--target=value")
				FlagHelp("Show only archives sent to the specified store", true, "-s", "--store=value")
				FlagHelp("Show only the <value> most recent archives", true, "--limit=value")
				FlagHelp(`Show only the archives taken before this point in time
				Specify in the format YYYYMMDD`, true, "-B", "--before=value")
				FlagHelp(`Show only the archives taken after this point in time
				Specify in the format YYYYMMDD`, true, "-A", "--after=value")
				FlagHelp(`Show all archives, regardless of validity.
									Equivalent to '--status=all'`, true, "-a", "--all")
				JSONHelp(`[{"uuid":"b4a842c5-cb61-4fa1-b0c7-08260fdc3533","key":"thisisastorekey","taken_at":"2016-05-18 11:02:43","expires_at":"2017-05-18 11:02:43","status":"valid","notes":"","target_uuid":"b7aa8269-008d-486a-ba1b-610ee191e4c1","target_plugin":"redis-broker","target_endpoint":"{\"redis_type\":\"broker\"}","store_uuid":"6d52c95f-8d7f-4697-ae32-b9ce51fb4808","store_plugin":"s3","store_endpoint":"{\"endpoint\":\"schmendpoint\"}"}]`)
				return nil
			}

			DEBUG("running 'list archives' command")

			if *options.Status == "" {
				*options.Status = "valid"
			}
			if *options.Status == "all" || *options.All {
				*options.Status = ""
			}
			DEBUG("  for status: '%s'", *opts.Status)

			if *options.Limit == "" {
				*options.Limit = "20"
			}
			DEBUG("  for limit: '%s'", *opts.Limit)

			archives, err := GetArchives(ArchiveFilter{
				Target: *options.Target,
				Store:  *options.Store,
				Before: *options.Before,
				After:  *options.After,
				Status: *options.Status,
				Limit:  *options.Limit,
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

			t := tui.NewTable("UUID", "Target", "Restore IP", "Store", "Taken at", "Expires at", "Status", "Notes")
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
	c.Alias("list archives", "archives")
	c.Alias("ls archives", "archives")

	c.Dispatch("archive", "Print detailed information about a backup archive",
		func(opts Options, args []string, help bool) error {
			if help {
				FlagHelp(`A UUID assigned to a single archive instance`, false, "<uuid>")
				FlagHelp("Returns information as a JSON object", true, "--raw")
				HelpKMacro()
				JSONHelp(`{"uuid":"b4a842c5-cb61-4fa1-b0c7-08260fdc3533","key":"thisisastorekey","taken_at":"2016-05-18 11:02:43","expires_at":"2017-05-18 11:02:43","status":"valid","notes":"","target_uuid":"b7aa8269-008d-486a-ba1b-610ee191e4c1","target_plugin":"redis-broker","target_endpoint":"{\"redis_type\":\"broker\"}","store_uuid":"6d52c95f-8d7f-4697-ae32-b9ce51fb4808","store_plugin":"s3","store_endpoint":"{\"endpoint\":\"schmendpoint\"}"}`)
				return nil
			}

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

			ShowArchive(archive)
			return nil
		})
	c.Alias("show archive", "archive")
	c.Alias("view archive", "archive")
	c.Alias("display archive", "archive")
	c.Alias("list archive", "archive")
	c.Alias("ls archive", "archive")

	c.Dispatch("restore", "Restore a backup archive",
		func(opts Options, args []string, help bool) error {
			if help {
				MessageHelp("Note: If raw mode is specified and the targeted SHIELD backend does not support handing back the task uuid, the task_uuid in the JSON will be the empty string")
				FlagHelp(`Outputs the result as a JSON object.`, true, "--raw")
				FlagHelp(`The name or UUID of a single target to restore. In raw mode, it must be a UUID assigned to a single archive instance`, false, "<target or uuid>")
				HelpKMacro()
				return nil
			}

			DEBUG("running 'restore archive' command")

			var id uuid.UUID

			if *opts.Raw {
				require(len(args) == 1, "USAGE: shield restore archive <UUID>")
				id = uuid.Parse(args[0])
				DEBUG("  trying archive UUID '%s'", args[0])

			} else {
				target, _, err := FindTarget(strings.Join(args, " "), false)
				if err != nil {
					return err
				}

				_, id, err = FindArchivesFor(target, 10)
				if err != nil {
					return err
				}
			}
			DEBUG("  archive UUID = '%s'", id)

			var params = struct {
				Owner  string `json:"owner,omitempty"`
				Target string `json:"target,omitempty"`
			}{
				Owner: CurrentUser(),
			}

			if *opts.To != "" {
				params.Target = *opts.To
			}

			b, err := json.Marshal(params)
			if err != nil {
				return err
			}

			taskUUID, err := RestoreArchive(id, string(b))
			if err != nil {
				return err
			}

			targetMsg := ""
			if params.Target != "" {
				targetMsg = fmt.Sprintf("to target '%s'", params.Target)
			}
			if *opts.Raw {
				RawJSON(map[string]interface{}{
					"ok":        fmt.Sprintf("Scheduled immediate restore of archive '%s' %s", id, targetMsg),
					"task_uuid": taskUUID,
				})
			} else {
				//`OK` handles raw checking
				OK("Scheduled immediate restore of archive '%s' %s", id, targetMsg)
				if taskUUID != "" {
					ansi.Printf("To view task, type @B{shield task %s}\n", taskUUID)
				}
			}

			return nil
		})
	c.Alias("restore archive", "restore")
	c.Alias("restore-archive", "restore")

	c.Dispatch("delete-archive", "Delete a backup archive",
		func(opts Options, args []string, help bool) error {
			if help {
				FlagHelp(`A UUID assigned to a single archive instance`, false, "<uuid>")
				FlagHelp(`Outputs the result as a JSON object.
				The cli will not prompt for confirmation in raw mode.`, true, "--raw")
				HelpKMacro()
				JSONHelp(`{"ok":"Deleted archive"}`)
				return nil
			}

			DEBUG("running 'delete archive' command")

			require(len(args) == 1, "USAGE: shield delete archive <UUID>")
			id := uuid.Parse(args[0])
			DEBUG("  archive UUID = '%s'", id)

			archive, err := GetArchive(id)
			if err != nil {
				return err
			}

			if !*opts.Raw {
				ShowArchive(archive)
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
	c.Alias("delete archive", "delete-archive")
	c.Alias("remove archive", "delete-archive")
	c.Alias("rm archive", "delete-archive")

	/**************************************************************************/
	err := LoadConfig(*options.Config)
	if err != nil {
		ansi.Fprintf(os.Stderr, "\n@R{ERROR:} Could not parse %s: %s\n", *options.Config, err)
		os.Exit(1)
	}

	// only check for backends + creds if we aren't manipulating backends/help
	nonAPICommands := regexp.MustCompile(`(help|commands|flags|options|backends|list backends|ls be|create backend|c be|update backend|backend|use backend)`)
	if !nonAPICommands.MatchString(strings.Join(command, " ")) {
		DEBUG("Command: '%s'", strings.Join(command, " "))

		if *options.Shield != "" || os.Getenv("SHIELD_API") != "" {
			ansi.Fprintf(os.Stderr, "@Y{WARNING: -H, --host, and the SHIELD_API environment variable have been deprecated and will be removed in a later release.} Use `shield backend` instead\n")
		}

		if len(Cfg.Backends) == 0 {
			backend := os.Getenv("SHIELD_API")
			if *options.Shield != "" {
				backend = *options.Shield
			}

			if backend != "" {
				ansi.Fprintf(os.Stderr, "@C{Initializing `default` backend as `%s`}\n", backend)
				err := Cfg.AddBackend(backend, "default")
				if err != nil {
					ansi.Fprintf(os.Stderr, "@R{Error creating `default` backend: %s}", err)
				}
				Cfg.UseBackend("default")
			}
		}

		if Cfg.BackendURI() == "" {
			ansi.Fprintf(os.Stderr, "@R{No backend targeted. Use `shield list backends` and `shield backend` to target one}\n")
			os.Exit(1)
		}

		err = Cfg.Save()
		if err != nil {
			DEBUG("Unable to save shield config: %s", err)
		}
	}

	if err := c.Execute(command...); err != nil {
		if *options.Raw {
			_ = RawJSON(map[string]string{"error": err.Error()})
		} else {
			ansi.Fprintf(os.Stderr, "@R{%s}\n", err)
		}
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}
