package main

import (
	"fmt"
	"os"

	"github.com/voxelbrain/goptions"

	"github.com/starkandwayne/goutils/log"
	"github.com/starkandwayne/shield/agent"
)

type ShieldAgentOpts struct {
	Help       bool   `goptions:"-h, --help, description='Show the help screen'"`
	ConfigFile string `goptions:"-c, --config, description='Path to the shield-agent configuration file'"`
	Log        string `goptions:"-l, --log-level, description='Set logging level to debug, info, notice, warn, error, crit, alert, or emerg'"`
	Version    bool   `goptions:"-v, --version, description='Display the SHIELD version'"`
}

var Version = ""

func main() {
	var opts ShieldAgentOpts
	opts.Log = "Info"
	if err := goptions.Parse(&opts); err != nil {
		fmt.Printf("%s\n", err)
		goptions.PrintHelp()
		return
	}
	if opts.Help {
		goptions.PrintHelp()
		os.Exit(0)
	}
	if opts.Version {
		if Version == "" {
			fmt.Printf("shield-agent (development)%s\n", Version)
		} else {
			fmt.Printf("shield-agent v%s\n", Version)
		}
		os.Exit(0)
	}
	if opts.ConfigFile == "" {
		fmt.Fprintf(os.Stderr, "You must specify a configuration file via `--config`\n")
		os.Exit(1)
	}

	log.SetupLogging(log.LogConfig{Type: "console", Level: opts.Log})
	log.Infof("starting agent")

	ag := agent.NewAgent()
	if err := ag.ReadConfig(opts.ConfigFile); err != nil {
		log.Errorf("configuration failed: %s", err)
		return
	}
	ag.Run()
}
