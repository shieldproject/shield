package cmd

import (
	biui "github.com/cloudfoundry/bosh-init/ui"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type Runner struct {
	factory Factory
}

func NewRunner(factory Factory) *Runner {
	return &Runner{
		factory: factory,
	}
}

func (r *Runner) Run(stage biui.Stage, args ...string) error {
	args = r.processArgs(args)
	commandName := args[0]

	cmd, err := r.factory.CreateCommand(commandName)
	if err != nil {
		return err
	}

	err = cmd.Run(stage, args[1:])
	if err != nil {
		return bosherr.WrapErrorf(err, "Command '%s' failed", commandName)
	}

	return nil
}

func (r *Runner) processArgs(args []string) []string {
	if len(args) == 0 {
		return []string{"help"}
	}

	for i, arg := range args {
		if arg == "help" || arg == "-h" || arg == "--help" {
			return append(append([]string{"help"}, args[:i]...), args[i+1:]...)
		}
		if arg == "version" || arg == "-v" || arg == "--version" {
			return []string{"version"}
		}
	}

	return args
}
