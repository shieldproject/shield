package version

import (
	"errors"
	"regexp"
)

var (
	verSegCompStrRegexp = regexp.MustCompile(`\A[0-9A-Za-z_\-]+\z`)
)

type VerSegCompStr struct{ S string }

func NewVerSegCompStrFromString(piece string) (VerSegCompStr, bool) {
	if !verSegCompStrRegexp.MatchString(piece) {
		return VerSegCompStr{}, false
	}

	return VerSegCompStr{piece}, true
}

func (s VerSegCompStr) Validate() error {
	if len(s.S) == 0 {
		return errors.New("Expected string component to be non-empty")
	}

	return nil
}

func (s VerSegCompStr) Compare(other VerSegComp) int {
	otherTyped := other.(VerSegCompStr)
	switch {
	case s.S < otherTyped.S:
		return -1
	case s.S == otherTyped.S:
		return 0
	case s.S > otherTyped.S:
		return 1
	}
	panic("unreachable")
}

func (s VerSegCompStr) String() string { return s.AsString() }

func (s VerSegCompStr) AsString() string { return s.S }
