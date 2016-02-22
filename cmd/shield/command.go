package main

import (
	"fmt"
	"strings"
)

type Options struct {
	Shield *string

	Used     *bool
	Unused   *bool
	Paused   *bool
	Unpaused *bool
	All      *bool

	Debug *bool
	Trace *bool
	Raw   *bool

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
}

type Handler func(opts Options, args []string) error

type Command struct {
	help     [][]string
	commands map[string]Handler
	options  Options
}

func NewCommand() *Command {
	return &Command{
		commands: map[string]Handler{},
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
	c.commands[command] = fn
}

func (c *Command) Alias(alias string, command string) {
	if fn, ok := c.commands[command]; ok {
		c.Dispatch(alias, "", fn)
	} else {
		panic(fmt.Sprintf("unknown command `%s' for alias `%s'", command, alias))
	}
}

func (c *Command) With(opts Options) *Command {
	c.options = opts
	return c
}

func (c *Command) Execute(cmd ...string) error {
	var last = 0
	for i := 1; i <= len(cmd); i++ {
		command := strings.Join(cmd[0:i], " ")
		if _, ok := c.commands[command]; ok {
			last = i
		}
	}
	if last != 0 {
		command := strings.Join(cmd[0:last], " ")
		if fn, ok := c.commands[command]; ok {
			return fn(c.options, cmd[last:])
		}
	}
	return fmt.Errorf("unrecognized command %s\n", strings.Join(cmd, " "))
}
