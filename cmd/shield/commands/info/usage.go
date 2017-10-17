package info

import (
	"fmt"
	"os"
	"strings"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/cmd/shield/commands"
)

//Usage - Get detailed help with a specific command
var Usage = &commands.Command{
	Summary: "Get detailed help with a specific command",
	Help: &commands.HelpInfo{
		Message: ansi.Sprintf("@R{This is getting a bit too meta, don't you think?}"),
	},
	RunFn: cliUsage,
	Group: commands.InfoGroup,
}

func cliUsage(opts *commands.Options, args ...string) error {
	if len(args) == 0 {
		ansi.Fprintf(os.Stderr, "For more help with a command, type @M{shield help <command>}\n")
		printUsage()
		lineBreak()

		printEnvVars()
		lineBreak()

		printGlobalFlags()
		lineBreak()

		printCommandList()
		lineBreak()

		return nil
	}

	c, commandname, _ := commands.ParseCommand(args...)
	if c != nil {
		c.DisplayHelp()
	} else {
		ansi.Fprintf(os.Stderr, "@R{unrecognized command %s}\n", commandname)
	}
	return nil
}

func header(contents string) {
	ansi.Fprintf(os.Stderr, "@R{%s:}\n", contents)
}

func contents(contents string) {
	for _, line := range strings.Split(contents, "\n") {
		fmt.Fprintf(os.Stderr, "  %s\n", line)
	}
}

func lineBreak() {
	fmt.Fprintln(os.Stderr, "")
}

func printUsage() {
	header("NAME")
	contents("shield\t\tCLI for interacting with the Shield API.")
	header("USAGE")
	contents("shield [options] <command>")
}

func printEnvVars() {
	header("ENVIRONMENT VARIABLES")
	contents("SHIELD_TRACE\t\tset to 'true' for trace output.")
	contents("SHIELD_DEBUG\t\tset to 'true' for debug output.")
	contents("SHIELD_API_TOKEN\t\tset to send API token on requests")
}

func printGlobalFlags() {
	header("GLOBAL FLAGS")
	contents(commands.GlobalFlagHelp())
}

func printCommandList() {
	header("COMMANDS")
	contents(commands.CommandString())
}
