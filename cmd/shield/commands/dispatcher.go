package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/starkandwayne/goutils/ansi"
)

var (
	commands = map[string]*Command{}
	//GlobalFlags is the array of global flags for the program
	GlobalFlags = []FlagInfo{}
	//Length, in chars, of the longest command name. Set from Dispatch()
	maxCmdLen = 0
)

//HelpGroup represents a set of commands for help organization purposes
type HelpGroup struct {
	name     string
	commands []*Command
}

//Enumeration for HelpGroupTypes
var (
	InfoGroup      = &HelpGroup{name: "INFO"}
	BackendsGroup  = &HelpGroup{name: "BACKENDS"}
	TargetsGroup   = &HelpGroup{name: "TARGETS"}
	StoresGroup    = &HelpGroup{name: "STORES"}
	SchedulesGroup = &HelpGroup{name: "SCHEDULES"}
	PoliciesGroup  = &HelpGroup{name: "POLICIES"}
	JobsGroup      = &HelpGroup{name: "JOBS"}
	ArchivesGroup  = &HelpGroup{name: "ARCHIVES"}
	TasksGroup     = &HelpGroup{name: "TASKS"}
)

func (h *HelpGroup) addCommand(c *Command) {
	h.commands = append(h.commands, c)
}

func (h *HelpGroup) String() string {
	var helpLines []string
	groupHeader := ansi.Sprintf("@M{%s:}", h.name)
	helpLines = append(helpLines, groupHeader) //Add the helpGroup header

	for _, command := range h.commands {
		helpLines = append(helpLines, command.ShortHelp()) //Add each command's help line
	}

	return strings.Join(helpLines, "\n")
}

//CommandString returns a string listing all the commands dispatched to this
//package, each on a line with their command summary
func CommandString() string {
	var helpLines []string
	groupList := []*HelpGroup{
		InfoGroup,
		BackendsGroup,
		TargetsGroup,
		SchedulesGroup,
		PoliciesGroup,
		StoresGroup,
		JobsGroup,
		TasksGroup,
		ArchivesGroup,
	}

	for _, group := range groupList {
		//Blank line before next group starts
		helpLines = append(helpLines, group.String(), "")
	}

	return strings.Join(helpLines[:len(helpLines)-1], "\n") //Split by newline
}

//GlobalFlagHelp returns the formatted help lines for the registered global flags
func GlobalFlagHelp() string {
	return strings.Join(HelpInfo{Flags: GlobalFlags}.FlagHelp(), "\n")
}

//Add registers a command to the Dispatcher object, callable by the name
// `command`, and then returns the newly-created and registered Command struct.
func Add(commandName string, cmd *Command) *Command {
	if _, exists := commands[commandName]; exists {
		panic(fmt.Sprintf("command `%s' already registered", commandName))
	}

	cmd.canonical = commandName
	cmd.validate()
	cmd.Group.addCommand(cmd)

	commands[commandName] = cmd
	if len(commandName) > maxCommandLength() {
		setMaxCommandLength(len(commandName))
	}

	return cmd
}

//AliasesFor returns a slice of alias names for the given command
func AliasesFor(command *Command) []string {
	aliases := []string{}
	for name, cmd := range commands {
		//If both point to the same Command struct but this isn't the canonical name
		if command == cmd && name != command.canonical {
			aliases = append(aliases, name)
		}
	}
	sort.Strings(aliases)
	return aliases
}

//ParseCommand finds the Command struct registered under the given name.
//Searches for the longest name it can construct from the beginning of the
//input that matches a registered command. Returns the matched Command, the
//name given by the user that matched that command, and the unmatched remainder
//of the input array to be used as Command args. Returns nil, "the bad command", nil if no
//match can be found
func ParseCommand(userInput ...string) (cmd *Command, givenName string, args []string) {
	if len(userInput) == 0 {
		userInput = []string{"help"}
	}

	for i := 1; i <= len(userInput); i++ {
		givenName = strings.Join(userInput[:i], " ")
		args = userInput[i:]
		var found bool
		if cmd, found = commands[givenName]; found {
			break
		}
	}
	return
}

//MaybeWarnDeprecation will print a deprecation message to the screen if the
//given name for the command is an alias
func MaybeWarnDeprecation(name string, cmd *Command) {
	if cmd != nil && name != cmd.canonical {
		ansi.Fprintf(os.Stderr, "@R{The alias `%s` is deprecated in favor of `%s`}\n", name, cmd.canonical)
	}
}

//Really, the only reason this is a function is because I've realized that it
// can be difficult in go to, while reading Go, tell what is a local variable
// vs a package variable until you track down the definition. At least a function
// will make people seek out the definition. The compiler should inline it anyway.
func maxCommandLength() int {
	return maxCmdLen
}

func setMaxCommandLength(i int) {
	maxCmdLen = i
}
