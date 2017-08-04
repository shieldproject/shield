package main

import (
	"bytes"
	"os"
	"strings"

	"github.com/pborman/getopt"
	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

//Get detailed help with a specific command
func cliUsage(args ...string) error {
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
	}
	//Allow `help commands`
	if c, _, _ := dispatch.ParseCommand(args...); c == dispatch.commands["commands"] {
		cliCommands()
		return nil
	}
	//Allow `help commands`
	if c, _, _ := dispatch.ParseCommand(args...); c == dispatch.commands["flags"] {
		cliFlags()
		return nil
	}

	c, _, _ := dispatch.ParseCommand(args...)
	c.DisplayHelp()
	return nil
}

//Show the list of available commands
func cliCommands(args ...string) error {
	ansi.Fprintf(os.Stderr, "\n@R{NAME:}\n  shield\t\tCLI for interacting with the Shield API.\n")
	ansi.Fprintf(os.Stderr, "\n@R{USAGE:}\n  shield [options] <command>\n")
	ansi.Fprintf(os.Stderr, "\n@R{ENVIRONMENT VARIABLES:}\n")
	ansi.Fprintf(os.Stderr, "  SHIELD_TRACE\t\tset to 'true' for trace output.\n")
	ansi.Fprintf(os.Stderr, "  SHIELD_DEBUG\t\tset to 'true' for debug output.\n\n")
	ansi.Fprintf(os.Stderr, "@R{GLOBAL FLAGS:}\n")
	ansi.Fprintf(os.Stderr, dispatch.GlobalFlags())
	ansi.Fprintf(os.Stderr, "\n\n")
	ansi.Fprintf(os.Stderr, "@R{COMMANDS:}\n")
	ansi.Fprintf(os.Stderr, dispatch.Usage())
	ansi.Fprintf(os.Stderr, "\n")
	return nil
}

//Show the list of all command line flags
func cliFlags(args ...string) error {
	getopt.PrintUsage(os.Stderr)
	return nil
}

//Query the SHIELD backup server for its status and version info
func cliStatus(args ...string) error {
	status, err := api.GetStatus()
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
}
