package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/pborman/getopt"
	"github.com/starkandwayne/goutils/ansi"

	. "github.com/starkandwayne/shield/api"
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
	//Version gets overridden by lflags when building
	Version = ""
	options = Options{
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
	c = NewCommand().With(options)
)

func main() {
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

	/*
		   #### ##    ## ########  #######
			  ##  ###   ## ##       ##     ##
			  ##  ####  ## ##       ##     ##
			  ##  ## ## ## ######   ##     ##
			  ##  ##  #### ##       ##     ##
			  ##  ##   ### ##       ##     ##
			 #### ##    ## ##        #######
	*/
	c.HelpGroup("INFO:")
	c.Dispatch("help", "Get detailed help with a specific command", cliUsage)
	c.Alias("usage", "help")

	c.Dispatch("commands", "Show the list of available commands", cliCommands)

	c.Dispatch("flags", "Show the list of all command line flags", cliFlags)
	c.Alias("options", "flags")

	c.Dispatch("status", "Query the SHIELD backup server for its status and version info", cliStatus)
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
	c.HelpGroup("BACKENDS:")
	c.Dispatch("backends", "List configured SHIELD backends", cliListBackends)
	c.Alias("list backends", "backends")
	c.Alias("ls be", "backends")

	c.Dispatch("create-backend", "Create or modify a SHIELD backend", cliCreateBackend)
	c.Alias("create backend", "create-backend")
	c.Alias("c be", "create-backend")
	c.Alias("update backend", "create-backend")
	c.Alias("update-backend", "create-backend")
	c.Alias("edit-backend", "create-backend")
	c.Alias("edit backend", "create-backend")

	c.Dispatch("backend", "Select a particular backend for use", cliUseBackend)
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

	c.HelpGroup("TARGETS:")
	c.Dispatch("targets", "List available backup targets", cliListTargets)
	c.Alias("list targets", "targets")
	c.Alias("ls targets", "targets")

	c.Dispatch("target", "Print detailed information about a specific backup target", cliGetTarget)
	c.Alias("show target", "target")
	c.Alias("view target", "target")
	c.Alias("display target", "target")
	c.Alias("list target", "target")
	c.Alias("ls target", "target")

	c.Dispatch("create-target", "Create a new backup target", cliCreateTarget)
	c.Alias("create target", "create-target")
	c.Alias("new target", "create-target")
	c.Alias("create new target", "create-target")
	c.Alias("make target", "create-target")
	c.Alias("c t", "create-target")
	c.Alias("add target", "create-target")

	c.Dispatch("edit-target", "Modify an existing backup target", cliEditTarget)
	c.Alias("edit target", "edit-target")
	c.Alias("update target", "edit-target")

	c.Dispatch("delete-target", "Delete a backup target", cliDeleteTarget)
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

	c.HelpGroup("SCHEDULES:")
	c.Dispatch("schedules", "List available backup schedules", cliListSchedules)
	c.Alias("list schedules", "schedules")
	c.Alias("ls schedules", "schedules")

	c.Dispatch("schedule", "Print detailed information about a specific backup schedule", cliGetSchedule)
	c.Alias("show schedule", "schedule")
	c.Alias("view schedule", "schedule")
	c.Alias("display schedule", "schedule")
	c.Alias("list schedule", "schedule")
	c.Alias("ls schedule", "schedule")

	c.Dispatch("create-schedule", "Create a new backup schedule", cliCreateSchedule)
	c.Alias("create schedule", "create-schedule")
	c.Alias("new schedule", "create-schedule")
	c.Alias("create new schedule", "create-schedule")
	c.Alias("make schedule", "create-schedule")
	c.Alias("c s", "create-schedule")

	c.Dispatch("edit-schedule", "Modify an existing backup schedule", cliEditSchedule)
	c.Alias("edit schedule", "edit-schedule")
	c.Alias("update schedule", "edit-schedule")

	c.Dispatch("delete-schedule", "Delete a backup schedule", cliDeleteSchedule)
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

	c.HelpGroup("POLICIES:")
	c.Dispatch("policies", "List available retention policies", cliListPolicies)
	c.Alias("list retention policies", "policies")
	c.Alias("ls retention policies", "policies")
	c.Alias("list policies", "policies")
	c.Alias("ls policies", "policies")

	c.Dispatch("policy", "Print detailed information about a specific retention policy", cliGetPolicy)
	c.Alias("show retention policy", "policy")
	c.Alias("view retention policy", "policy")
	c.Alias("display retention policy", "policy")
	c.Alias("list retention policy", "policy")
	c.Alias("show policy", "policy")
	c.Alias("view policy", "policy")
	c.Alias("display policy", "policy")
	c.Alias("list policy", "policy")

	c.Dispatch("create-policy", "Create a new retention policy", cliCreatePolicy)
	c.Alias("create retention policy", "create-policy")
	c.Alias("new retention policy", "create-policy")
	c.Alias("create new retention policy", "create-policy")
	c.Alias("make retention policy", "create-policy")
	c.Alias("create policy", "create-policy")
	c.Alias("new policy", "create-policy")
	c.Alias("create new policy", "create-policy")
	c.Alias("make policy", "create-policy")
	c.Alias("c p", "create-policy")

	c.Dispatch("edit-policy", "Modify an existing retention policy", cliEditPolicy)
	c.Alias("edit retention policy", "edit-policy")
	c.Alias("update retention policy", "edit-policy")
	c.Alias("edit policy", "edit-policy")
	c.Alias("update policy", "edit-policy")

	c.Dispatch("delete-policy", "Delete a retention policy", cliDeletePolicy)
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

	c.HelpGroup("STORES:")
	c.Dispatch("stores", "List available archive stores", cliListStores)
	c.Alias("list stores", "stores")
	c.Alias("ls stores", "stores")

	c.Dispatch("store", "Print detailed information about a specific archive store", cliGetStore)
	c.Alias("show store", "store")
	c.Alias("view store", "store")
	c.Alias("display store", "store")
	c.Alias("list store", "store")
	c.Alias("ls store", "store")

	c.Dispatch("create-store", "Create a new archive store", cliCreateStore)
	c.Alias("create store", "create-store")
	c.Alias("new store", "create-store")
	c.Alias("create new store", "create-store")
	c.Alias("make store", "create-store")
	c.Alias("c st", "create-store")

	c.Dispatch("edit-store", "Modify an existing archive store", cliEditStore)
	c.Alias("edit store", "edit-store")
	c.Alias("update store", "edit-store")

	c.Dispatch("delete-store", "Delete an archive store", cliDeleteStore)
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

	c.HelpGroup("JOBS:")
	c.Dispatch("jobs", "List available backup jobs", cliListJobs)
	c.Alias("list jobs", "jobs")
	c.Alias("ls jobs", "jobs")
	c.Alias("ls j", "jobs")

	c.Dispatch("job", "Print detailed information about a specific backup job", cliGetJob)
	c.Alias("show job", "job")
	c.Alias("view job", "job")
	c.Alias("display job", "job")
	c.Alias("list job", "job")
	c.Alias("ls job", "job")

	c.Dispatch("create-job", "Create a new backup job", cliCreateJob)
	c.Alias("create job", "create-job")
	c.Alias("new job", "create-job")
	c.Alias("create new job", "create-job")
	c.Alias("make job", "create-job")
	c.Alias("c j", "create-job")

	c.Dispatch("edit-job", "Modify an existing backup job", cliEditJob)
	c.Alias("edit job", "edit-job")
	c.Alias("update job", "edit-job")

	c.Dispatch("delete-job", "Delete a backup job", cliDeleteJob)
	c.Alias("delete job", "delete-job")
	c.Alias("remove job", "delete-job")
	c.Alias("rm job", "delete-job")

	c.Dispatch("pause", "Pause a backup job", cliPauseJob)
	c.Alias("pause job", "pause")

	c.Dispatch("unpause", "Unpause a backup job", cliUnpauseJob)
	c.Alias("unpause job", "unpause")

	c.Dispatch("run", "Schedule an immediate run of a backup job", cliRunJob)
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

	c.HelpGroup("TASKS:")
	c.Dispatch("tasks", "List available tasks", cliListTasks)
	c.Alias("list tasks", "tasks")
	c.Alias("ls tasks", "tasks")

	c.Dispatch("task", "Print detailed information about a specific task", cliGetTask)
	c.Alias("show task", "task")
	c.Alias("view task", "task")
	c.Alias("display task", "task")
	c.Alias("list task", "task")
	c.Alias("ls task", "task")

	c.Dispatch("cancel-task", "Cancel a running or pending task", cliCancelTask)
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

	c.HelpGroup("ARCHIVES:")
	c.Dispatch("archives", "List available backup archives", cliListArchives)
	c.Alias("list archives", "archives")
	c.Alias("ls archives", "archives")

	c.Dispatch("archive", "Print detailed information about a backup archive", cliGetArchive)
	c.Alias("show archive", "archive")
	c.Alias("view archive", "archive")
	c.Alias("display archive", "archive")
	c.Alias("list archive", "archive")
	c.Alias("ls archive", "archive")

	c.Dispatch("restore", "Restore a backup archive", cliRestoreArchive)
	c.Alias("restore archive", "restore")
	c.Alias("restore-archive", "restore")

	c.Dispatch("delete-archive", "Delete a backup archive", cliDeleteArchive)
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

		loadBackend()
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
