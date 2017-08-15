package info

import (
	"fmt"
	"os"
	"strings"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/cmd/shield/commands"
)

func init() {
	help := commands.Register("help", cliUsage)
	help.Summarize("Get detailed help with a specific command")
	help.Aliases("usage", "commands")
	help.Help(commands.HelpInfo{
		Message: ansi.Sprintf("@R{This is getting a bit too meta, don't you think?}"),
	})
	help.HelpGroup(commands.InfoGroup)
}

//Get detailed help with a specific command
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
		lineBreak()

		//This message should go away in v8, along with the deprecated aliases
		ansi.Fprintf(os.Stderr, "@R{The verbose, multi-word commands (such as `list schedules`) are now deprecated}\n"+
			"@R{in favor of, for example, the shorter `schedules`. Other long commands have had their}\n"+
			"@R{spaces replaced with dashes. Check `commands` for the new canonical names.}\n")
		return nil
	}

	c, _, _ := commands.ParseCommand(args...)
	c.DisplayHelp()
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
}

func printGlobalFlags() {
	header("GLOBAL FLAGS")
	contents(commands.GlobalFlags())
}

func printCommandList() {
	header("COMMANDS")
	contents(commands.CommandString())
}
