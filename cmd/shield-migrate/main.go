package main

import (
	"fmt"
	"os"

	"github.com/jhunt/go-cli"
	"github.com/jhunt/go-log"

	// sql drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/starkandwayne/shield/db"
)

var Version = ""

func main() {
	log.Infof("starting database migration...")

	var opts struct {
		Help     bool   `cli:"-h, --help"`
		Version  bool   `cli:"-v, --version"`
		Debug    bool   `cli:"-D, --debug"`
		FromType string `cli:"--type"`
		FromDSN  string `cli:"--from"`
		Database string `cli:"--to"`
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
		fmt.Printf("shield-migrate - Migrate SHIELD Data\n\n")
		fmt.Printf("Options\n")
		fmt.Printf("  -h, --help       Show this help screen.\n")
		fmt.Printf("  -D, --debug      Enable debugging output.\n")
		fmt.Printf("  -v, --version    Display the SHIELD version.\n")
		fmt.Printf("\n")
		fmt.Printf("  -T, --type       Type of database to migrate from.\n")
		fmt.Printf("                   (either: sqlite3, mysql, or postgres)\n")
		fmt.Printf("  -F, --from       DSN of the database to migrate.\n")
		fmt.Printf("  -d, --database   Path to the SQLite3 DB to migrate to.\n")
		fmt.Printf("\n")
		os.Exit(0)
	}

	if opts.Version {
		if Version == "" {
			fmt.Printf("shield-migrate (development)%s\n", Version)
		} else {
			fmt.Printf("shield-migrate v%s\n", Version)
		}
		os.Exit(0)
	}

	if opts.Database == "" {
		fmt.Fprintf(os.Stderr, "You must specify the path to your database, via the `--database` option.\n")
		os.Exit(1)
	}
	if opts.FromType == "" {
		fmt.Fprintf(os.Stderr, "You must specify the type of database to migrate from, via the `--type` option.\n")
		os.Exit(1)
	}
	if opts.FromDSN == "" {
		fmt.Fprintf(os.Stderr, "You must specify the DSN of the database to migrate from, via the `--from` option.\n")
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

	if _, err := os.Stat(opts.Database); err == nil || os.IsExist(err) {
		log.Errorf("final sqlite3 database %s already exist; aborting...", opts.Database)
		os.Exit(0)
	}

	from := &db.DB{
		Driver: opts.FromType,
		DSN:    opts.FromDSN,
	}
	log.Debugf("reading from %s database '%s'", from.Driver, from.DSN)
	if err := from.Connect(); err != nil {
		log.Errorf("failed to connect to the 'from' database %s:%s: %s", from.Driver, from.DSN, err)
		os.Exit(2)
	}

	to := &db.DB{
		Driver: "sqlite3",
		DSN:    opts.Database,
	}
	log.Debugf("writing to %s database at %s", to.DSN)
	if err := to.Connect(); err != nil {
		log.Errorf("failed to connect to database at %s: %s",
			to.DSN, err)
		os.Exit(1)
	}

	if _, err := to.Setup(3); err != nil {
		log.Errorf("failed to set up schema v3 in database at %s: %s", to.DSN, err)
		os.Exit(1)
	}

	migrateTargets(to, from)
	migrateStores(to, from)
	migrateSchedules(to, from)
	migrateRetention(to, from)
	migrateJobs(to, from)
	migrateArchives(to, from)
	migrateTasks(to, from)
}
