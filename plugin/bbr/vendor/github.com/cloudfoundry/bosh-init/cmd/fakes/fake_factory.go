package fakes

import (
	cmd "github.com/cloudfoundry/bosh-init/cmd"
)

type FakeFactory struct {
	CommandName   string
	PresetError   error
	PresetCommand *FakeCommand
}

func (f *FakeFactory) CreateCommand(name string) (cmd.Cmd, error) {
	f.CommandName = name
	return f.PresetCommand, f.PresetError
}
