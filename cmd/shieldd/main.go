package main

import (
	"fmt"
	"os"

	"github.com/jhunt/go-cli"
	"github.com/jhunt/go-log"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	"github.com/starkandwayne/shield/core"
)

var Version = ""

func main() {
	core.Version = Version

	var opts struct {
		Help       bool   `cli:"-h, --help"`
		Version    bool   `cli:"-v, --version"`
		ConfigFile string `cli:"-c, --config"`
		Log        string `cli:"-l, --log-level"`
	}
	opts.Log = "info"

	_, args, err := cli.Parse(&opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "!!! %s\n", err)
		os.Exit(1)
	}
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "!!! extra arguments found\n")
		os.Exit(1)
	}

	if opts.Help {
		fmt.Printf("shieldd - Run a SHIELD Core daemon\n\n")
		fmt.Printf("Options\n")
		fmt.Printf("  -h, --help       Show this help screen.\n")
		fmt.Printf("  -v, --version    Display the SHIELD version.\n")
		fmt.Printf("\n")
		fmt.Printf("  -l, --log-level  What messages to log (error, warning, info, or debug).\n")
		fmt.Printf("  -c, --config     Path to the SHIELD Core configuration file.\n")
		fmt.Printf("\n")
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

	log.SetupLogging(log.LogConfig{
		Type:  "console",
		Level: opts.Log,
	})
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
