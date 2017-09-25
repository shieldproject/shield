package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

var (
	Start    int
	End      int
	ExitCode int
	Output   string
	Interval time.Duration
)

func init() {
	flag.IntVar(&Start, "start", 0, "sequence start")
	flag.IntVar(&End, "end", 5, "sequence end")
	flag.IntVar(&ExitCode, "exitcode", 0, "exitcode")
	flag.StringVar(&Output, "out", "stdout", "sets output (stdout, stderr or both)")
	flag.DurationVar(&Interval, "int", time.Millisecond*100, "sequence interval")
}

func run() {
	defer func() {
		if e := recover(); e != nil {
			fmt.Fprintf(os.Stderr, "PANIC: %v\n", e)
		}
	}()

	Output = strings.ToLower(strings.TrimSpace(Output))
	var out io.Writer
	switch Output {
	case "stdout":
		out = os.Stdout
	case "stderr":
		out = os.Stderr
	case "both":
		out = io.MultiWriter(os.Stdout, os.Stderr)
	default:
		out = os.Stdout
	}

	for i := Start; i <= End; i++ {
		fmt.Fprintln(out, i)
		time.Sleep(Interval)
	}
}

func main() {
	flag.Parse()
	run()
	os.Exit(ExitCode)
}
