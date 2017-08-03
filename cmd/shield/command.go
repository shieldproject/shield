package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/starkandwayne/goutils/ansi"
)

type Options struct {
	Shield *string

	Used     *bool
	Unused   *bool
	Paused   *bool
	Unpaused *bool
	All      *bool

	Debug             *bool
	Trace             *bool
	Raw               *bool
	ShowUUID          *bool
	UpdateIfExists    *bool
	Fuzzy             *bool
	SkipSSLValidation *bool
	Version           *bool

	Status *string

	Target    *string
	Store     *string
	Schedule  *string
	Retention *string

	Plugin *string

	After  *string
	Before *string

	To *string

	Limit *string

	Config   *string
	User     *string
	Password *string
}

type commandFn func(args ...string) error

//Command holds all the information about a command that the Dispatcher, well...
// dispatches to
type Command struct {
	canonical string      //Canonical name of the command
	summary   string      //Summary description of a command (short help)
	runFn     commandFn   //Function to call to run a command
	help      HelpInfo    //Function to call to print help about a command
	dispatch  *Dispatcher //Pointer to the dispatcher that created this Command
}

//GetAliases retrieves all aliases for this command registered with the Dispatcher
func (c *Command) GetAliases() []string {
	return c.dispatch.AliasesFor(c)
}

//Aliases registers alterative names for this command with the Dispatcher
func (c *Command) Aliases(aliases ...string) *Command {
	for _, alias := range aliases {
		dispatch.commands[alias] = c
	}
	return c
}

//Summarize sets the short help string for this command
func (c *Command) Summarize(summary string) *Command {
	c.summary = summary
	return c
}

//SetHelp sets the function to call if the help for this command is requested
func (c *Command) Help(help HelpInfo) *Command {
	c.help = help
	return c
}

//ShortHelp prints the one line help for a command
func (c *Command) ShortHelp() string {
	const minBuffer = 3
	//Number of spaces taken up by command name and extra buffer spaces before
	//summary. All commands will have the same buffer, so this justifies the
	//all of the summarys' text
	summaryIndent := c.dispatch.maxCmdLen + minBuffer
	//Create format string with buffer specified
	return ansi.Sprintf("  @B{%-[1]*[2]s}%[3]s", summaryIndent, c.canonical, c.summary)
}

//Run runs the attached runFn with the given args
func (c *Command) Run(args ...string) error {
	return c.runFn(args...)
}

//DisplayHelp prints the help for this command to stderr
func (c *Command) DisplayHelp() {
	//Print Usage Line
	fmt.Fprintln(os.Stderr, c.usageLine())
	if c.help.Message != "" {
		fmt.Fprintln(os.Stderr, c.help.Message)
	} else { //Default to summary unless overridden in help
		fmt.Fprintln(os.Stderr, c.summary)
	}

	//Print Aliases, if present
	aliases := c.GetAliases()
	if len(aliases) > 0 {
		fmt.Fprintln(os.Stderr, HelpHeader("ALIASES"))
		fmt.Fprintf(os.Stderr, "  %s\n", strings.Join(c.GetAliases(), ", "))
	}

	//Print flag help if there are any flags for this command
	if len(c.help.Flags) > 0 {
		fmt.Fprintln(os.Stderr, HelpHeader("FLAGS"))
		fmt.Fprintln(os.Stderr, strings.Join(c.help.FlagHelp(), "\n"))
	}

	//Print JSON input, if present
	if c.help.JSONInput != "" {
		fmt.Fprintln(os.Stderr, HelpHeader("RAW INPUT"))
		fmt.Fprintln(os.Stderr, PrettyJSON(c.help.JSONInput))
	}

	//Print JSON output, if present
	if c.help.JSONOutput != "" {
		fmt.Fprintln(os.Stderr, HelpHeader("RAW OUTPUT"))
		fmt.Fprintln(os.Stderr, PrettyJSON(c.help.JSONOutput))
	}
}

func (c *Command) usageLine() string {
	components := []string{ansi.Sprintf("@G{shield %s}", c.canonical)}
	for _, f := range c.help.Flags {
		var flagNotation string
		if !f.mandatory {
			flagNotation = ansi.Sprintf("@G{[%s]}", f.formatShortIfPresent())
		} else {
			flagNotation = ansi.Sprintf("@G{%s}", f.formatShortIfPresent())
		}
		components = append(components, flagNotation)
	}
	return strings.Join(components, " ")
}
