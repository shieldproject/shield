package cmd

import (
	biui "github.com/cloudfoundry/bosh-init/ui"
)

const VersionLabel = "[DEV BUILD]"

type versionCmd struct {
	ui biui.UI
}

func NewVersionCmd(ui biui.UI) Cmd {
	return versionCmd{ui: ui}
}

func (c versionCmd) Name() string {
	return "version"
}

func (c versionCmd) Meta() Meta {
	return Meta{
		Synopsis: "Show version",
		Usage:    "version",
	}
}

func (c versionCmd) Run(_ biui.Stage, args []string) error {
	c.ui.PrintLinef("version %s", VersionLabel)
	return nil
}
