// Jamie: This contains the go source code that will become shield.

package main

import (
	"fmt"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/db"
	"github.com/starkandwayne/shield/supervisor"
	"os"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	fmt.Printf("starting up...\n")

	database := &db.DB{
		Driver: "sqlite3",
		DSN:    "/tmp/db.sqlite3", // FIXME: need configuration
	}

	// spin up the HTTP API
	c := make(chan int)
	go api.Run(":8080", database, c)

	// connect in main goroutine
	if err := database.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to %s database at %s: %s\n",
			database.Driver, database.DSN, err)
		return
	}

	if err := database.CheckCurrentSchema(); err != nil {
		fmt.Fprintf(os.Stderr, "database failed schema version check: %s\n", err)
		return
	}

	s := supervisor.NewSupervisor(database, c)

	s.SpawnScheduler()
	s.SpawnWorker()
	s.SpawnWorker()
	s.SpawnWorker()

	s.Run()
	fmt.Printf("shutting down...\n")
}
