package main

import (
	"fmt"

	"github.com/voxelbrain/goptions"

	"github.com/starkandwayne/goutils/log"
	"github.com/starkandwayne/shield/agent"
)

type ShieldAgentOpts struct {
	ConfigFile string `goptions:"-c, --config, obligatory, description='Path to the shield-agent configuration file'"`
	Log        string `goptions:"-l, --log-level, description='Set logging level to debug, info, notice, warn, error, crit, alert, or emerg'"`
}

func main() {
	var opts ShieldAgentOpts
	opts.Log = "Info"
	if err := goptions.Parse(&opts); err != nil {
		fmt.Printf("%s\n", err)
		goptions.PrintHelp()
		return
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
