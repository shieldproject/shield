package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
)

//Options contains all the possible command line options that commands may
//possibly use
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

//Opts is the options flag struct to be used by all commands
var Opts *Options

type commandFn func(opts *Options, args ...string) error

//Command holds all the information about a command that the Dispatcher, well...
// dispatches to
type Command struct {
	canonical string    //Canonical name of the command
	summary   string    //Summary description of a command (short help)
	runFn     commandFn //Function to call to run a command
	help      HelpInfo  //Function to call to print help about a command
}

//GetAliases retrieves all aliases for this command registered with the Dispatcher
func (c *Command) GetAliases() []string {
	return AliasesFor(c)
}

//Aliases registers alterative names for this command with the Dispatcher
func (c *Command) Aliases(aliases ...string) *Command {
	for _, alias := range aliases {
		commands[alias] = c
	}
	return c
}

//Summarize sets the short help string for this command
func (c *Command) Summarize(summary string) *Command {
	c.summary = summary
	return c
}

//Help sets the function to call if the help for this command is requested
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
	summaryIndent := maxCmdLen + minBuffer
	//Create format string with buffer specified
	return ansi.Sprintf("@B{%-[1]*[2]s}%[3]s", summaryIndent, c.canonical, c.summary)
}

//Run runs the attached runFn with the given args
func (c *Command) Run(args ...string) error {
	return c.runFn(Opts, args...)
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
		fmt.Fprintln(os.Stderr, strings.Join(internal.IndentSlice(c.help.FlagHelp()), "\n"))
	}

	if len(globals.Flags) != 0 {
		fmt.Fprintln(os.Stderr, HelpHeader("GLOBAL FLAGS"))
		fmt.Fprintln(os.Stderr, internal.IndentString(GlobalFlags()))
	}

	//Print JSON input, if present
	if c.help.JSONInput != "" {
		fmt.Fprintln(os.Stderr, HelpHeader("RAW INPUT"))
		fmt.Fprintln(os.Stderr, internal.PrettyJSON(c.help.JSONInput))
	}

	//Print JSON output, if present
	if c.help.JSONOutput != "" {
		fmt.Fprintln(os.Stderr, HelpHeader("RAW OUTPUT"))
		fmt.Fprintln(os.Stderr, internal.PrettyJSON(c.help.JSONOutput))
	}
}

//HelpGroup assigns this command to a help group
func (c *Command) HelpGroup(group *helpGroup) {
	group.addCommand(c)
}

func (c *Command) usageLine() string {
	components := []string{ansi.Sprintf("@G{shield %s}", c.canonical)}
	for _, f := range c.help.Flags {
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
