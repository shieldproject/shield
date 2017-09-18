package backends

import (
	"os"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/cmd/shield/config"
)

//DisplayCurrent displays information about the currently targeted backend to
//the screen
func DisplayCurrent() {
	cur := config.Current()
	if cur == nil {
		ansi.Fprintf(os.Stderr, "No current SHIELD backend\n\n")
	} else {
		ansi.Fprintf(os.Stderr, "Using @G{%s} (%s) as SHIELD backend\n\n", cur.Address, cur.Name)
	}
}
