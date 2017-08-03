package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/starkandwayne/goutils/ansi"
)

//Dispatcher is a registry and lookup table for commands, accessible by command
// names
type Dispatcher struct {
	commands   map[string]*Command
	helpgroups []helpGroup
	globals    HelpInfo
	maxCmdLen  int //Length, in chars, of the longest command name. Set from Dispatch()
}

type helpGroup struct {
	name     string
	commands []*Command
}

func (h *helpGroup) addCommand(c *Command) {
	h.commands = append(h.commands, c)
}

//Singleton command dispatcher
var dispatch = Dispatcher{commands: map[string]*Command{}}

//HelpGroup marks a new section for command help, under which all future
// Register()ed commands will be added to until this function is called again
// with a different label
func (d *Dispatcher) HelpGroup(groupname string) {
	d.helpgroups = append(d.helpgroups, helpGroup{name: groupname})
}

//Usage returns a string listing all the commands dispatched to this struct,
// each on a line with their command summary
func (d *Dispatcher) Usage() string {
	var helpLines []string
	for _, group := range d.helpgroups {
		groupHeader := fmt.Sprintf("  @M{%s}", group.name)
		helpLines = append(helpLines, groupHeader) //Add the helpGroup header

		for _, command := range group.commands {
			helpLines = append(helpLines, command.ShortHelp()) //Add each command's help line
		}

		helpLines = append(helpLines, "") //Add extra line before next group starts
	}

	return strings.Join(helpLines, "\n") //Split by newline
}

//GlobalFlags returns the formatted help lines for the registered global flags
func (d *Dispatcher) GlobalFlags() string {
	return strings.Join(d.globals.FlagHelp(), "\n")
}

//AddGlobalFlag registers a global flag with the dispatcher to be printed in the
// help if necessary
func (d *Dispatcher) AddGlobalFlag(flag FlagInfo) {
	d.globals.Flags = append(d.globals.Flags, flag)
}

//Register registers a command to the Dispatcher object, callable by the name
// `command`, and then returns the newly-created and registered Command struct.
func (d *Dispatcher) Register(commandName string, fn commandFn) *Command {
	if _, exists := d.commands[commandName]; exists {
		panic(fmt.Sprintf("command `%s' already registered", commandName))
	}

	cmd := &Command{
		canonical: commandName,
		runFn:     fn,
		dispatch:  d,
	}

	d.commands[commandName] = cmd
	d.helpgroups[len(d.helpgroups)-1].addCommand(cmd)
	if len(commandName) > d.maxCmdLen {
		d.maxCmdLen = len(commandName)
	}
	return cmd
}

//AliasesFor returns a slice of alias names for the given command
func (d *Dispatcher) AliasesFor(command *Command) []string {
	aliases := []string{}
	for name, cmd := range d.commands {
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
//of the input array to be used as Command args. Returns nil, "", nil if no
//match can be found
func (d *Dispatcher) ParseCommand(userInput ...string) (cmd *Command, givenName string, args []string) {
	for i := 1; i <= len(userInput); i++ {
		cmdName := strings.Join(userInput[:i], " ")
		if command, found := d.commands[cmdName]; found {
			cmd = command
			args = userInput[i:]
			givenName = cmdName
		}
	}
	return
}

func maybeWarnDeprecation(name string, cmd *Command) {
	if cmd != nil && name != cmd.canonical {
		ansi.Fprintf(os.Stderr, "@R{The alias `%s` is deprecated in favor of `%s`}\n", name, cmd.canonical)
	}
}
