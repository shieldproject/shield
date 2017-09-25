package fakes

import (
	bicmd "github.com/cloudfoundry/bosh-init/cmd"
	biui "github.com/cloudfoundry/bosh-init/ui"
)

type FakeCommand struct {
	name        string
	meta        bicmd.Meta
	Stage       biui.Stage
	Args        []string
	PresetError error
}

func NewFakeCommand(name string, meta bicmd.Meta) *FakeCommand {
	return &FakeCommand{
		name: name,
		meta: meta,
		Args: []string{},
	}
}

func (f *FakeCommand) Name() string {
	return f.name
}

func (f *FakeCommand) Meta() bicmd.Meta {
	return f.meta
}

func (f *FakeCommand) Run(stage biui.Stage, args []string) error {
	f.Stage = stage
	f.Args = args
	return f.PresetError
}

func (f *FakeCommand) GetArgs() []string {
	return f.Args
}
