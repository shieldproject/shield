// Jamie: This contains the go source code that will become shield.

package main

import (
	"api"
	"db"
	"supervisor"

	"fmt"
	"os"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	fmt.Printf("starting up...\n")

	db := &db.DB{
		Driver: "sqlite3",
		DSN:    "/tmp/db.sqlite3", // FIXME: need configuration
	}

	// connect in main goroutine
	if err := db.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to %s database at %s: %s\n",
			db.Driver, db.DSN, err)
		return
	}

	// FIXME: move this into a separate schema tool ...
	if err := db.Setup(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set up schema in %s database at %s: %s\n",
			db.Driver, db.DSN, err)
		return
	}

	c := make(chan int)
	go api.Run(":8080", db, c)

	s := supervisor.NewSupervisor(db, c)

	s.SpawnScheduler()
	s.SpawnWorker()
	s.SpawnWorker()
	s.SpawnWorker()

	s.Run()
	fmt.Printf("shutting down...\n")
}
