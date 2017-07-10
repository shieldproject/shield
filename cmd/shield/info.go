package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/pborman/getopt"
	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

//Get detailed help with a specific command
func cliUsage(opts Options, args []string, help bool) error {
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
}

//Show the list of available commands
func cliCommands(opts Options, args []string, help bool) error {
	ansi.Fprintf(os.Stderr, "\n@R{NAME:}\n  shield\t\tCLI for interacting with the Shield API.\n")
	ansi.Fprintf(os.Stderr, "\n@R{USAGE:}\n  shield [options] <command>\n")
	ansi.Fprintf(os.Stderr, "\n@R{ENVIRONMENT VARIABLES:}\n")
	ansi.Fprintf(os.Stderr, "  SHIELD_TRACE\t\tset to 'true' for trace output.\n")
	ansi.Fprintf(os.Stderr, "  SHIELD_DEBUG\t\tset to 'true' for debug output.\n\n")
	ansi.Fprintf(os.Stderr, "@R{COMMANDS:}\n\n")
	ansi.Fprintf(os.Stderr, c.Usage())
	ansi.Fprintf(os.Stderr, "\n")
	return nil
}

//Show the list of all command line flags
func cliFlags(opts Options, args []string, help bool) error {
	getopt.PrintUsage(os.Stderr)
	return nil
}

//Query the SHIELD backup server for its status and version info
func cliStatus(opts Options, args []string, help bool) error {
	if help {
		FlagHelp("Outputs information as a JSON object", true, "--raw")
		JSONHelp(fmt.Sprintf("{\"name\":\"MyShield\",\"version\":\"%s\"}\n", Version))
		return nil
	}

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
