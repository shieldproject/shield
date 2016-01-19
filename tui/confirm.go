package tui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func Confirm(prompt string) bool {
	in := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s", Yellow(fmt.Sprintf("%s [y/n] ", prompt)))
		v, err := in.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed: %s\n", err)
			return false
		}

		switch strings.TrimRight(v, "\n") {
		case "Y": fallthrough
		case "y": fallthrough
		case "yes": return true

		case "N": fallthrough
		case "n": fallthrough
		case "no": return false
		}
	}
}
