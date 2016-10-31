package main

import (
	"fmt"
	"os"

	"github.com/starkandwayne/goutils/log"
	"github.com/voxelbrain/goptions"

	// sql drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/starkandwayne/shield/db"
)

var Version = ""

func main() {
	log.Infof("starting schema...")

	options := struct {
		Help    bool   `goptions:"-h, --help, description='Show the help screen'"`
		Driver  string `goptions:"-t, --type, description='Type of database backend (postgres or mysql)'"`
		DSN     string `goptions:"-d,--database, description='DSN of the database backend'"`
		Version bool   `goptions:"-v, --version, description='Display the SHIELD version'"`
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
	if options.Driver == "" {
		fmt.Fprintf(os.Stderr, "You must indicate which type of database you wish to manage, via the `--type` option.\n")
		os.Exit(1)
	}
	if options.DSN == "" {
		fmt.Fprintf(os.Stderr, "You must specify the DSN of your database, via the `--database` option.\n")
		os.Exit(1)
	}

	database := &db.DB{
		Driver: options.Driver,
		DSN:    options.DSN,
	}

	log.Debugf("connecting to %s database at %s", database.Driver, database.DSN)
	if err := database.Connect(); err != nil {
		log.Errorf("failed to connect to %s database at %s: %s",
			database.Driver, database.DSN, err)
	}

	if err := database.Setup(); err != nil {
		log.Errorf("failed to set up schema in %s database at %s: %s",
			database.Driver, database.DSN, err)
		return
	}

	log.Infof("deployed schema version %d", db.CurrentSchema)
}
