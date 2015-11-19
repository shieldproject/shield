// Jamie: This contains the go source code that will become shield.

package main

import (
	"github.com/starkandwayne/shield/db"

	"github.com/voxelbrain/goptions"
	"github.com/geofffranks/bmad/log"

	// sql drivers
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	log.Info("starting schema...")

	options := struct {
		Driver string `goptions:"-t,--type, obligatory, description='Type of database backend'"`
		DSN    string `goptions:"-d,--database, obligatory, description='DSN of the database backend'"`
	}{
	// No defaults
	}
	goptions.ParseAndFail(&options)

	database := &db.DB{
		Driver: options.Driver,
		DSN:    options.DSN,
	}

	log.Debug("connecting to %s database at %s", database.Driver, database.DSN)
	if err := database.Connect(); err != nil {
		log.Error("failed to connect to %s database at %s: %s",
			database.Driver, database.DSN, err)
	}

	if err := database.Setup(); err != nil {
		log.Error("failed to set up schema in %s database at %s: %s",
			database.Driver, database.DSN, err)
		return
	}

	log.Info("deployed schema version %d", db.CurrentSchema)
}
