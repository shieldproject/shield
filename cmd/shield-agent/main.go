package main

import (
	"fmt"

	"github.com/starkandwayne/shield/agent"
	"github.com/voxelbrain/goptions"

	log "gopkg.in/inconshreveable/log15.v2"
)

type ShieldAgentOpts struct {
	ConfigFile string `goptions:"-c, --config, obligatory, description='Path to the shield-agent configuration file'"`
}

func main() {
	fmt.Printf("starting up...\n")
	log.Info("starting agent...") 
	var opts ShieldAgentOpts
	if err := goptions.Parse(&opts); err != nil {
		fmt.Printf("%s\n", err)
		goptions.PrintHelp()
		return
	}

	ag := agent.NewAgent()
	if err := ag.ReadConfig(opts.ConfigFile); err != nil {
		fmt.Printf("configuration failed: %s\n", err)
		log.Error("configuration failed: ", err) 
		return
	}
	ag.Run()
}
