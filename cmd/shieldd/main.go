// Jamie: This contains the go source code that will become shield.

package main

import (
	"fmt"
	"github.com/starkandwayne/shield/supervisor"
	"github.com/voxelbrain/goptions"

	log "gopkg.in/inconshreveable/log15.v2"

	// sql drivers
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type ShielddOpts struct {
	ConfigFile string `goptions:"-c, --config, obligatory, description='Path to the shieldd configuration file'"`
}

func main() {
	fmt.Printf("starting up...\n")
	log.Info("starting shield daemon")

	var opts ShielddOpts
	if err := goptions.Parse(&opts); err != nil {
		fmt.Printf("%s\n", err)
		goptions.PrintHelp()
		return
	}

	s := supervisor.NewSupervisor()
	if err := s.ReadConfig(opts.ConfigFile); err != nil {
		fmt.Printf("configuraiton failed: %s\n", err)
		log.Error("invalid configuration", "error", err)
		return
	}

	s.SpawnAPI()
	s.SpawnScheduler()
	s.SpawnWorker()
	s.SpawnWorker()
	s.SpawnWorker()

	err := s.Run()
	if err != nil {
		if e, ok := err.(supervisor.JobFailedError); ok {
			fmt.Printf("errors encountered while retrieving initial jobs list from database\n")
			log.Error("errors encountered while retrieving initial jobs list from database")
			for _, fail := range e.FailedJobs {

				fmt.Printf("  - job %s (%s) failed: %s\n", fail.UUID, fail.Tspec, fail.Error)
				log.Error(" - failed job: ", "UUID", fail.UUID, "time spec", fail.Tspec, "error", fail.Error)
			}
		} else {
			fmt.Printf("shield daemon failed: %s\n", err)
			log.Error("shield daemon failed", "error", err)
		}
	}
	log.Info("stopping daemon")
	fmt.Printf("shutting down...\n")
}
