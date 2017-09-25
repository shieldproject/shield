package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [STOP|START] [STOPFILE]\n", os.Args[0])
	os.Exit(1)
}

func main() {
	if len(os.Args) != 3 {
		Usage()
	}
	mode := strings.ToLower(os.Args[1])
	filename := os.Args[2]
	switch mode {
	case "start":
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			f, err := os.Create(filename)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			f.Close()
		}
		tick := time.NewTicker(time.Second)
		for _ = range tick.C {
			if _, err := os.Stat(filename); os.IsNotExist(err) {
				fmt.Println("Exiting now...")
				return
			}
			fmt.Println("Hello")
		}
	case "stop":
		os.Remove(filename)
	default:
		Usage()
	}
}
