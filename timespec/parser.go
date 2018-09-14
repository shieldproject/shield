package timespec

import (
	"fmt"
)

func Parse(s string) (*Spec, error) {
	l := LexerForString(s)
	rc := yyParse(l)

	if rc != 0 || l.spec == nil {
		return nil, fmt.Errorf("There was a syntax error in your SHIELD timespec '%s'", s)
	}

	return l.spec, l.spec.Error
}
