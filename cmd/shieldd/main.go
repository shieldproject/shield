package main

import (
	"fmt"
	"os"

	"github.com/starkandwayne/goutils/log"
	"github.com/voxelbrain/goptions"

	// sql drivers
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/starkandwayne/shield/supervisor"
)

type ShielddOpts struct {
	ConfigFile string `goptions:"-c, --config, description='Path to the shieldd configuration file'"`
	Log        string `goptions:"-l, --log-level, description='Set logging level to debug, info, notice, warn, error, crit, alert, or emerg'"`
	Version    bool   `goptions:"-v, --version, description='Display the shieldd version'"`
}

var Version = "(development)"

func main() {
	supervisor.Version = Version
	var opts ShielddOpts
	opts.Log = "Info"
	if err := goptions.Parse(&opts); err != nil {
		fmt.Printf("%s\n", err)
		goptions.PrintHelp()
		return
	}

	if opts.Version {
		fmt.Printf("%s - Version %s\n", os.Args[0], Version)
		os.Exit(0)
	}

	if opts.ConfigFile == "" {
		fmt.Fprintf(os.Stderr, "No config specified. Please try again using the -c/--config argument\n")
		os.Exit(1)
	}

	log.SetupLogging(log.LogConfig{Type: "console", Level: opts.Log})
	log.Infof("starting shield daemon")

	s := supervisor.NewSupervisor()
	if err := s.ReadConfig(opts.ConfigFile); err != nil {
		log.Errorf("Failed to load config: %s", err)
		return
	}

	s.SpawnAPI()
	s.SpawnWorkers()

	if err := s.Run(); err != nil {
		log.Errorf("shield daemon failed: %s", err)
	}
	log.Infof("stopping daemon")
}
