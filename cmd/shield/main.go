package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pborman/getopt"
	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	_ "github.com/starkandwayne/shield/cmd/shield/commands/archives"
	"github.com/starkandwayne/shield/cmd/shield/commands/backends"
	_ "github.com/starkandwayne/shield/cmd/shield/commands/info"
	_ "github.com/starkandwayne/shield/cmd/shield/commands/jobs"
	_ "github.com/starkandwayne/shield/cmd/shield/commands/policies"
	_ "github.com/starkandwayne/shield/cmd/shield/commands/schedules"
	_ "github.com/starkandwayne/shield/cmd/shield/commands/stores"
	_ "github.com/starkandwayne/shield/cmd/shield/commands/targets"
	_ "github.com/starkandwayne/shield/cmd/shield/commands/tasks"
	"github.com/starkandwayne/shield/cmd/shield/log"
)

var (
	debug = false
	//Version gets overridden by lflags when building
	Version = ""
)

func main() {
	commands.Opts = &commands.Options{
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

	var command []string
	var cmdLine = getopt.CommandLine
	args := os.Args
	for {
		cmdLine.Parse(args)
		if cmdLine.NArgs() == 0 {
			break
		}
		command = append(command, cmdLine.Arg(0))
		args = cmdLine.Args()
	}

	log.ToggleDebug(*commands.Opts.Debug)
	log.ToggleTrace(*commands.Opts.Trace)

	log.DEBUG("shield cli starting up")

	if *commands.Opts.SkipSSLValidation {
		os.Setenv("SHIELD_SKIP_SSL_VERIFY", "true")
	}

	if *commands.Opts.Version {
		if Version == "" {
			fmt.Println("shield cli (development)")
		} else {
			fmt.Printf("shield cli v%s\n", Version)
		}
		os.Exit(0)
	}

	commands.AddGlobalFlag(commands.FlagInfo{
		Name: "debug", Short: 'D',
		Desc: "Enable the output of debug output",
	})
	commands.AddGlobalFlag(commands.FlagInfo{
		Name: "trace", Short: 'T',
		Desc: "Enable the output of verbose trace output",
	})
	commands.AddGlobalFlag(commands.FlagInfo{
		Name: "skip-ssl-validation", Short: 'k',
		Desc: "Disable SSL certificate validation",
	})
	commands.AddGlobalFlag(commands.FlagInfo{
		Name: "raw",
		Desc: "Takes any input and gives any output as a JSON object",
	})

	err := api.LoadConfig(*commands.Opts.Config)
	if err != nil {
		ansi.Fprintf(os.Stderr, "\n@R{ERROR:} Could not parse %s: %s\n", *commands.Opts.Config, err)
		os.Exit(1)
	}

	cmd, cmdname, args := commands.ParseCommand(command...)
	log.DEBUG("Command: '%s'", cmdname)
	//Check if user gave a valid command
	if cmd == nil {
		ansi.Fprintf(os.Stderr, "@R{unrecognized command `%s'}\n", cmdname)
		os.Exit(1)
	}
	commands.MaybeWarnDeprecation(cmdname, cmd)

	// only check for backends + creds if we aren't manipulating backends/help
	helpCmd, _, _ := commands.ParseCommand("help")
	backendsCmd, _, _ := commands.ParseCommand("backends")
	backendCmd, _, _ := commands.ParseCommand("backend")
	cBackendCmd, _, _ := commands.ParseCommand("create-backend")
	if cmd != helpCmd && cmd != backendsCmd && cmd != backendCmd && cmd != cBackendCmd {
		if *commands.Opts.Shield != "" || os.Getenv("SHIELD_API") != "" {
			ansi.Fprintf(os.Stderr, "@Y{WARNING: -H, --host, and the SHIELD_API environment variable have been deprecated and will be removed in a later release.} Use `shield backend` instead\n")
		}

		backends.Load()
	}

	if err := cmd.Run(args...); err != nil {
		if *commands.Opts.Raw {
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
