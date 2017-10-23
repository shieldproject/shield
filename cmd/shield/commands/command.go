package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
)

type commandFn func(opts *Options, args ...string) error

//Command holds all the information about a command that the Dispatcher, well...
// dispatches to
type Command struct {
	canonical string    //Canonical name of the command
	Summary   string    //Summary description of a command (short help)
	RunFn     commandFn //Function to call to run a command
	Flags     FlagList  //Function to call to print help about a command
}

//GetAliases retrieves all aliases for this command registered with the Dispatcher
func (c *Command) GetAliases() []string {
	return AliasesFor(c)
}

//AKA registers alterative names for this command with the Dispatcher
func (c *Command) AKA(aliases ...string) {
	for _, alias := range aliases {
		if _, found := commands[alias]; found {
			panic(fmt.Sprintf("Attempting to register duplicate alias `%s'", alias))
		}

		commands[alias] = c
	}
}

//ShortHelp prints the one line help for a command
func (c *Command) ShortHelp() string {
	const minBuffer = 3
	//Number of spaces taken up by command name and extra buffer spaces before
	//summary. All commands will have the same buffer, so this justifies the
	//all of the summarys' text
	summaryIndent := maxCmdLen + minBuffer
	//Create format string with buffer specified
	return ansi.Sprintf("@B{%-[1]*[2]s}%[3]s", summaryIndent, c.canonical, c.Summary)
}

//Run runs the attached runFn with the given args
func (c *Command) Run(args ...string) error {
	return c.RunFn(Opts, args...)
}

//DisplayHelp prints the help for this command to stderr
func (c *Command) DisplayHelp() {
	//Print Usage Line
	fmt.Fprintln(os.Stderr, c.usageLine())
	fmt.Fprintln(os.Stderr, c.Summary)

	//Print Aliases, if present
	aliases := c.GetAliases()
	if len(aliases) > 0 {
		fmt.Fprintln(os.Stderr, HelpHeader("ALIASES"))
		fmt.Fprintf(os.Stderr, "  %s\n", strings.Join(c.GetAliases(), ", "))
	}

	//Print flag help if there are any flags for this command
	if len(c.Flags) > 0 {
		fmt.Fprintln(os.Stderr, HelpHeader("FLAGS"))
		fmt.Fprintln(os.Stderr, strings.Join(internal.IndentSlice(c.Flags.HelpStrings()), "\n"))
	}

	if len(GlobalFlags) != 0 {
		fmt.Fprintln(os.Stderr, HelpHeader("GLOBAL FLAGS"))
		fmt.Fprintln(os.Stderr, strings.Join(internal.IndentSlice(GlobalFlags.HelpStrings()), "\n"))
	}
}

func (c *Command) usageLine() string {
	components := []string{ansi.Sprintf("@G{shield %s}", c.canonical)}
	for _, f := range c.Flags {
		var flagNotation string
		if !f.Mandatory {
			flagNotation = ansi.Sprintf("@G{[%s]}", f.formatShortIfPresent())
		} else {
			flagNotation = ansi.Sprintf("@G{%s}", f.formatShortIfPresent())
		}
		components = append(components, flagNotation)
	}
	return strings.Join(components, " ")
}

//Panics if command doesn't have all the things a command should
func (c *Command) validate() {
	if c.canonical == "" {
		panic(fmt.Sprintf("Command missing canonical name: %+v\n", c))
	}

	if c.RunFn == nil {
		panic(fmt.Sprintf("Command missing function: %+v\n", c))
	}
}
