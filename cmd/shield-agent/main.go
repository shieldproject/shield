package main

import (
	"fmt"

	"github.com/starkandwayne/goutils/log"
	"github.com/starkandwayne/shield/agent"
	
	"github.com/voxelbrain/goptions"
)

type ShieldAgentOpts struct {
	ConfigFile string `goptions:"-c, --config, obligatory, description='Path to the shield-agent configuration file'"`
}

func main() {
	log.Infof("starting agent...")
	var opts ShieldAgentOpts
	if err := goptions.Parse(&opts); err != nil {
		fmt.Printf("%s\n", err)
		goptions.PrintHelp()
		return
	}

	ag := agent.NewAgent()
	if err := ag.ReadConfig(opts.ConfigFile); err != nil {
		log.Errorf("configuration failed: %s", err)
		return
	}
	ag.Run()
}
