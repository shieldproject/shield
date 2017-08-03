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
	"github.com/starkandwayne/shield/api"
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
	opts    = Options{
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
)

func main() {
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

	if len(command) == 0 {
		command = []string{"help"}
	}

	debug = *opts.Debug
	DEBUG("shield cli starting up")

	if *opts.Trace {
		DEBUG("enabling TRACE output")
		os.Setenv("SHIELD_TRACE", "1")
	}

	if *opts.SkipSSLValidation {
		os.Setenv("SHIELD_SKIP_SSL_VERIFY", "true")
	}

	if *opts.Version {
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
	dispatch.HelpGroup("INFO:")

	help := dispatch.Register("help", cliUsage).Aliases("usage")
	help.Summarize("Get detailed help with a specific command")
	help.Help(HelpInfo{
		Message: ansi.Sprintf("@R{This is getting a bit too meta, don't you think?}"),
	})

	commands := dispatch.Register("commands", cliCommands)
	commands.Summarize("Show the list of available commands")

	flags := dispatch.Register("flags", cliFlags).Aliases("options")
	flags.Summarize("Show the list of all command line flags")

	status := dispatch.Register("status", cliStatus).Aliases("stat")
	status.Summarize("Query the SHIELD backup server for its status and version info")
	status.Help(HelpInfo{
		JSONOutput: fmt.Sprintf(`{"name":"MyShield","version":"%s"}`, Version),
	})

	/*
	   ########     ###     ######  ##    ## ######## ##    ## ########   ######
	   ##     ##   ## ##   ##    ## ##   ##  ##       ###   ## ##     ## ##    ##
	   ##     ##  ##   ##  ##       ##  ##   ##       ####  ## ##     ## ##
	   ########  ##     ## ##       #####    ######   ## ## ## ##     ##  ######
	   ##     ## ######### ##       ##  ##   ##       ##  #### ##     ##       ##
	   ##     ## ##     ## ##    ## ##   ##  ##       ##   ### ##     ## ##    ##
	   ########  ##     ##  ######  ##    ## ######## ##    ## ########   ######
	*/
	dispatch.HelpGroup("BACKENDS:")

	backends := dispatch.Register("backends", cliListBackends).Aliases("list backends", "ls be")
	backends.Summarize("List configured SHIELD backends")
	backends.Help(HelpInfo{
		JSONOutput: `[{
			"name":"mybackend",
			"uri":"https://10.244.2.2:443"
		}]`,
	})

	cbackend := dispatch.Register("create-backend", cliCreateBackend)
	cbackend.Aliases("create backend", "c be", "update backend", "update-backend", "edit-backend", "edit backend")
	cbackend.Summarize("Create or modify a SHIELD backend")
	cbackend.Help(HelpInfo{
		Flags: []FlagInfo{
			FlagInfo{
				name: "name", mandatory: true, positional: true,
				desc: `The name of the new backend`,
			},
			FlagInfo{
				name: "uri", mandatory: true, positional: true,
				desc: `The address at which the new backend can be found`,
			},
		},
	})

	backend := dispatch.Register("backend", cliUseBackend).Aliases("use backend", "use-backend")
	backend.Summarize("Select a particular backend for use")
	backend.Help(HelpInfo{
		Flags: []FlagInfo{
			FlagInfo{
				name: "name", mandatory: true, positional: true,
				desc: "The name of the backend to target",
			},
		},
	})

	/*
	   ########    ###    ########   ######   ######## ########
	      ##      ## ##   ##     ## ##    ##  ##          ##
	      ##     ##   ##  ##     ## ##        ##          ##
	      ##    ##     ## ########  ##   #### ######      ##
	      ##    ######### ##   ##   ##    ##  ##          ##
	      ##    ##     ## ##    ##  ##    ##  ##          ##
	      ##    ##     ## ##     ##  ######   ########    ##
	*/

	dispatch.HelpGroup("TARGETS:")
	targets := dispatch.Register("targets", cliListTargets).Aliases("list targets", "ls targets")
	targets.Summarize("List available backup targets")
	targets.Help(HelpInfo{
		Flags: []FlagInfo{
			FlagInfo{
				name: "plugin", short: 'P', valued: true,
				desc: "Only show targets using the named target plugin",
			},
			UsedFlag,
			UnusedFlag,
			FuzzyFlag,
		},
		JSONOutput: `[{
				"uuid":"8add3e57-95cd-4ec0-9144-4cd5c50cd392",
				"name":"SampleTarget",
				"summary":"A Sample Target",
				"plugin":"postgres",
				"endpoint":"{\"endpoint\":\"127.0.0.1:5432\"}",
				"agent":"127.0.0.1:1234"
			}]`,
	})

	target := dispatch.Register("target", cliGetTarget)
	target.Aliases("show target", "view target", "display target", "list target", "ls target")
	target.Summarize("Print detailed information about a specific backup target")
	target.Help(HelpInfo{
		Flags: []FlagInfo{TargetNameFlag},
		JSONOutput: `{
			"uuid":"8add3e57-95cd-4ec0-9144-4cd5c50cd392",
			"name":"SampleTarget",
			"summary":"A Sample Target",
			"plugin":"postgres",
			"endpoint":"{\"endpoint\":\"127.0.0.1:5432\"}",
			"agent":"127.0.0.1:1234"
		}`,
	})

	ctarget := dispatch.Register("create-target", cliCreateTarget)
	ctarget.Aliases("create target", "new target", "create new target", "make target", "c t", "add target")
	ctarget.Summarize("Create a new backup target")
	ctarget.Help(HelpInfo{
		JSONInput: `{
			"agent":"127.0.0.1:1234",
			"endpoint":"{\"endpoint\":\"schmendpoint\"}",
			"name":"TestTarget",
			"plugin":"postgres",
			"summary":"A Test Target"
		}`,
		JSONOutput: `{
			"uuid":"77398f3e-2a31-4f20-b3f7-49d3f0998712",
			"name":"TestTarget",
			"summary":"A Test Target",
			"plugin":"postgres",
			"endpoint":"{\"endpoint\":\"schmendpoint\"}",
			"agent":"127.0.0.1:1234"
		}`,
	})

	etarget := dispatch.Register("edit-target", cliEditTarget).Aliases("edit target", "update target")
	etarget.Summarize("Modify an existing backup target")
	etarget.Help(HelpInfo{
		Message: "Modify an existing backup target. The UUID of the target will remain the same after modification.",
		Flags:   []FlagInfo{TargetNameFlag},
		JSONInput: `{
			"agent":"127.0.0.1:1234",
			"endpoint":"{\"endpoint\":\"newschmendpoint\"}",
			"name":"NewTargetName",
			"plugin":"postgres",
			"summary":"Some Target"
		}`,
		JSONOutput: `{
			"uuid":"8add3e57-95cd-4ec0-9144-4cd5c50cd392",
			"name":"SomeTarget",
			"summary":"Just this target, you know?",
			"plugin":"postgres",
			"endpoint":"{\"endpoint\":\"schmendpoint\"}",
			"agent":"127.0.0.1:1234"
		}`,
	})

	dtarget := dispatch.Register("delete-target", cliDeleteTarget).Aliases("delete target", "remove target", "rm target")
	dtarget.Summarize("Delete a backup target")
	dtarget.Help(HelpInfo{
		Flags:      []FlagInfo{TargetNameFlag},
		JSONOutput: `{"ok":"Deleted target"}`,
	})

	/*
	    ######   ######  ##     ## ######## ########  ##     ## ##       ########
	   ##    ## ##    ## ##     ## ##       ##     ## ##     ## ##       ##
	   ##       ##       ##     ## ##       ##     ## ##     ## ##       ##
	    ######  ##       ######### ######   ##     ## ##     ## ##       ######
	         ## ##       ##     ## ##       ##     ## ##     ## ##       ##
	   ##    ## ##    ## ##     ## ##       ##     ## ##     ## ##       ##
	    ######   ######  ##     ## ######## ########   #######  ######## ########
	*/

	dispatch.HelpGroup("SCHEDULES:")

	schedules := dispatch.Register("schedules", cliListSchedules).Aliases("list schedules", "ls schedules")
	schedules.Summarize("List available backup schedules")
	schedules.Help(HelpInfo{
		Flags: []FlagInfo{UsedFlag, UnusedFlag, FuzzyFlag},
		JSONOutput: `[{
			"uuid":"86ff3fec-76c5-48c4-880d-c37563033613",
			"name":"TestSched",
			"summary":"A Test Schedule",
			"when":"daily 4am"
		}]`,
	})

	schedule := dispatch.Register("schedule", cliGetSchedule)
	schedule.Aliases("show schedule", "view schedule", "display schedule", "list schedule", "ls schedule")
	schedule.Summarize("Print detailed information about a specific backup schedule")
	schedule.Help(HelpInfo{
		Flags: []FlagInfo{ScheduleNameFlag},
		JSONOutput: `{
			"uuid":"9a58a3fa-7457-431c-b094-e201b42b5c7b",
			"name":"TestSched",
			"summary":"A Test Schedule",
			"when":"daily 4am"
		}`,
	})

	cSchedule := dispatch.Register("create-schedule", cliCreateSchedule)
	cSchedule.Summarize("Create a new backup schedule")
	cSchedule.Aliases("create schedule", "new schedule", "create new schedule", "make schedule", "c s")
	cSchedule.Help(HelpInfo{
		JSONInput: `{
			"name":"TestSched",
			"summary":"A Test Schedule",
			"when":"daily 4am"
		}`,
		JSONOutput: `{
			"uuid":"9a58a3fa-7457-431c-b094-e201b42b5c7b",
			"name":"TestSched",
			"summary":"A Test Schedule",
			"when":"daily 4am"
		}`,
	})

	eSchedule := dispatch.Register("edit-schedule", cliEditSchedule).Aliases("edit schedule", "update schedule")
	eSchedule.Summarize("Modify an existing backup schedule")
	eSchedule.Help(HelpInfo{
		Flags: []FlagInfo{ScheduleNameFlag},
		JSONInput: `{
			"name":"AnotherSched",
			"summary":"A Test Schedule",
			"when":"daily 4am"
		}`,
		JSONOutput: `{
			"uuid":"9a58a3fa-7457-431c-b094-e201b42b5c7b",
			"name":"AnotherSched",
			"summary":"A Test Schedule",
			"when":"daily 4am"
		}`,
	})

	dSchedule := dispatch.Register("delete-schedule", cliDeleteSchedule)
	dSchedule.Summarize("Delete a backup schedule")
	dSchedule.Aliases("delete schedule", "remove schedule", "rm schedule")
	dSchedule.Help(HelpInfo{
		Flags:      []FlagInfo{ScheduleNameFlag},
		JSONOutput: `{"ok":"Deleted schedule"}`,
	})

	/*
	   ########  ######## ######## ######## ##    ## ######## ####  #######  ##    ##
	   ##     ## ##          ##    ##       ###   ##    ##     ##  ##     ## ###   ##
	   ##     ## ##          ##    ##       ####  ##    ##     ##  ##     ## ####  ##
	   ########  ######      ##    ######   ## ## ##    ##     ##  ##     ## ## ## ##
	   ##   ##   ##          ##    ##       ##  ####    ##     ##  ##     ## ##  ####
	   ##    ##  ##          ##    ##       ##   ###    ##     ##  ##     ## ##   ###
	   ##     ## ########    ##    ######## ##    ##    ##    ####  #######  ##    ##
	*/

	dispatch.HelpGroup("POLICIES:")
	policies := dispatch.Register("policies", cliListPolicies)
	policies.Summarize("List available retention policies")
	policies.Aliases("list retention policies", "ls retention policies", "list policies", "ls policies")
	policies.Help(HelpInfo{
		Flags: []FlagInfo{UnusedFlag, UsedFlag, FuzzyFlag},
		JSONOutput: `[{
			"uuid":"8c6f894f-9c27-475f-ad5a-8c0db37926ec",
			"name":"apolicy",
			"summary":"a policy",
			"expires":5616000
		}]`,
	})

	policy := dispatch.Register("policy", cliGetPolicy)
	policy.Summarize("Print detailed information about a specific retention policy")
	policy.Aliases("show retention policy", "view retention policy", "display retention policy", "list retention policy")
	policy.Aliases("show policy", "view policy", "display policy", "list policy")
	policy.Help(HelpInfo{
		Flags: []FlagInfo{PolicyNameFlag},
		JSONOutput: `{
			"uuid":"8c6f894f-9c27-475f-ad5a-8c0db37926ec",
			"name":"apolicy",
			"summary":"a policy",
			"expires":5616000
		}`,
	})

	cPolicy := dispatch.Register("create-policy", cliCreatePolicy)
	cPolicy.Summarize("Create a new retention policy")
	cPolicy.Aliases("create retention policy", "new retention policy", "create new retention policy", "make retention policy")
	cPolicy.Aliases("create policy", "new policy", "create new policy", "make policy")
	cPolicy.Help(HelpInfo{
		JSONInput: `{
			"expires":31536000,
			"name":"TestPolicy",
			"summary":"A Test Policy"
		}`,
		JSONOutput: `{
			"uuid":"18a446c4-c068-4c09-886c-cb77b6a85274",
			"name":"TestPolicy",
			"summary":"A Test Policy",
			"expires":31536000
		}`,
	})

	ePolicy := dispatch.Register("edit-policy", cliEditPolicy)
	ePolicy.Summarize("Modify an existing retention policy")
	ePolicy.Aliases("edit retention policy", "update retention policy", "edit policy", "update policy")
	ePolicy.Help(HelpInfo{
		Flags: []FlagInfo{PolicyNameFlag},
		JSONInput: `{
			"expires":31536000,
			"name":"AnotherPolicy",
			"summary":"A Test Policy"
		}`,
		JSONOutput: `{
			"uuid":"18a446c4-c068-4c09-886c-cb77b6a85274",
			"name":"AnotherPolicy",
			"summary":"A Test Policy",
			"expires":31536000
		}`,
	})

	dPolicy := dispatch.Register("delete-policy", cliDeletePolicy)
	dPolicy.Summarize("Delete a retention policy")
	dPolicy.Aliases("delete retention policy", "remove retention policy", "rm retention policy")
	dPolicy.Aliases("delete policy", "remove policy", "rm policy")
	dPolicy.Help(HelpInfo{
		Flags:      []FlagInfo{PolicyNameFlag},
		JSONOutput: `{"ok":"Deleted policy"}`,
	})

	/*
	    ######  ########  #######  ########  ########
	   ##    ##    ##    ##     ## ##     ## ##
	   ##          ##    ##     ## ##     ## ##
	    ######     ##    ##     ## ########  ######
	         ##    ##    ##     ## ##   ##   ##
	   ##    ##    ##    ##     ## ##    ##  ##
	    ######     ##     #######  ##     ## ########
	*/

	dispatch.HelpGroup("STORES:")
	stores := dispatch.Register("stores", cliListStores).Aliases("list stores, ls stores")
	stores.Summarize("List available archive stores")
	stores.Help(HelpInfo{
		Flags: []FlagInfo{UsedFlag, UnusedFlag, FuzzyFlag},
		JSONOutput: `[{
			"uuid":"6e83bfb7-7ae1-4f0f-88a8-84f0fe4bae20",
			"name":"test store",
			"summary":"a test store named \"test store\"",
			"plugin":"s3",
			"endpoint":"{ \"endpoint\": \"doesntmatter\" }"
		}]`,
	})

	store := dispatch.Register("store", cliGetStore)
	store.Summarize("Print detailed information about a specific archive store")
	store.Aliases("show store", "view store", "display store", "list store", "ls store")
	store.Help(HelpInfo{
		Flags: []FlagInfo{StoreNameFlag},
		JSONOutput: `{
			"uuid":"6e83bfb7-7ae1-4f0f-88a8-84f0fe4bae20",
			"name":"test store",
			"summary":"a test store named \"test store\"",
			"plugin":"s3",
			"endpoint":"{ \"endpoint\": \"doesntmatter\" }"
		}`,
	})

	cStore := dispatch.Register("create-store", cliCreateStore)
	cStore.Summarize("Create a new archive store")
	cStore.Aliases("create store", "new store", "create new store", "make store", "c st")
	cStore.Help(HelpInfo{
		JSONInput: `{
			"endpoint":"{\"endpoint\":\"schmendpoint\"}",
			"name":"TestStore",
			"plugin":"s3",
			"summary":"A Test Store"
		}`,
		JSONOutput: `{
			"uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf",
			"name":"TestStore",
			"summary":"A Test Store",
			"plugin":"s3",
			"endpoint":"{\"endpoint\":\"schmendpoint\"}"
		}`,
	})

	eStore := dispatch.Register("edit-store", cliEditStore).Aliases("edit store", "update store")
	eStore.Summarize("Modify an existing archive store")
	eStore.Help(HelpInfo{
		Flags: []FlagInfo{StoreNameFlag},
		JSONInput: `{
			"endpoint":"{\"endpoint\":\"schmendpoint\"}",
			"name":"AnotherStore",
			"plugin":"s3",
			"summary":"A Test Store"
		}`,
		JSONOutput: `{
			"uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf",
			"name":"AnotherStore",
			"summary":"A Test Store",
			"plugin":"s3",
			"endpoint":"{\"endpoint\":\"schmendpoint\"}"
		}`,
	})

	dStore := dispatch.Register("delete-store", cliDeleteStore)
	dStore.Summarize("Delete an archive store")
	dStore.Aliases("delete store", "remove store", "rm store")
	dStore.Help(HelpInfo{
		Flags:      []FlagInfo{StoreNameFlag},
		JSONOutput: `{"ok":"Deleted store"}`,
	})

	/*
	         ##  #######  ########
	         ## ##     ## ##     ##
	         ## ##     ## ##     ##
	         ## ##     ## ########
	   ##    ## ##     ## ##     ##
	   ##    ## ##     ## ##     ##
	    ######   #######  ########
	*/

	dispatch.HelpGroup("JOBS:")
	jobs := dispatch.Register("jobs", cliListJobs)
	jobs.Summarize("List available backup jobs")
	jobs.Aliases("list jobs", "ls jobs", "ls j")
	jobs.Help(HelpInfo{
		Flags: []FlagInfo{
			{
				name: "target", short: 't', valued: true,
				desc: "Show only jobs using the specified target",
			},
			{
				name: "store", short: 's', valued: true,
				desc: "Show only jobs using the specified store",
			},
			{
				name: "schedule", short: 'w', valued: true,
				desc: "Show only jobs using the specified schedule",
			},
			{
				name: "policy", short: 'p', valued: true,
				desc: "Show only jobs using the specified retention policy",
			},
			{name: "paused", desc: "Show only jobs which are paused"},
			{name: "unpaused", desc: "Show only jobs which are unpaused"},
			FuzzyFlag,
		},
		JSONOutput: `[{
			"uuid":"f6623a6f-8dce-46b2-a293-5525bc3a3588",
			"name":"TestJob",
			"summary":"A Test Job",
			"retention_name":"AnotherPolicy",
			"retention_uuid":"18a446c4-c068-4c09-886c-cb77b6a85274",
			"expiry":31536000,
			"schedule_name":"AnotherSched",
			"schedule_uuid":"9a58a3fa-7457-431c-b094-e201b42b5c7b",
			"schedule_when":"daily 4am",
			"paused":true,
			"store_uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf",
			"store_name":"AnotherStore",
			"store_plugin":"s3",
			"store_endpoint":"{\"endpoint\":\"schmendpoint\"}",
			"target_uuid":"84751f04-2be2-428d-b6a3-2022c63bf6ee",
			"target_name":"TestTarget",
			"target_plugin":"postgres",
			"target_endpoint":"{\"endpoint\":\"schmendpoint\"}",
			"agent":"127.0.0.1:1234"
		}]`,
	})

	job := dispatch.Register("job", cliGetJob)
	job.Summarize("Print detailed information about a specific backup job")
	job.Aliases("show job", "view job", "display job", "list job", "ls job")
	job.Help(HelpInfo{
		Flags: []FlagInfo{JobNameFlag},
		JSONOutput: `{
			"uuid":"f6623a6f-8dce-46b2-a293-5525bc3a3588",
			"name":"TestJob",
			"summary":"A Test Job",
			"retention_name":"AnotherPolicy",
			"retention_uuid":"18a446c4-c068-4c09-886c-cb77b6a85274",
			"expiry":31536000,
			"schedule_name":"AnotherSched",
			"schedule_uuid":"9a58a3fa-7457-431c-b094-e201b42b5c7b",
			"schedule_when":"daily 4am",
			"paused":true,
			"store_uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf",
			"store_name":"AnotherStore",
			"store_plugin":"s3",
			"store_endpoint":"{\"endpoint\":\"schmendpoint\"}",
			"target_uuid":"84751f04-2be2-428d-b6a3-2022c63bf6ee",
			"target_name":"TestTarget",
			"target_plugin":"postgres",
			"target_endpoint":"{\"endpoint\":\"schmendpoint\"}",
			"agent":"127.0.0.1:1234"
		}`,
	})

	cJob := dispatch.Register("create-job", cliCreateJob)
	cJob.Summarize("Create a new backup job")
	cJob.Aliases("create job", "new job", "create new job", "make job", "c j")
	cJob.Help(HelpInfo{
		JSONInput: `{
			"name":"TestJob",
			"paused":true,
			"retention":"18a446c4-c068-4c09-886c-cb77b6a85274",
			"schedule":"9a58a3fa-7457-431c-b094-e201b42b5c7b",
			"store":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf",
			"summary":"A Test Job",
			"target":"84751f04-2be2-428d-b6a3-2022c63bf6ee"
		}`,
		JSONOutput: `{
			"uuid":"f6623a6f-8dce-46b2-a293-5525bc3a3588",
			"name":"TestJob",
			"summary":"A Test Job",
			"retention_name":"AnotherPolicy",
			"retention_uuid":"18a446c4-c068-4c09-886c-cb77b6a85274",
			"expiry":31536000,
			"schedule_name":"AnotherSched",
			"schedule_uuid":"9a58a3fa-7457-431c-b094-e201b42b5c7b",
			"schedule_when":"daily 4am",
			"paused":true,
			"store_uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf",
			"store_name":"AnotherStore",
			"store_plugin":"s3",
			"store_endpoint":"{\"endpoint\":\"schmendpoint\"}",
			"target_uuid":"84751f04-2be2-428d-b6a3-2022c63bf6ee",
			"target_name":"TestTarget",
			"target_plugin":"postgres",
			"target_endpoint":"{\"endpoint\":\"schmendpoint\"}",
			"agent":"127.0.0.1:1234"
		}`,
	})

	eJob := dispatch.Register("edit-job", cliEditJob).Aliases("edit job", "update job")
	eJob.Summarize("Modify an existing backup job")
	eJob.Help(HelpInfo{
		Flags: []FlagInfo{JobNameFlag},
		JSONInput: `{
			"name":"AnotherJob",
			"retention":"18a446c4-c068-4c09-886c-cb77b6a85274",
			"schedule":"9a58a3fa-7457-431c-b094-e201b42b5c7b",
			"store":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf",
			"summary":"A Test Job",
			"target":"84751f04-2be2-428d-b6a3-2022c63bf6ee"
		}`,
		JSONOutput: `{
			"uuid":"f6623a6f-8dce-46b2-a293-5525bc3a3588",
			"name":"AnotherJob",
			"summary":"A Test Job",
			"retention_name":"AnotherPolicy",
			"retention_uuid":"18a446c4-c068-4c09-886c-cb77b6a85274",
			"expiry":31536000,
			"schedule_name":"AnotherSched",
			"schedule_uuid":"9a58a3fa-7457-431c-b094-e201b42b5c7b",
			"schedule_when":"daily 4am",
			"paused":true,
			"store_uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf",
			"store_name":"AnotherStore",
			"store_plugin":"s3",
			"store_endpoint":"{\"endpoint\":\"schmendpoint\"}",
			"target_uuid":"84751f04-2be2-428d-b6a3-2022c63bf6ee",
			"target_name":"TestTarget",
			"target_plugin":"postgres",
			"target_endpoint":"{\"endpoint\":\"schmendpoint\"}",
			"agent":"127.0.0.1:1234"
		}`,
	})

	dJob := dispatch.Register("delete-job", cliDeleteJob)
	dJob.Summarize("Delete a backup job")
	dJob.Aliases("delete job", "remove job", "rm job")
	dJob.Help(HelpInfo{
		Flags:      []FlagInfo{JobNameFlag},
		JSONOutput: `{"ok":"Deleted job"}`,
	})

	pause := dispatch.Register("pause", cliPauseJob).Aliases("pause job")
	pause.Summarize("Pause a backup job")
	pause.Help(HelpInfo{
		Flags: []FlagInfo{JobNameFlag},
	})

	unpause := dispatch.Register("unpause", cliUnpauseJob).Aliases("unpause job")
	unpause.Summarize("Unpause a backup job")
	unpause.Help(HelpInfo{
		Flags: []FlagInfo{JobNameFlag},
	})

	run := dispatch.Register("run", cliRunJob).Aliases("run job")
	run.Summarize("Schedule an immediate run of a backup job")
	run.Help(HelpInfo{
		Flags: []FlagInfo{JobNameFlag},
		JSONOutput: `{
			"ok":"Scheduled immediate run of job",
			"task_uuid":"143e5494-63c4-4e05-9051-8b3015eae061"
		}`,
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

	dispatch.HelpGroup("TASKS:")
	tasks := dispatch.Register("tasks", cliListTasks).Aliases("list tasks", "ls tasks")
	tasks.Summarize("List available tasks")
	tasks.Help(HelpInfo{
		Flags: []FlagInfo{
			FlagInfo{
				name: "status", short: 'S', valued: true,
				desc: `Only show tasks with the specified status
							Valid values are one of ['all', 'running', 'pending', 'cancelled']
							If not explicitly set, it defaults to 'running'`,
			},
			FlagInfo{name: "all", short: 'a', desc: "Show all tasks, regardless of state"},
			FlagInfo{name: "limit", desc: "Show only the <value> most recent tasks"},
		},
		JSONOutput: `[{
			"uuid":"0e3736f3-6905-40ba-9adc-06641a282ff4",
			"owner":"system",
			"type":"backup",
			"job_uuid":"9b39b2ed-04dc-4de4-9ee8-265a3f9000e8",
			"archive_uuid":"2a4147ea-84a6-40fc-8028-143efabcc49d",
			"status":"done",
			"started_at":"2016-05-17 11:00:01",
			"stopped_at":"2016-05-17 11:00:02",
			"timeout_at":"",
			"log":"This is where I would put my plugin output if I had one"
		}]`,
	})

	task := dispatch.Register("task", cliGetTask)
	task.Summarize("Print detailed information about a specific task")
	task.Aliases("show task", "view task", "display task", "list task", "ls task")
	task.Help(HelpInfo{})

	cTask := dispatch.Register("cancel-task", cliCancelTask).Aliases("cancel task", "stop task")
	cTask.Summarize("Cancel a running or pending task")
	cTask.Help(HelpInfo{
		JSONOutput: `{
			"ok":"Cancelled task '81746508-bd18-46a8-842e-97911d4b23a3'"
		}`,
	})
	/*
	      ###    ########   ######  ##     ## #### ##     ## ########
	     ## ##   ##     ## ##    ## ##     ##  ##  ##     ## ##
	    ##   ##  ##     ## ##       ##     ##  ##  ##     ## ##
	   ##     ## ########  ##       #########  ##  ##     ## ######
	   ######### ##   ##   ##       ##     ##  ##   ##   ##  ##
	   ##     ## ##    ##  ##    ## ##     ##  ##    ## ##   ##
	   ##     ## ##     ##  ######  ##     ## ####    ###    ########
	*/

	dispatch.HelpGroup("ARCHIVES:")
	archives := dispatch.Register("archives", cliListArchives)
	archives.Summarize("List available backup archives")
	archives.Aliases("list archives", "ls archives")
	archives.Help(HelpInfo{
		Flags: []FlagInfo{
			FlagInfo{
				name: "status", short: 'S', valued: true,
				desc: `Only show archives with the specified state of validity.
								 Accepted values are one of ['all', 'valid']. If not
								 explicitly set, it defaults to 'valid'`,
			},
			FlagInfo{
				name: "target", short: 't', valued: true,
				desc: "Show only archives created from the specified target",
			},
			FlagInfo{
				name: "store", short: 's', valued: true,
				desc: "Show only archives sent to the specified store",
			},
			FlagInfo{
				name: "limit", valued: true,
				desc: "Show only the <value> most recent archives",
			},
			FlagInfo{
				name: "before", short: 'B', valued: true,
				desc: `Show only the archives taken before this point in time. Specify
				  in the format YYYYMMDD`,
			},
			FlagInfo{
				name: "after", short: 'A', valued: true,
				desc: `Show only the archives taken after this point in time. Specify
				  in the format YYYYMMDD`,
			},
			FlagInfo{
				name: "all", short: 'a',
				desc: "Show all archives, regardless of validity. Equivalent to '--status=all'",
			},
		},
		JSONOutput: `[{
			"uuid":"b4a842c5-cb61-4fa1-b0c7-08260fdc3533",
			"key":"thisisastorekey",
			"taken_at":"2016-05-18 11:02:43",
			"expires_at":"2017-05-18 11:02:43",
			"status":"valid",
			"notes":"",
			"target_uuid":"b7aa8269-008d-486a-ba1b-610ee191e4c1",
			"target_plugin":"redis-broker",
			"target_endpoint":"{\"redis_type\":\"broker\"}",
			"store_uuid":"6d52c95f-8d7f-4697-ae32-b9ce51fb4808",
			"store_plugin":"s3",
			"store_endpoint":"{\"endpoint\":\"schmendpoint\"}"
		}]`,
	})

	archive := dispatch.Register("archive", cliGetArchive)
	archive.Summarize("Print detailed information about a backup archive")
	archive.Aliases("show archive", "view archive", "display archive", "list archive", "ls archive")
	archive.Help(HelpInfo{
		Flags: []FlagInfo{
			FlagInfo{
				name: "uuid", positional: true, mandatory: true,
				desc: "A UUID assigned to a single archive instance",
			},
		},
		JSONOutput: `{
			"uuid":"b4a842c5-cb61-4fa1-b0c7-08260fdc3533",
			"key":"thisisastorekey",
			"taken_at":"2016-05-18 11:02:43",
			"expires_at":"2017-05-18 11:02:43",
			"status":"valid",
			"notes":"",
			"target_uuid":"b7aa8269-008d-486a-ba1b-610ee191e4c1",
			"target_plugin":"redis-broker",
			"target_endpoint":"{\"redis_type\":\"broker\"}",
			"store_uuid":"6d52c95f-8d7f-4697-ae32-b9ce51fb4808",
			"store_plugin":"s3",
			"store_endpoint":"{\"endpoint\":\"schmendpoint\"}"
		}`,
	})

	restore := dispatch.Register("restore", cliRestoreArchive)
	restore.Summarize("Restore a backup archive")
	restore.Aliases("restore archive", "restore-archive")
	restore.Help(HelpInfo{
		Flags: []FlagInfo{
			FlagInfo{
				name: "target or uuid", positional: true, mandatory: true,
				desc: `The name or UUID of a single target to restore. In raw mode, it
				  must be a UUID assigned to a single archive instance`,
			},
		},
	})

	dArchive := dispatch.Register("delete-archive", cliDeleteArchive)
	dArchive.Summarize("Delete a backup archive")
	dArchive.Aliases("delete archive", "remove archive", "rm archive")
	dArchive.Help(HelpInfo{
		Flags: []FlagInfo{
			FlagInfo{
				name: "uuid", positional: true, mandatory: true,
				desc: "A UUID assigned to a single archive instance",
			},
		},
		JSONOutput: `{"ok":"Deleted archive"}`,
	})

	dispatch.AddGlobalFlag(FlagInfo{
		name: "debug", short: 'D',
		desc: "Enable the output of debug output",
	})
	dispatch.AddGlobalFlag(FlagInfo{
		name: "trace", short: 'T',
		desc: "Enable the output of verbose trace output",
	})
	dispatch.AddGlobalFlag(FlagInfo{
		name: "skip-ssl-validation", short: 'k',
		desc: "Disable SSL certificate validation",
	})
	dispatch.AddGlobalFlag(FlagInfo{
		name: "raw",
		desc: "Takes any input and gives any output as a JSON object",
	})

	/**************************************************************************/
	err := api.LoadConfig(*opts.Config)
	if err != nil {
		ansi.Fprintf(os.Stderr, "\n@R{ERROR:} Could not parse %s: %s\n", *opts.Config, err)
		os.Exit(1)
	}

	// only check for backends + creds if we aren't manipulating backends/help
	nonAPICommands := regexp.MustCompile(`(help|commands|flags|options|backends|list backends|ls be|create backend|c be|update backend|backend|use backend)`)
	if !nonAPICommands.MatchString(strings.Join(command, " ")) {
		DEBUG("Command: '%s'", strings.Join(command, " "))

		if *opts.Shield != "" || os.Getenv("SHIELD_API") != "" {
			ansi.Fprintf(os.Stderr, "@Y{WARNING: -H, --host, and the SHIELD_API environment variable have been deprecated and will be removed in a later release.} Use `shield backend` instead\n")
		}

		loadBackend()
	}

	cmd, cmdname, args := dispatch.ParseCommand(command...)
	maybeWarnDeprecation(cmdname, cmd)
	if cmd == nil {
		ansi.Fprintf(os.Stderr, "@R{unrecognized command %s}\n", strings.Join(command, " "))
		os.Exit(1)
	}

	if err := cmd.Run(args...); err != nil {
		if *opts.Raw {
			RawJSON(map[string]string{"error": err.Error()})
		} else {
			ansi.Fprintf(os.Stderr, "@R{%s}\n", err)
		}
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}
