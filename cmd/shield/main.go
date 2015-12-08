package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/pborman/getopt"
)

var (
	//== Root Command for Shield

	ShieldCmd = &cobra.Command{
		Use: "shield",
		Long: `Shield - Protect your data with confidence

Shield allows you to schedule backups of all your data sources, set retention
policies, monitor and control your backup tasks, and restore that data should
the need arise.`,

		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if t := viper.GetString("ShieldTarget"); t != "" {
				os.Setenv("SHIELD_TARGET", t)
			}
		},
	}

	//== Base Verbs

	createCmd  = &cobra.Command{Use: "create", Short: "Create a new {{children}}"}
	listCmd    = &cobra.Command{Use: "list", Short: "List all the {{children}}"}
	showCmd    = &cobra.Command{Use: "show", Short: "Show details for the specified {{children}}"}
	deleteCmd  = &cobra.Command{Use: "delete", Short: "Delete the specified {{children}}"}
	updateCmd  = &cobra.Command{Use: "update", Short: "Update the specified {{children}}"}
	editCmd    = &cobra.Command{Use: "edit", Short: "Edit the specified {{children}}"}
	pauseCmd   = &cobra.Command{Use: "pause", Short: "Pause the specified {{children}}"}
	unpauseCmd = &cobra.Command{Use: "unpause", Short: "Continue the specified paused {{children}}"}
	pausedCmd  = &cobra.Command{Use: "paused", Short: "Check if the specified {{children}} is paused"}
	runCmd     = &cobra.Command{Use: "run", Short: "Run the specified {{children}}"}
	cancelCmd  = &cobra.Command{Use: "cancel", Short: "Cancel the specified running {{children}}"}
	restoreCmd = &cobra.Command{Use: "restore", Short: "Restore the specified {{children}}"}

	CfgFile, ShieldTarget string
	Verbose               bool
)

//--------------------------

