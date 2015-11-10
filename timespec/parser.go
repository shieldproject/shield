package timespec

import (
	"fmt"
)

func Parse(s string) (*Spec, error) {
	l := LexerForString(s)
	rc := yyParse(l)

	if rc != 0 {
		return nil, fmt.Errorf("parsing failed")
	}

	return l.spec, nil
}
