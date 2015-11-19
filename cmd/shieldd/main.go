// Jamie: This contains the go source code that will become shield.

package main

import (
	"fmt"
	"github.com/starkandwayne/shield/supervisor"
	"github.com/voxelbrain/goptions"

	// sql drivers
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type ShielddOpts struct {
	ConfigFile string `goptions:"-c, --config, obligatory, description='Path to the shieldd configuration file'"`
}

func main() {
	fmt.Printf("starting up...\n")

	var opts ShielddOpts
	if err := goptions.Parse(&opts); err != nil {
		fmt.Printf("%s\n", err)
		goptions.PrintHelp()
		return
	}

	s := supervisor.NewSupervisor()
	if err := s.ReadConfig(opts.ConfigFile); err != nil {
		fmt.Printf("configuraiton failed: %s\n", err)
		return
	}

	s.SpawnAPI()
	s.SpawnScheduler()
	s.SpawnWorkers()

	err := s.Run()
	if err != nil {
		if e, ok := err.(supervisor.JobFailedError); ok {
			fmt.Printf("errors encountered while retrieving initial jobs list from database\n")
			for _, fail := range e.FailedJobs {
				fmt.Printf("  - job %s (%s) failed: %s\n", fail.UUID, fail.Tspec, fail.Error)
			}
		} else {
			fmt.Printf("shield daemon failed: %s\n", err)
		}
	}
	fmt.Printf("shutting down...\n")
}
