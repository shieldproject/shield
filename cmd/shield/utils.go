package main

import (
	"fmt"
	"os"
	"strings"
)

func BoolString(tf bool) string {
	if tf {
		return "Y"
	}
	return "F"
}

func DEBUG(format string, args ...interface{}) {
	if debug {
		content := fmt.Sprintf(format, args...)
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			lines[i] = "DEBUG> " + line
		}
		content = strings.Join(lines, "\n")
		fmt.Fprintf(os.Stderr, "%s\n", content)
	}
}
