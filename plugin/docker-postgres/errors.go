package main

import (
	"fmt"
	"strings"

	. "github.com/starkandwayne/shield/plugin"
)

type MultiError struct {
	Message string
	errors  []error
}

func (e *MultiError) Append(x error) {
	DEBUG(x.Error())
	e.errors = append(e.errors, x)
}

func (e *MultiError) Appendf(msg string, args ...interface{}) {
	e.errors = append(e.errors, fmt.Errorf(msg, args...))
}

func (e *MultiError) Valid() bool {
	return len(e.errors) > 0
}

func (e MultiError) Error() string {
	s := []string{}
	for _, err := range e.errors {
		s = append(s, fmt.Sprintf("  - %s", err.Error()))
	}
	return fmt.Sprintf("%s:\n%s", e.Message, strings.Join(s, "\n"))
}
