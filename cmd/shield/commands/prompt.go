package commands

import (
	"fmt"
	"os"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
)

//OK prints an okay message to the screen
func OK(f string, l ...interface{}) {
	if *Opts.Raw {
		internal.RawJSON(map[string]string{"ok": fmt.Sprintf(f, l...)})
		return
	}
	ansi.Printf("@G{%s}\n", fmt.Sprintf(f, l...))
}

//MSG prints an informational message to the screen
func MSG(f string, l ...interface{}) {
	if !*Opts.Raw {
		ansi.Printf("\n@G{%s}\n", fmt.Sprintf(f, l...))
	}
}

//CurrentUser returns a string about the current user info
func CurrentUser() string {
	return fmt.Sprintf("%s@%s", os.Getenv("USER"), os.Getenv("HOSTNAME"))
}
