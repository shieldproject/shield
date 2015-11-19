// Jamie: This contains the go source code that will become shield.

package main

import (
	"fmt"

	"github.com/starkandwayne/shield/supervisor"

	"github.com/voxelbrain/goptions"
	"github.com/geofffranks/bmad/log"

	// sql drivers
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type ShielddOpts struct {
	ConfigFile string `goptions:"-c, --config, obligatory, description='Path to the shieldd configuration file'"`
}

func main() {
	log.Info("starting shield daemon")

	var opts ShielddOpts
	if err := goptions.Parse(&opts); err != nil {
		fmt.Printf("%s\n", err)
		goptions.PrintHelp()
		return
	}

	s := supervisor.NewSupervisor()
	if err := s.ReadConfig(opts.ConfigFile); err != nil {
		log.Error("configuraiton failed: %s", err)
		return
	}

	s.SpawnAPI()
	s.SpawnScheduler()
	s.SpawnWorkers()

	err := s.Run()
	if err != nil {
		if e, ok := err.(supervisor.JobFailedError); ok {
			log.Error("errors encountered while retrieving initial jobs list from database")
			for _, fail := range e.FailedJobs {

				log.Error("  - job %s (%s) failed: %s", fail.UUID, fail.Tspec, fail.Error)
			}
		} else {
			log.Error("shield daemon failed: %s", err)
		}
	}
	log.Info("stopping daemon")
}
