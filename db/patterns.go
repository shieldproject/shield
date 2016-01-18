package db

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	squeezer *regexp.Regexp
)

// Turns globs ('*search*') into SQL patterns ('%search%')
func Pattern(glob string) string {
	s := fmt.Sprintf("%%%s%%", strings.Replace(glob, "*", "%", -1))
	return squeezer.ReplaceAllString(s, "%")
}

// Compile regexn once
func init() {
	squeezer = regexp.MustCompile("%+")
}
