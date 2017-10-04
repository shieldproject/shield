package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/pborman/getopt/v2"
	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/api"
	cmds "github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/access"
	"github.com/starkandwayne/shield/cmd/shield/commands/archives"
	"github.com/starkandwayne/shield/cmd/shield/commands/backends"
	"github.com/starkandwayne/shield/cmd/shield/commands/info"
	"github.com/starkandwayne/shield/cmd/shield/commands/jobs"
	"github.com/starkandwayne/shield/cmd/shield/commands/misc"
	"github.com/starkandwayne/shield/cmd/shield/commands/policies"
	"github.com/starkandwayne/shield/cmd/shield/commands/stores"
	"github.com/starkandwayne/shield/cmd/shield/commands/targets"
	"github.com/starkandwayne/shield/cmd/shield/commands/tasks"
	"github.com/starkandwayne/shield/cmd/shield/config"
	"github.com/starkandwayne/shield/cmd/shield/log"
)

//Version gets overridden by lflags when building
var Version = ""

func main() {
	cmds.Opts = &cmds.Options{
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
		CACert:            getopt.StringLong("ca-cert", 0, "", "Path to file to set as trusted root CA for requests"),

		Status:    getopt.StringLong("status", 'S', "", "Only show archives/tasks with the given status"),
		Target:    getopt.StringLong("target", 't', "", "Only show things for the target with this UUID"),
		Store:     getopt.StringLong("store", 's', "", "Only show things for the store with this UUID"),
		Retention: getopt.StringLong("policy", 'p', "", "Only show things for the retention policy with this UUID"),
		Plugin:    getopt.StringLong("plugin", 'P', "", "Only show things for the given target or store plugin"),
		After:     getopt.StringLong("after", 'A', "", "Only show archives that were taken after the given date, in YYYYMMDD format."),
		Before:    getopt.StringLong("before", 'B', "", "Only show archives that were taken before the given date, in YYYYMMDD format."),
		To:        getopt.StringLong("to", 0, "", "Restore the archive in question to a different target, specified by UUID"),
		Limit:     getopt.StringLong("limit", 0, "", "Display only the X most recent tasks or archives"),

		Full: getopt.BoolLong("full", 0, "Show all backend information when listing backends"),

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

	err := config.Load(*cmds.Opts.Config)
	if err != nil {
		ansi.Fprintf(os.Stderr, "\n@R{ERROR:} Could not parse %s: %s\n", *cmds.Opts.Config, err)
		os.Exit(1)
	}

	cmd, cmdname, args := cmds.ParseCommand(command...)
	log.DEBUG("Command: '%s'", cmdname)
	//Check if user gave a valid command
	if cmd == nil {
		ansi.Fprintf(os.Stderr, "@R{unrecognized command `%s'}\n", cmdname)

		re := regexp.MustCompile("schedule")
		if re.MatchString(cmdname) {
			warnScheduleDeprecation()
		}
		os.Exit(1)
	}

	currentBackend := config.Current()
	var curCACert string
	var shouldResetCACert bool
	// only check for backends + creds if we aren't manipulating backends/help
	if cmd != info.Usage && cmd.Group != cmds.BackendsGroup {
		if currentBackend == nil {
			ansi.Fprintf(os.Stderr, "@R{No backend targeted. Use `shield list backends` and `shield backend` to target one}\n")
			os.Exit(1)
		}

		if *cmds.Opts.SkipSSLValidation {
			os.Setenv("SHIELD_SKIP_SSL_VERIFY", "true")
			if *cmds.Opts.CACert != "" {
				ansi.Fprintf(os.Stderr, "@R{Can't skip validation with a specified CA cert}\n")
				os.Exit(1)
			}
		}

		//If overriding the ca-cert on an individual command, save the current
		//ca-cert for later, and load in the specified CA cert file
		if *cmds.Opts.CACert != "" {
			curCACert = currentBackend.CACert
			currentBackend.CACert, err = backends.ParseCACertFlag(*cmds.Opts.CACert)
			if err != nil {
				ansi.Fprintf(os.Stderr, "@R{Could not parse --ca-cert flag: %s}\n", err.Error())
				os.Exit(1)
			}
			shouldResetCACert = true
		}

		err = api.SetBackend(currentBackend)
		if err != nil {
			ansi.Fprintf(os.Stderr, "@R{Could not set current backend: %s}\n", err.Error())
			os.Exit(1)
		}

		cmds.Opts.APIVersion, err = apiVersion()
		if err != nil {
			ansi.Fprintf(os.Stderr, "@R{Could not contact backend: %s}\n", err.Error())
			os.Exit(1)
		}
		log.DEBUG("Using API Version %d", cmds.Opts.APIVersion)
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
		//Save the config changes if everything worked out
		if currentBackend != nil {
			if shouldResetCACert { //Reset CACert to configured if we overrode with flag
				currentBackend.CACert = curCACert
			}
			err = config.Commit(currentBackend)
			if err != nil {
				ansi.Fprintf(os.Stderr, "@R{%s}\n", err)
				os.Exit(1)
			}
		}
		err = config.Save()
		if err != nil {
			ansi.Fprintf(os.Stderr, "@R{Error saving config: %s}\n", err)
			os.Exit(1)
		}

		os.Exit(0)
	}
}

func addCommands() {
	cmds.Add("help", info.Usage).AKA("usage", "commands")
	cmds.Add("status", info.Status)

	cmds.Add("curl", misc.Curl)

	cmds.Add("backends", backends.List)
	cmds.Add("backend", backends.Use).AKA("use backend", "use-backend")
	cmds.Add("create-backend", backends.Create).AKA("create backend")
	cmds.Add("rename-backend", backends.Rename).AKA("rename backend")
	cmds.Add("delete-backend", backends.Delete).AKA("delete backend")

	cmds.Add("targets", targets.List)
	cmds.Add("target", targets.Get)
	cmds.Add("create-target", targets.Create).AKA("create target")
	cmds.Add("edit-target", targets.Edit).AKA("edit target")
	cmds.Add("delete-target", targets.Delete).AKA("delete target")

	cmds.Add("stores", stores.List)
	cmds.Add("store", stores.Get)
	cmds.Add("create-store", stores.Create).AKA("create store")
	cmds.Add("edit-store", stores.Edit).AKA("edit store")
	cmds.Add("delete-store", stores.Delete).AKA("delete store")

	cmds.Add("policies", policies.List).AKA("retention policies", "retention-policies")
	cmds.Add("policy", policies.Get).AKA("retention policy", "retention-policy")
	cmds.Add("create-policy", policies.Create).AKA("create retention policy", "create-retention-policy", "create policy")
	cmds.Add("edit-policy", policies.Edit).AKA("edit retention policy", "edit policy", "edit-retention-policy")
	cmds.Add("delete-policy", policies.Delete).AKA("delete retention policy", "delete policy", "delete-retention-policy")

	cmds.Add("jobs", jobs.List)
	cmds.Add("job", jobs.Get)
	cmds.Add("create-job", jobs.Create).AKA("create job")
	cmds.Add("edit-job", jobs.Edit).AKA("edit job")
	cmds.Add("delete-job", jobs.Delete).AKA("delete job")
	cmds.Add("pause", jobs.Pause).AKA("pause job", "pause-job")
	cmds.Add("unpause", jobs.Unpause).AKA("unpause job", "unpause-job")
	cmds.Add("run", jobs.Run).AKA("run job", "run-job")

	cmds.Add("archives", archives.List)
	cmds.Add("archive", archives.Get)
	cmds.Add("restore", archives.Restore).AKA("restore archive", "restore-archive")
	cmds.Add("delete-archive", archives.Delete).AKA("delete archive")

	cmds.Add("tasks", tasks.List)
	cmds.Add("task", tasks.Get)
	cmds.Add("cancel", tasks.Cancel).AKA("cancel-task", "cancel task")

	cmds.Add("unlock", access.Unlock).AKA("unseal")
	cmds.Add("init", access.Init).AKA("initialize")
	cmds.Add("rekey", access.Rekey).AKA("rekey-master")

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

func apiVersion() (int, error) {
	status, err := api.GetStatus()
	return status.APIVersion, err
}

func warnScheduleDeprecation() {
	output := `
As of SHIELD v8, schedules are no longer objects in the job flow, and have been
reduced to simply the timespec string (e.g. daily 4am), which is now attached
directly to a job. Therefore, schedule commands have been removed from the CLI.
The CLI is still backward-compatible, and when contacting SHIELD deployments
which still expect a SHIELD, it will manage schedules for you transparently.`
	ansi.Fprintf(os.Stderr, "@R{%s}\n", output)
}
