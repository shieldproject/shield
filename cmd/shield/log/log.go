package log

import (
	"fmt"
	"os"
	"strings"
)

var shouldDebug, shouldTrace bool

//ToggleDebug sets debug statements on if given true, and turns them off otherwise.
func ToggleDebug(should bool) {
	shouldDebug = should
}

//ToggleTrace sets trace statements on if given true, and turns them off otherwise.
func ToggleTrace(should bool) {
	if should {
		os.Setenv("SHIELD_TRACE", "1")
		DEBUG("enabling TRACE output")
	} else {
		os.Unsetenv("SHIELD_TRACE")
	}
}

//DEBUG sends a debug log to stderr
func DEBUG(format string, args ...interface{}) {
	if shouldDebug {
		content := fmt.Sprintf(format, args...)
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			lines[i] = "DEBUG> " + line
		}
		content = strings.Join(lines, "\n")
		fmt.Fprintf(os.Stderr, "%s\n", content)
	}
}
