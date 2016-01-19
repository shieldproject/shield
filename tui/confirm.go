package tui

import (
	"bufio"
	"os"
	"strings"

	"github.com/jhunt/ansi"
)

func Confirm(prompt string) bool {
	in := bufio.NewReader(os.Stdin)

	for {
		ansi.Printf("@Y{%s [y/n]} ", prompt)
		v, err := in.ReadString('\n')
		if err != nil {
			ansi.Fprintf(os.Stderr, "failed: @R{%s}\n", err)
			return false
		}

		switch strings.TrimRight(v, "\n") {
		case "Y":
			fallthrough
		case "y":
			fallthrough
		case "yes":
			return true

		case "N":
			fallthrough
		case "n":
			fallthrough
		case "no":
			return false
		}
	}
}
