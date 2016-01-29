package main

import (
	"github.com/starkandwayne/goutils/log"
	"github.com/voxelbrain/goptions"

	// sql drivers
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/starkandwayne/shield/db"
)

func main() {
	log.Infof("starting schema...")

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
