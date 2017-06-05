package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pborman/getopt/v2"
	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/api"
	cmds "github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/archives"
	"github.com/starkandwayne/shield/cmd/shield/commands/backends"
	"github.com/starkandwayne/shield/cmd/shield/commands/info"
	"github.com/starkandwayne/shield/cmd/shield/commands/jobs"
	"github.com/starkandwayne/shield/cmd/shield/commands/policies"
	"github.com/starkandwayne/shield/cmd/shield/commands/schedules"
	"github.com/starkandwayne/shield/cmd/shield/commands/stores"
	"github.com/starkandwayne/shield/cmd/shield/commands/targets"
	"github.com/starkandwayne/shield/cmd/shield/commands/tasks"
	"github.com/starkandwayne/shield/cmd/shield/log"
)

//Version gets overridden by lflags when building
var Version = ""

func main() {
	cmds.Opts = &cmds.Options{
		Shield:   getopt.StringLong("shield", 'H', "", "DEPRECATED - Previously required to point to a SHIELD backend to talk to. Now used to auto-vivify ~/.shield_config if necessary"),
		Used:     getopt.BoolLong("used", 0, "Only show things that are in-use by something else"),
		Unused:   getopt.BoolLong("unused", 0, "Only show things that are not used by something else"),
		Paused:   getopt.BoolLong("paused", 0, "Only show jobs that are paused"),
		Unpaused: getopt.BoolLong("unpaused", 0, "Only show jobs that are unpaused"),
		All:      getopt.BoolLong("all", 'a', "Show all the things"),

		Debug:             getopt.BoolLong("debug", 'D', "Enable debugging"),
		Trace:             getopt.BoolLong("trace", 'T', "Enable trace mode"),
		Raw:               getopt.BoolLong("raw", 0, "Operate in RAW mode, reading and writing only JSON"),
		ShowUUID:          getopt.BoolLong("uuid", 0, "Return UUID"),
		UpdateIfExists:    getopt.BoolLong("update-if-exists", 0, "Create will update record if another exists with same name"),
		Fuzzy:             getopt.BoolLong("fuzzy", 0, "In RAW mode, perform fuzzy (inexact) searching"),
		SkipSSLValidation: getopt.BoolLong("skip-ssl-validation", 'k', "Disable SSL Certificate Validation"),

		Status:    getopt.StringLong("status", 'S', "", "Only show archives/tasks with the given status"),
		Target:    getopt.StringLong("target", 't', "", "Only show things for the target with this UUID"),
		Store:     getopt.StringLong("store", 's', "", "Only show things for the store with this UUID"),
		Retention: getopt.StringLong("policy", 'p', "", "Only show things for the retention policy with this UUID"),
		Plugin:    getopt.StringLong("plugin", 'P', "", "Only show things for the given target or store plugin"),
		After:     getopt.StringLong("after", 'A', "", "Only show archives that were taken after the given date, in YYYYMMDD format."),
		Before:    getopt.StringLong("before", 'B', "", "Only show archives that were taken before the given date, in YYYYMMDD format."),
		To:        getopt.StringLong("to", 0, "", "Restore the archive in question to a different target, specified by UUID"),
		Limit:     getopt.StringLong("limit", 0, "", "Display only the X most recent tasks or archives"),

		Config:  getopt.StringLong("config", 'c', os.Getenv("HOME")+"/.shield_config", "Overrides ~/.shield_config as the SHIELD config file"),
		Version: getopt.BoolLong("version", 'v', "Display the SHIELD version"),
		Help:    getopt.BoolLong("help", 'h'),
	}

	var command []string
	var cmdLine = getopt.CommandLine
	args := os.Args
	for {
		err := cmdLine.Getopt(args, nil)
		if err != nil {
			ansi.Fprintf(os.Stderr, "@R{%s}\n", err.Error())
			os.Exit(1)
		}
		if cmdLine.NArgs() == 0 {
			break
		}
		command = append(command, cmdLine.Arg(0))
		args = cmdLine.Args()
	}

	log.ToggleDebug(*cmds.Opts.Debug)
	log.ToggleTrace(*cmds.Opts.Trace)

	log.DEBUG("shield cli starting up")

	if *cmds.Opts.SkipSSLValidation {
		os.Setenv("SHIELD_SKIP_SSL_VERIFY", "true")
	}

	if *cmds.Opts.Version {
		if Version == "" {
			fmt.Println("shield cli (development)")
		} else {
			fmt.Printf("shield cli v%s\n", Version)
		}
		os.Exit(0)
	}

	if *cmds.Opts.Help {
		command = append([]string{"help"}, command...)
	}

	addCommands()
	addGlobalFlags()

	err := api.LoadConfig(*cmds.Opts.Config)
	if err != nil {
		ansi.Fprintf(os.Stderr, "\n@R{ERROR:} Could not parse %s: %s\n", *cmds.Opts.Config, err)
		os.Exit(1)
	}

	cmd, cmdname, args := cmds.ParseCommand(command...)
	log.DEBUG("Command: '%s'", cmdname)
	//Check if user gave a valid command
	if cmd == nil {
		ansi.Fprintf(os.Stderr, "@R{unrecognized command `%s'}\n", cmdname)
		os.Exit(1)
	}
	cmds.MaybeWarnDeprecation(cmdname, cmd)

	// only check for backends + creds if we aren't manipulating backends/help
	if cmd != info.Usage && cmd != backends.List && cmd != backends.Use && cmd != backends.Create {
		if *cmds.Opts.Shield != "" || os.Getenv("SHIELD_API") != "" {
			ansi.Fprintf(os.Stderr, "@Y{WARNING: -H, --host, and the SHIELD_API environment variable have been deprecated and will be removed in a later release.} Use `shield backend` instead\n")
		}

		backends.Load()
	}

	if err := cmd.Run(args...); err != nil {
		if *cmds.Opts.Raw {
			j, err := json.Marshal(map[string]string{"error": err.Error()})
			if err != nil {
				panic("Couldn't parse error json")
			}
			fmt.Println(string(j))
		} else {
			ansi.Fprintf(os.Stderr, "@R{%s}\n", err)
		}
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}

func addCommands() {
	cmds.Add("help", info.Usage).AKA("usage", "commands")
	cmds.Add("status", info.Status).AKA("stat")

	cmds.Add("backends", backends.List).AKA("list backends", "ls be")
	cmds.Add("backend", backends.Use).AKA("use backend", "use-backend")
	cmds.Add("create-backend", backends.Create).AKA("create backend", "c be", "update backend")

	cmds.Add("targets", targets.List).AKA("list targets", "ls targets")
	cmds.Add("target", targets.Get).AKA("show target", "view target", "display target", "list target", "ls target")
	cmds.Add("create-target", targets.Create).AKA("create target", "new target", "create new target", "make target", "c t", "add target")
	cmds.Add("edit-target", targets.Edit).AKA("edit target", "update target")
	cmds.Add("delete-target", targets.Delete).AKA("delete target", "remove target", "rm target")

	cmds.Add("stores", stores.List).AKA("list stores, ls stores")
	cmds.Add("store", stores.Get).AKA("show store", "view store", "display store", "list store", "ls store")
	cmds.Add("create-store", stores.Create).AKA("create store", "new store", "create new store", "make store", "c st")
	cmds.Add("edit-store", stores.Edit).AKA("edit store", "update store")
	cmds.Add("delete-store", stores.Delete).AKA("delete store", "remove store", "rm store")

	cmds.Add("schedules", schedules.List).AKA("list schedules", "ls schedules")
	cmds.Add("schedule", schedules.Get).AKA("show schedule", "view schedule", "display schedule", "list schedule", "ls schedule")
	cmds.Add("create-schedule", schedules.Create).AKA("create schedule", "new schedule", "create new schedule", "make schedule", "c s")
	cmds.Add("edit-schedule", schedules.Edit).AKA("edit schedule", "update schedule")
	cmds.Add("delete-schedule", schedules.Delete).AKA("delete schedule", "remove schedule", "rm schedule")

	cmds.Add("policies", policies.List).AKA("list retention policies", "ls retention policies", "list policies", "ls policies")
	cmds.Add("policy", policies.Get).AKA("show retention policy", "view retention policy", "display retention policy",
		"list retention policy", "show policy", "view policy", "display policy", "list policy")
	cmds.Add("create-policy", policies.Create).AKA("create retention policy", "new retention policy", "create new retention policy",
		"make retention policy", "create policy", "new policy", "create new policy", "make policy")
	cmds.Add("edit-policy", policies.Edit).AKA("edit retention policy", "update retention policy", "edit policy", "update policy")
	cmds.Add("delete-policy", policies.Delete).AKA("delete retention policy", "remove retention policy", "rm retention policy",
		"delete policy", "remove policy", "rm policy")

	cmds.Add("jobs", jobs.List).AKA("list jobs", "ls jobs", "ls j")
	cmds.Add("job", jobs.Get).AKA("show job", "view job", "display job", "list job", "ls job")
	cmds.Add("create-job", jobs.Create).AKA("create job", "new job", "create new job", "make job", "c j")
	cmds.Add("edit-job", jobs.Edit).AKA("edit job", "update job")
	cmds.Add("delete-job", jobs.Delete).AKA("delete job", "remove job", "rm job")
	cmds.Add("pause", jobs.Pause).AKA("pause job")
	cmds.Add("unpause", jobs.Unpause).AKA("unpause job")
	cmds.Add("run", jobs.Run).AKA("run job")

	cmds.Add("archives", archives.List).AKA("list archives", "ls archives")
	cmds.Add("archive", archives.Get).AKA("show archive", "view archive", "display archive", "list archive", "ls archive")
	cmds.Add("restore", archives.Restore).AKA("restore archive", "restore-archive")
	cmds.Add("delete-archive", archives.Delete).AKA("delete archive", "remove archive", "rm archive")

	cmds.Add("tasks", tasks.List).AKA("list tasks", "ls tasks")
	cmds.Add("task", tasks.Get).AKA("show task", "view task", "display task", "list task", "ls task")
	cmds.Add("cancel", tasks.Cancel).AKA("cancel-task", "cancel task", "stop task")
}

func addGlobalFlags() {
	cmds.GlobalFlags = []cmds.FlagInfo{
		cmds.FlagInfo{
			Name: "debug", Short: 'D',
			Desc: "Enable the output of debug output",
		},
		cmds.FlagInfo{
			Name: "trace", Short: 'T',
			Desc: "Enable the output of verbose trace output",
		},
		cmds.FlagInfo{
			Name: "skip-ssl-validation", Short: 'k',
			Desc: "Disable SSL certificate validation",
		},
		cmds.FlagInfo{
			Name: "raw",
			Desc: "Takes any input and gives any output as a JSON object",
		},
	}
}
