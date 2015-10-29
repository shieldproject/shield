package main

import (
	"fmt"
	"supervisor"
)

func main() {
	fmt.Printf("starting up...\n")
	s := supervisor.NewSupervisor()

	s.SpawnScheduler()
	s.SpawnWorker()
	s.SpawnWorker()
	s.SpawnWorker()

	s.Run()
	fmt.Printf("shutting down...\n")
}
