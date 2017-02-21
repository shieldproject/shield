package main

import (
	"fmt"
	"os"
	"reflect"
	"sort"
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

type Handler func(opts Options, args []string, help bool) error

type Command struct {
	help       [][]string
	summary    map[string]string
	commands   map[string]Handler
	deprecated map[string]bool
	options    Options
}

func NewCommand() *Command {
	return &Command{
		summary:    map[string]string{},
		commands:   map[string]Handler{},
		deprecated: map[string]bool{},
	}
}

func (c *Command) HelpBreak() {
	c.help = append(c.help, []string{"", ""})
}

func (c *Command) HelpGroup(groupname string) {
	groupname = "@M{" + fmt.Sprintf("%s", groupname) + "}"
	c.help = append(c.help, []string{groupname, ""})
}

func (c *Command) Usage() string {
	n := 0
	for _, v := range c.help {
		if len(v[0]) > n {
			n = len(v[0])
		}
	}

	format := fmt.Sprintf("  %%-%ds   %%s\n", n)
	l := make([]string, len(c.help))
	for i, v := range c.help {
		l[i] = fmt.Sprintf(format, v[0], v[1])
	}
	return strings.Join(l, "")
}

func (c *Command) Dispatch(command string, help string, fn Handler) {
	if _, ok := c.commands[command]; ok {
		panic(fmt.Sprintf("command `%s' already registered", command))
	}

	if help != "" {
		c.help = append(c.help, []string{command, help})
	}

	c.summary[command] = help
	c.commands[command] = fn
}

func (c *Command) Alias(alias string, command string) {
	if fn, ok := c.commands[command]; ok {
		c.Dispatch(alias, "", fn)
		if summary, ok := c.summary[command]; ok {
			c.summary[alias] = summary
		}
	} else {
		panic(fmt.Sprintf("unknown command `%s' for alias `%s'", command, alias))
	}
	c.deprecated[alias] = true
}

//Returns a newline separated list of aliases for the given command
func (c *Command) AliasesFor(command string) []string {
	aliases := []string{}
	if _, found := c.commands[command]; !found {
		panic(fmt.Sprintf("unknown command `%s' to find aliases for", command))
	}
	for alias, _ := range c.commands {
		if c.isAliasFor(alias, command) {
			aliases = append(aliases, alias)
		}
	}
	sort.Strings(aliases)
	return aliases
}

func (c *Command) isAliasFor(alias, command string) bool {
	return reflect.ValueOf(c.commands[alias]).Pointer() == reflect.ValueOf(c.commands[command]).Pointer()
}

func (c *Command) isCanonicalFor(canon, command string) bool {
	return c.isAliasFor(canon, command) && !c.deprecated[canon]
}

func (c *Command) getCanonicalOf(command string) string {
	for alias := range c.commands {
		if c.isCanonicalFor(alias, command) {
			return alias
		}
	}
	return ""
}

func (c *Command) With(opts Options) *Command {
	c.options = opts
	return c
}

func (c *Command) do(cmd []string, help bool) error {
	var last = 0
	var err error = nil
	for i := 1; i <= len(cmd); i++ {
		command := strings.Join(cmd[0:i], " ")
		if _, ok := c.commands[command]; ok {
			last = i
		}
	}
	if last != 0 {
		command := strings.Join(cmd[0:last], " ")
		if fn, ok := c.commands[command]; ok {
			if c.deprecated[command] {
				canon := c.getCanonicalOf(command)
				ansi.Fprintf(os.Stderr, "@R{The alias `%s` is deprecated in favor of `%s`}\n", command, canon)
			}
			err = fn(c.options, cmd[last:], help)

			//Avoid recursive help
			helpComs := []string{}
			helpComs = append(helpComs, c.AliasesFor("help")...)
			helpComs = append(helpComs, c.AliasesFor("commands")...)
			helpComs = append(helpComs, c.AliasesFor("flags")...)
			isHelper := false
			for _, v := range helpComs {
				if command == v {
					isHelper = true
					break
				}
			}
			if help && !isHelper {
				if c.deprecated[command] {
					command = c.getCanonicalOf(command)
				}
				PrintUsage(command)
				PrintMessage(command, c)
				PrintAliasHelp(command, c)
				PrintFlagHelp()
				PrintInputHelp()
				PrintJSONHelp()
			}
			return err
		}
	}
	return fmt.Errorf("unrecognized command %s\n", strings.Join(cmd, " "))
}

func (c *Command) Execute(cmd ...string) error {
	return c.do(cmd, false)
}

func (c *Command) Help(cmd ...string) error {
	return c.do(cmd, true)
}
