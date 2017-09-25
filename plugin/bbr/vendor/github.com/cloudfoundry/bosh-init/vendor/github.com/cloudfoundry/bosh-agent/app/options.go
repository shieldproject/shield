package app

import (
	"flag"
	"io/ioutil"
)

type Options struct {
	InfrastructureName string
	PlatformName       string
	BaseDirectory      string
	JobSupervisor      string
	ConfigPath         string
}

func ParseOptions(args []string) (Options, error) {
	var opts Options

	flagSet := flag.NewFlagSet("bosh-agent-args", flag.ContinueOnError)
	flagSet.SetOutput(ioutil.Discard)

	flagSet.StringVar(&opts.PlatformName, "P", "", "Set Platform")

	flagSet.StringVar(&opts.ConfigPath, "C", "", "Config path")
	flagSet.StringVar(&opts.JobSupervisor, "M", "monit", "Set jobsupervisor")
	flagSet.StringVar(&opts.BaseDirectory, "b", "/var/vcap", "Set Base Directory")

	// The following two options are accepted but ignored for compatibility with the old agent
	var systemRoot string
	flagSet.StringVar(&systemRoot, "r", "/", "system root (ignored by go agent)")

	var noAlerts bool
	flagSet.BoolVar(&noAlerts, "no-alerts", false, "don't process alerts (ignored by go agent)")

	// cannot call flagSet.Parse in the return statement due to gccgo
	// execution order issues: https://code.google.com/p/go/issues/detail?id=8698&thanks=8698&ts=1410376474
	err := flagSet.Parse(args[1:])

	return opts, err
}
