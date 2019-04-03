package main

import (
	"fmt"
	"os"

	"github.com/jhunt/go-cli"
	"github.com/jhunt/go-log"

	"github.com/starkandwayne/shield/agent"
)

var Version = ""

func main() {
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
		fmt.Printf("shield-agent - Run a remote SHIELD orchestration agent\n\n")
		fmt.Printf("Options\n")
		fmt.Printf("  -h, --help       Show this help screen.\n")
		fmt.Printf("  -v, --version    Display the SHIELD version.\n")
		fmt.Printf("\n")
		fmt.Printf("  -l, --log-level  What messages to log (error, warning, info, or debug).\n")
		fmt.Printf("  -c, --config     Path to the agent configuration file.\n")
		fmt.Printf("\n")
		os.Exit(0)
	}

	if opts.Version {
		if Version == "" || Version == "dev" {
			fmt.Printf("shield-agent (development)\n")
		} else {
			fmt.Printf("shield-agent v%s\n", Version)
		}
		os.Exit(0)
	}

	if opts.ConfigFile == "" {
		fmt.Fprintf(os.Stderr, "You must specify a configuration file via `--config`\n")
		os.Exit(1)
	}

	log.SetupLogging(log.LogConfig{
		Type:  "console",
		Level: opts.Log,
	})
	log.Infof("starting agent")

	ag := agent.NewAgent()
	ag.Version = Version
	if err := ag.ReadConfig(opts.ConfigFile); err != nil {
		log.Errorf("configuration failed: %s", err)
		return
	}
	ag.Run()
}
