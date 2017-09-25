package main

import (
	"flag"
	"os"
	"os/exec"
	"time"
)

const HardTimeout = 5 * time.Minute

var (
	LoopInterval time.Duration
	DieAfter     time.Duration
	SubProcess   bool
	ExitCode     int
	Message      string
)

func init() {
	flag.DurationVar(&LoopInterval, "loop", time.Millisecond*250, "Loop interval")
	flag.DurationVar(&LoopInterval, "l", time.Millisecond*250, "Loop interval (shorthand)")

	flag.DurationVar(&DieAfter, "die", -1, "Die after this duration has elapsed")
	flag.DurationVar(&DieAfter, "d", -1, "Die after this duration has elapsed (shorthand)")

	flag.BoolVar(&SubProcess, "subproc", false, "Create a sub process each iteration")
	flag.BoolVar(&SubProcess, "s", false, "Create a sub process each iteration (shorthand)")

	flag.IntVar(&ExitCode, "exit", 2, "Exit code - used in conjunction with die")
	flag.IntVar(&ExitCode, "e", 2, "Exit code - used in conjunction with die (shorthand)")

	flag.StringVar(&Message, "message", "Hello", "Message to print")
	flag.StringVar(&Message, "m", "Hello", "Message to print (shorthand)")
}

func main() {
	flag.Parse()

	// Ensure the message is terminated with a newline
	b := []byte(Message)
	if len(b) == 0 || b[len(b)-1] != '\n' {
		b = append(b, '\n')
	}

	t := time.Now()
	for {
		os.Stdout.Write(b)
		if SubProcess {
			cmd := exec.Command(os.Args[0], "-loop", "5s")
			if err := cmd.Start(); err != nil {
				os.Stderr.WriteString("Error: " + err.Error())
				os.Exit(1)
			}
		}
		time.Sleep(LoopInterval)
		d := time.Since(t)
		if DieAfter >= 0 && d > DieAfter {
			os.Stdout.WriteString("Dying now\n")
			os.Exit(ExitCode)
		}
		if d > HardTimeout {
			os.Stdout.WriteString("Hard timeout!\n")
			os.Exit(1)
		}
	}
}
