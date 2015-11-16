// Jamie: This contains the go source code that will become shield.

package main

import (
	"fmt"
	"github.com/starkandwayne/shield/db"
	"github.com/voxelbrain/goptions"
	"os"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/lib/pq"
)

func main() {
	fmt.Printf("starting up...\n")

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

	fmt.Fprintf(os.Stderr, "connecting to %s database at %s\n", database.Driver, database.DSN)
	if err := database.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to %s database at %s: %s\n",
			database.Driver, database.DSN, err)
	}

	if err := database.Setup(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set up schema in %s database at %s: %s\n",
			database.Driver, database.DSN, err)
		return
	}

	fmt.Printf("deployed schema version %d\n", db.CurrentSchema)
}
