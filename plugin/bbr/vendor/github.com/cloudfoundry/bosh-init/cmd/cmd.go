package cmd

import (
	biui "github.com/cloudfoundry/bosh-init/ui"
)

type Cmd interface {
	Name() string

	Meta() Meta

	Run(biui.Stage, []string) error
}

type Meta struct {
	Synopsis string
	Usage    string
	Env      map[string]MetaEnv
}

type MetaEnv struct {
	Example string
	Default string

	Description string
}

var genericEnv = map[string]MetaEnv{
	"BOSH_INIT_LOG_LEVEL": MetaEnv{
		Example:     "debug",
		Default:     "none",
		Description: "none, info, debug, warn, or error",
	},
	"BOSH_INIT_LOG_PATH": MetaEnv{
		Example:     "/path/to/file.log",
		Default:     "standard out/err",
		Description: "The path where logs will be written",
	},
}
