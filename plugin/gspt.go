// +build cgo

package plugin

import (
	"os"

	"github.com/ErikDubbelboer/gspt"
)

func init() {
	gspt.SetProcTitle(os.Args[0])
}
