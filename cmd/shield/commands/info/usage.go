package info

import (
	"os"
	"strings"

	fmt "github.com/jhunt/go-ansi"

	"github.com/starkandwayne/shield/cmd/shield/commands"
)

//Usage - Get detailed help with a specific command
var Usage = &commands.Command{
	Summary: "Get detailed help with a specific command",
	RunFn:   cliUsage,
}

func cliUsage(opts *commands.Options, args ...string) error {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "For more help with a command, type @M{shield help <command>}\n")
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
		fmt.Fprintf(os.Stderr, "@R{unrecognized command %s}\n", commandname)
	}
	return nil
}

func header(contents string) {
	fmt.Fprintf(os.Stderr, "@R{%s:}\n", contents)
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
	contents("SHIELD_API_TOKEN\tset as your API token to send API token on requests")
}

func printGlobalFlags() {
	header("GLOBAL FLAGS")
	contents(strings.Join(commands.GlobalFlags.HelpStrings(), "\n"))
}

func printCommandList() {
	header("COMMANDS")
	contents(commands.CommandString())
}
