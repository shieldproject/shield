package tui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func Menu(intro string, t *Table, prompt string) interface{} {
	fmt.Printf("%s\n\n", intro)
	t.OutputWithIndices(os.Stdout)

	in := bufio.NewReader(os.Stdin)
	fmt.Printf("\n\n")
	for {
		fmt.Printf("  %s [1-%d] ", prompt, t.Rows())
		v, err := in.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed: %s\n", err)
			return false
		}

		n, err := strconv.ParseInt(strings.TrimRight(v, "\n"), 10, 64)
		if err != nil || n <= 0 || n > int64(t.Rows()) {
			continue
		}
		fmt.Printf("\n\n")
		return t.Object(int(n - 1))
	}
}
