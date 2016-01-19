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
}

type Handler func(opts Options, args []string) error

type Command struct {
	commands map[string]Handler
	options  Options
}

func NewCommand() *Command {
	return &Command{
		commands: map[string]Handler{},
	}
}

func (c *Command) Dispatch(command string, fn Handler) {
	if _, ok := c.commands[command]; ok {
		panic(fmt.Sprintf("command `%s' already registered", command))
	}

	c.commands[command] = fn
}

func (c *Command) Alias(alias string, command string) {
	if fn, ok := c.commands[command]; ok {
		c.Dispatch(alias, fn)
	} else {
		panic(fmt.Sprintf("unknown command `%s' for alias `%s'", command, alias))
	}
}

func (c *Command) With(opts Options) *Command {
	c.options = opts
	return c
}

func (c *Command) Execute(cmd ...string) error {
	for i := 1; i <= len(cmd); i++ {
		command := strings.Join(cmd[0:i], " ")
		if fn, ok := c.commands[command]; ok {
			return fn(c.options, cmd[i:])
		}
	}
	return fmt.Errorf("unrecognized command %s\n", strings.Join(cmd, " "))
}
