package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	log "github.com/shieldproject/shield/internal/log"

	// sql drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"

	"github.com/shieldproject/shield/db"
)

var Version = ""

func main() {
	var driver string
	var database string
	var revision int
	var debug bool

	rootCmd := &cobra.Command{
		Use:   "shield-schema",
		Short: "Deploy a SHIELD database schema",
		Long:  "shield-schema - Deploy a SHIELD database schema",
		Version: func() string {
			if Version == "" || Version == "dev" {
				return "development"
			}
			return Version
		}(),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if database == "" {
				fmt.Fprintf(os.Stderr, "You must specify the path to your database, via the `--database` option.\n")
				os.Exit(1)
			}

			level := "info"
			if debug {
				level = "debug"
			}
			log.SetupLogging(level)

			log.Infof("starting schema...")
			log.Debugf("connecting to database at %s", database)
			d, err := db.Connect(driver, database)
			if err != nil {
				log.Errorf("failed to connect to database at %s: %s", database, err)
				os.Exit(1)
			}

			if revision > db.CurrentSchema {
				log.Errorf("unable to deploy schema revision %d: latest available is %d",
					revision, db.CurrentSchema)
				os.Exit(1)
			}
			if revision < 0 {
				log.Errorf("invalid schema revision '%d'", revision)
				os.Exit(1)
			}
			deployed, err := d.Setup(revision)
			if err != nil {
				log.Errorf("failed to set up schema in database at %s: %s", d.DSN, err)
				os.Exit(1)
			}

			log.Infof("deployed schema version %d", deployed)
			return nil
		},
	}

	rootCmd.Flags().StringVarP(&driver, "driver", "", "sqlite3", "Database driver (sqlite3, pgx, mysql)")
	rootCmd.Flags().StringVarP(&database, "database", "d", "", "Database DSN or file path")
	rootCmd.Flags().IntVarP(&revision, "revision", "r", 0, "Schema version to deploy (default: latest)")
	rootCmd.Flags().BoolVarP(&debug, "debug", "D", false, "Enable debugging output")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
