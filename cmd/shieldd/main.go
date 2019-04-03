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
		Help    bool `cli:"-h, --help"`
		Version bool `cli:"-v, --version"`

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
		fmt.Printf("shieldd - The SHIELD Core daemon\n\n")
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
		if core.Version == "" || core.Version == "dev" {
			fmt.Printf("shieldd (development)\n")
		} else {
			fmt.Printf("shieldd v%s\n", core.Version)
		}
		os.Exit(0)
	}

	log.SetupLogging(log.LogConfig{
		Type:  "console",
		Level: opts.Log,
	})
	log.Infof("starting up shield core")

	c, err := core.Configure(opts.ConfigFile, core.DefaultConfig)
	if err != nil {
		log.Errorf("shield core failed to start up: %s", err)
		os.Exit(1)
	}

	c.Main()
}
