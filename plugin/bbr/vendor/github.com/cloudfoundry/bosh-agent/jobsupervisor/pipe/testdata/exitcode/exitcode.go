package main

import (
	"flag"
	"os"
	"time"
)

var ExitCode int

func init() {
	flag.IntVar(&ExitCode, "exitcode", 0, "set the exit code")
}

func main() {
	flag.Parse()
	time.Sleep(time.Millisecond * 100)
	os.Exit(ExitCode)
}
