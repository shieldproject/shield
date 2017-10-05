package main

import (
	"fmt"
	"os"

	"github.com/starkandwayne/goutils/log"
	"github.com/voxelbrain/goptions"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	"github.com/starkandwayne/shield/core"
)

type ShielddOpts struct {
	Help       bool   `goptions:"-h, --help, description='Show the help screen'"`
	ConfigFile string `goptions:"-c, --config, description='Path to the shieldd configuration file'"`
	Log        string `goptions:"-l, --log-level, description='Set logging level to debug, info, notice, warn, error, crit, alert, or emerg'"`
	Version    bool   `goptions:"-v, --version, description='Display the SHIELD version'"`
}

var Version = ""

func main() {
	core.Version = Version
	var opts ShielddOpts
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
			fmt.Printf("shieldd (development)\n")
		} else {
			fmt.Printf("shieldd v%s\n", Version)
		}
		os.Exit(0)
	}

	if opts.ConfigFile == "" {
		fmt.Fprintf(os.Stderr, "No config specified. Please try again using the -c/--config argument\n")
		os.Exit(1)
	}

	log.SetupLogging(log.LogConfig{Type: "console", Level: opts.Log})
	log.Infof("starting up shield core")

	daemon, err := core.NewCore(opts.ConfigFile)
	if err != nil {
		log.Errorf("shield core failed to start up: %s", err)
		os.Exit(1)
	}

	if err := daemon.Run(); err != nil {
		log.Errorf("shield core failed to run: %s", err)
		os.Exit(1)
	}

	log.Infof("shutting down shield core")
}
