package main

import (
	"fmt"
	"os"

	"github.com/jhunt/go-cli"
	"github.com/jhunt/go-log"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	"github.com/starkandwayne/shield/db"
)

var Version = ""

func main() {
	log.Infof("starting schema...")

	var opts struct {
		Help     bool   `cli:"-h, --help"`
		Version  bool   `cli:"-v, --version"`
		Debug    bool   `cli:"-D, --debug"`
		Database string `cli:"-d, --database"`
		Revision int    `cli:"-r, --revision"`
	}

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
		fmt.Printf("shield-schema - Deploy a SHIELD database schema\n\n")
		fmt.Printf("Options\n")
		fmt.Printf("  -h, --help       Show this help screen.\n")
		fmt.Printf("  -D, --debug      Enable debugging output.\n")
		fmt.Printf("  -v, --version    Display the SHIELD version.\n")
		fmt.Printf("\n")
		fmt.Printf("  -d, --database   Path to the SQLite3 database file.\n")
		fmt.Printf("  -r, --revision   What version of the SHIELD schema\n")
		fmt.Printf("                   to deploy.  Defaults to latest.\n")
		fmt.Printf("\n")
		os.Exit(0)
	}

	if opts.Version {
		if Version == "" || Version == "dev" {
			fmt.Printf("shield-schema (development)\n")
		} else {
			fmt.Printf("shield-schema v%s\n", Version)
		}
		os.Exit(0)
	}

	if opts.Database == "" {
		fmt.Fprintf(os.Stderr, "You must specify the path to your database, via the `--database` option.\n")
		os.Exit(1)
	}

	level := "info"
	if opts.Debug {
		level = "debug"
	}
	log.SetupLogging(log.LogConfig{
		Type:  "console",
		Level: level,
	})

	log.Debugf("connecting to database at %s", opts.Database)
	database, err := db.Connect(opts.Database)
	if err != nil {
		log.Errorf("failed to connect to database at %s: %s",
			opts.Database, err)
		os.Exit(1)
	}

	if opts.Revision > db.CurrentSchema {
		log.Errorf("unable to deploy schema revision %d: latest available is %d",
			opts.Revision, db.CurrentSchema)
		os.Exit(1)
	}
	if opts.Revision < 0 {
		log.Errorf("invalid schema revision '%d'", opts.Revision)
		os.Exit(1)
	}
	deployed, err := database.Setup(opts.Revision)
	if err != nil {
		log.Errorf("failed to set up schema in database at %s: %s",
			database.DSN, err)
		os.Exit(1)
	}

	log.Infof("deployed schema version %d", deployed)
}