func main() {
	var options = struct {
		Shield *string

		Used     *bool
		Unused   *bool
		Paused   *bool
		Unpaused *bool
		All      *bool

		Debug *bool

		Status *string

		Target    *string
		Store     *string
		Schedule  *string
		Retention *string

		Plugin *string

		After  *string
		Before *string

		To *string
	}{
		Shield:   getopt.StringLong("shield", 'H', "SHIELD target to run command against, i.e. http://shield.my.domain:8080"),
		Used:     getopt.BoolLong("used", 0, "Only show things that are in-use by something else"),
		Unused:   getopt.BoolLong("unused", 0, "Only show things that are not used by something else"),
		Paused:   getopt.BoolLong("paused", 0, "Only show jobs that are paused"),
		Unpaused: getopt.BoolLong("unpaused", 0, "Only show jobs that are unpaused"),
		All:      getopt.BoolLong("all", 'a', "Show all the things"),

		Debug: getopt.BoolLong("debug", 'D', "Enable debugging"),

		Status:    getopt.StringLong("status", 'S', "", "Only show tasks with the given status (one of 'pending', 'running', 'canceled' or 'done')"),
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

	var err error
	switch command[0] {
	case "list":
		switch command[1] {
		case "targets":
			err = ListTargets(ListTargetOptions{
				Unused: *options.Unused,
				Used:   *options.Used,
				Plugin: *options.Plugin,
			})

		case "schedules":
			err = ListSchedules(ListScheduleOptions{
				Unused: *options.Unused,
				Used:   *options.Used,
			})

		case "retention":
			switch command[2] {
			case "policies":
				err = ListRetentionPolicies(ListRetentionOptions{
					Unused: *options.Unused,
					Used:   *options.Used,
				})
			}
		case "stores":
			err = ListStores(ListStoreOptions{
				Unused: *options.Unused,
				Used:   *options.Used,
				Plugin: *options.Plugin,
			})
		case "jobs":
			err = ListJobs(ListJobOptions{
				Unpaused:  *options.Unpaused,
				Paused:    *options.Paused,
				Target:    *options.Target,
				Store:     *options.Store,
				Schedule:  *options.Schedule,
				Retention: *options.Retention,
			})
		case "tasks":
			err = ListTasks(ListTaskOptions{
				All:   *options.All,
				Debug: *options.Debug,
			})
		case "archives":
			err = ListArchives(ListArchiveOptions{
				Target: *options.Target,
				Store:  *options.Store,
				Before: *options.Before,
				After:  *options.After,
			})
		}

	case "show":
		switch command[1] {
		case "target":
			err = ListTargets(ListTargetOptions{
				UUID: command[2],
			})
		case "schedule":
			err = ListSchedules(ListScheduleOptions{
				UUID: command[2],
			})
		case "retention":
			switch command[2] {
			case "policy":
				err = ListRetentionPolicies(ListRetentionOptions{
					UUID: command[3],
				})
			}
		case "store":
			err = ListStores(ListStoreOptions{
				UUID: command[2],
			})
		case "job":
			err = ListJobs(ListJobOptions{
				UUID: command[2],
			})
		case "task":
			err = ListTasks(ListTaskOptions{
				UUID: command[2],
			})
		case "archive":
			err = ListArchives(ListArchiveOptions{
				UUID: command[2],
			})
		}

		/*
			case "edit", "update":
				switch command[1] {
				case "target":
				case "schedule":
				case "retention":
					switch command[2] {
					case "policy":
					}
				case "store":
				case "job":
				case "task":
				case "archive":
				}

			case "delete":
				switch command[1] {
				case "target":
				case "schedule":
				case "retention":
					switch command[2] {
					case "policy":
					}
				case "store":
				case "job":
				case "archive":
				}*/
	case "restore":
		switch command[1] {
		case "archive":
			err = RestoreArchiveByUUID(ListArchiveOptions{
				Target: *options.To,
				UUID:   command[2],
			})
		}
	case "cancel":
		switch command[1] {
		case "task":
			err = CancelTaskByUUID(command[2])
		}
	case "pause":
		switch command[1] {
		case "job":
			err = PauseUnpauseJob(true, command[2])
		}
	case "unpause":
		switch command[1] {
		case "job":
			err = PauseUnpauseJob(false, command[2])
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)

	// legacy
	viper.SetConfigType("yaml") // To support lnguyen development

	//ShieldCmd.PersistentFlags().StringVar(&CfgFile, "shield_config", "shield_config.yml", "config file (default is shield_config.yaml)")
	ShieldCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")
	ShieldCmd.PersistentFlags().StringVar(&ShieldTarget, "target", "", "Full URI of the SHIELD backup system, i.e. http://shield:8080")

	viper.BindPFlag("Verbose", ShieldCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("ShieldTarget", ShieldCmd.PersistentFlags().Lookup("target"))

	addSubCommandWithHelp(ShieldCmd, createCmd, listCmd, showCmd, deleteCmd, updateCmd, editCmd, pauseCmd, unpauseCmd, pausedCmd, runCmd, cancelCmd, restoreCmd)

	ShieldCmd.Execute()

	if Verbose {
		fmt.Printf("config: %s\ntarget: %s\n", CfgFile, ShieldTarget)
	}
}

func debug(cmd *cobra.Command, args []string) {

	// Trace back through the cmd chain to assemble the full command
	var cmd_list = make([]string, 0)
	ptr := cmd
	for {
		cmd_list = append([]string{ptr.Use}, cmd_list...)
		if ptr.Parent() != nil {
			ptr = ptr.Parent()
		} else {
			break
		}
	}

	fmt.Print("Command: ")
	fmt.Print(strings.Join(cmd_list, " "))
	fmt.Printf(" Argv [%s]\n", args)
}

func addSubCommandWithHelp(tgtCmd *cobra.Command, subCmds ...*cobra.Command) {
	tgtCmd.AddCommand(subCmds...)

	for _, subCmd := range subCmds {
		var children = make([]string, 0)
		var sentence string

		for _, childCmd := range subCmd.Commands() {
			// TODO: if subCommand children have further children, assume compound command and add it
			children = append(children, childCmd.Use)
		}

		if len(children) > 0 {
			if len(children) == 1 {
				sentence = children[0]
			} else {
				sentence = strings.Join(children[0:(len(children)-1)], ", ") + " or " + children[len(children)-1]
			}
			subCmd.Short = strings.Replace(subCmd.Short, "{{children}}", sentence, -1)
		}
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

func parseTristateOptions(cmd *cobra.Command, trueFlag, falseFlag string) string {

	trueFlagSet, _ := cmd.Flags().GetBool(trueFlag)
	falseFlagSet, _ := cmd.Flags().GetBool(falseFlag)

	// Validate Request
	tristate := ""
	if trueFlagSet {
		if falseFlagSet {
			fmt.Fprintf(os.Stderr, "\nERROR: Cannot specify --%s and --%s at the same time\n\n", trueFlag, falseFlag)
			os.Exit(1)
		}
		tristate = "t"
	}
	if falseFlagSet {
		tristate = "f"
	}
	return tristate
}
