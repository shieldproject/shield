package internal

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

//ReadAll reads everything from the given Reader and
// returns it as a string.
func ReadAll(in io.Reader) (string, error) {
	b, err := ioutil.ReadAll(in)
	return string(b), err
}

func Require(good bool, msg string) {
	if !good {
		fmt.Fprintf(os.Stderr, "USAGE: %s ...\n", msg)
		os.Exit(1)
	}
}
