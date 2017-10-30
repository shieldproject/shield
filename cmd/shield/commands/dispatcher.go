package commands

import (
	"sort"
	"strings"

	fmt "github.com/jhunt/go-ansi"
)

var (
	commands = map[string]*Command{}
	//GlobalFlags is the array of global flags for the program
	GlobalFlags = FlagList{}
	//helpGroups is the list of help groups for the commands
	//Length, in chars, of the longest command name. Set from Dispatch()
	helpGroups = []helpGroup{helpGroup{name: "placeholder"}}
	maxCmdLen  = 0
)

//HelpGroup represents a set of commands for help organization purposes
type helpGroup struct {
	name     string
	commands []*Command
}

func (h *helpGroup) addCommand(c *Command) {
	h.commands = append(h.commands, c)
}

func (h *helpGroup) String() string {
	var helpLines []string
	groupHeader := fmt.Sprintf("@M{%s:}", h.name)
	helpLines = append(helpLines, groupHeader) //Add the helpGroup header

	for _, command := range h.commands {
		helpLines = append(helpLines, command.ShortHelp()) //Add each command's help line
	}

	return strings.Join(helpLines, "\n")
}

//HelpGroup sets the dispatcher to put further dispatched commands into a
// group with this name, until HelpGroup is called again
func HelpGroup(name string) {
	helpGroups = append(helpGroups, helpGroup{name: name})
}

//Reset wipes away all of the registered commands and global flags, leaving you
// with a fresh dispatcher state. Useless for the actual program, but great for
// testing
func Reset() {
	commands = map[string]*Command{}
	GlobalFlags = []FlagInfo{}
	maxCmdLen = 0
}

//CommandString returns a string listing all the commands dispatched to this
//package, each on a line with their command summary
func CommandString() string {
	var helpLines []string
	for _, group := range helpGroups {
		if len(group.commands) > 0 {
			helpLines = append(helpLines, group.String(), "")
		}
	}

	return strings.Join(helpLines[:len(helpLines)-1], "\n") //Split by newline
}

//Add registers a command to the Dispatcher object, callable by the name
// `command`, and then returns the newly-created and registered Command struct.
func Add(commandName string, cmd *Command) *Command {
	if _, exists := commands[commandName]; exists {
		panic(fmt.Sprintf("command `%s' already registered", commandName))
	}

	cmd.canonical = commandName
	cmd.validate()
	helpGroups[len(helpGroups)-1].addCommand(cmd)

	commands[commandName] = cmd
	if len(commandName) > maxCmdLen {
		maxCmdLen = len(commandName)
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
	givenName = strings.Join(userInput, " ")

	if len(userInput) == 0 {
		userInput = []string{"help"}
	}

	for i := 1; i <= len(userInput); i++ {
		thisName := strings.Join(userInput[:i], " ")
		if thisCmd, found := commands[thisName]; found {
			cmd, givenName = thisCmd, thisName
			args = userInput[i:]
		}
	}
	return
}
