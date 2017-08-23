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
	Help              *bool

	Status *string

	Target    *string
	Store     *string
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
	Summary   string    //Summary description of a command (short help)
	RunFn     commandFn //Function to call to run a command
	Help      *HelpInfo //Function to call to print help about a command
	Group     *HelpGroup
}

//GetAliases retrieves all aliases for this command registered with the Dispatcher
func (c *Command) GetAliases() []string {
	return AliasesFor(c)
}

//AKA registers alterative names for this command with the Dispatcher
func (c *Command) AKA(aliases ...string) {
	for _, alias := range aliases {
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
	if c.Help.Message != "" {
		fmt.Fprintln(os.Stderr, c.Help.Message)
	} else { //Default to summary unless overridden in help
		fmt.Fprintln(os.Stderr, c.Summary)
	}

	//Print Aliases, if present
	aliases := c.GetAliases()
	if len(aliases) > 0 {
		fmt.Fprintln(os.Stderr, HelpHeader("ALIASES"))
		fmt.Fprintf(os.Stderr, "  %s\n", strings.Join(c.GetAliases(), ", "))
	}

	//Print flag help if there are any flags for this command
	if len(c.Help.Flags) > 0 {
		fmt.Fprintln(os.Stderr, HelpHeader("FLAGS"))
		fmt.Fprintln(os.Stderr, strings.Join(internal.IndentSlice(c.Help.FlagHelp()), "\n"))
	}

	if len(GlobalFlags) != 0 {
		fmt.Fprintln(os.Stderr, HelpHeader("GLOBAL FLAGS"))
		fmt.Fprintln(os.Stderr, internal.IndentString(GlobalFlagHelp()))
	}

	//Print JSON input, if present
	if c.Help.JSONInput != "" {
		fmt.Fprintln(os.Stderr, HelpHeader("RAW INPUT"))
		fmt.Fprintln(os.Stderr, internal.PrettyJSON(c.Help.JSONInput))
	}

	//Print JSON output, if present
	if c.Help.JSONOutput != "" {
		fmt.Fprintln(os.Stderr, HelpHeader("RAW OUTPUT"))
		fmt.Fprintln(os.Stderr, internal.PrettyJSON(c.Help.JSONOutput))
	}
}

func (c *Command) usageLine() string {
	components := []string{ansi.Sprintf("@G{shield %s}", c.canonical)}
	for _, f := range c.Help.Flags {
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

	if c.Summary == "" && c.Help.Message == "" {
		panic(fmt.Sprintf("Command missing summary: %+v\n", c))
	}

	if c.RunFn == nil {
		panic(fmt.Sprintf("Command missing function: %+v\n", c))
	}

	if c.Help == nil {
		panic(fmt.Sprintf("Command missing help: %+v\n", c))
	}

	if c.Group == nil {
		panic(fmt.Sprintf("Command not assigned to group: %+v\n", c))
	}
}
