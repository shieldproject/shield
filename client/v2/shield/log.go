package shield

import (
	"fmt"
)

func (c *Client) Debugf(s string, args ...interface{}) {
	if c.Debug {
		fmt.Printf(s+"\n", args...)
	}
}
