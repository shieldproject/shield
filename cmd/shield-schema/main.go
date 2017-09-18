package main

import (
	"fmt"
	"os"

	"github.com/starkandwayne/goutils/log"
	"github.com/voxelbrain/goptions"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	"github.com/starkandwayne/shield/db"
)

var Version = ""

func main() {
	log.Infof("starting schema...")

	options := struct {
		Help     bool   `goptions:"-h, --help, description='Show the help screen'"`
		Database string `goptions:"-d,--database, description='Path to the database file'"`
		Version  bool   `goptions:"-v, --version, description='Display the SHIELD version'"`
	}{
	// No defaults
	}
	if err := goptions.Parse(&options); err != nil {
		fmt.Printf("%s\n", err)
		goptions.PrintHelp()
		return
	}
	if options.Help {
		goptions.PrintHelp()
		os.Exit(0)
	}
	if options.Version {
		if Version == "" {
			fmt.Printf("shield-schema (development)%s\n", Version)
		} else {
			fmt.Printf("shield-schema v%s\n", Version)
		}
		os.Exit(0)
	}

	if options.Database == "" {
		fmt.Fprintf(os.Stderr, "You must specify the path to your database, via the `--database` option.\n")
		os.Exit(1)
	}

	database := &db.DB{
		Driver: "sqlite3",
		DSN:    options.Database,
	}

	log.Debugf("connecting to database at %s", database.DSN)
	if err := database.Connect(); err != nil {
		log.Errorf("failed to connect to database at %s: %s",
			database.DSN, err)
		os.Exit(1)
	}

	if err := database.Setup(); err != nil {
		log.Errorf("failed to set up schema in database at %s: %s",
			database.DSN, err)
		os.Exit(1)
	}

	log.Infof("deployed schema version %d", db.CurrentSchema)
}
