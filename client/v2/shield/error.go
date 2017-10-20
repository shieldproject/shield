package shield

import (
	"fmt"
	"strings"
)

type Error struct {
	Message string   `json:"error"`
	Missing []string `json:"missing"`
	Extra   string   `json:"diagnostic"`
}

func (e Error) Error() string {
	msg := e.Message
	if msg == "" && len(e.Missing) > 0 {
		msg = fmt.Sprintf("The following fields are missing: %s", strings.Join(e.Missing, ", "))
	}
	if e.Extra == "" {
		return msg
	}
	return fmt.Sprintf("%s (%s)", msg, e.Extra)
}
