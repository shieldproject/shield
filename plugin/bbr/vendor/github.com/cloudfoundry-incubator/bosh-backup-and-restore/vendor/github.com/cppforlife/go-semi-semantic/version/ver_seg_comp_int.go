package version

import (
	"errors"
	"regexp"
	"strconv"
)

var (
	verSegCompIntRegexp          = regexp.MustCompile(`\A[0-9]+\z`)
	verSegCompIntZeroStartRegexp = regexp.MustCompile(`\A0+[0-9]+\z`)
)

type VerSegCompInt struct{ I int }

func NewVerSegCompIntFromString(piece string) (VerSegCompInt, bool, error) {
	if !verSegCompIntRegexp.MatchString(piece) {
		return VerSegCompInt{}, false, nil
	}

	if verSegCompIntZeroStartRegexp.MatchString(piece) {
		return VerSegCompInt{}, false, nil
	}

	i, err := strconv.Atoi(piece)
	if err != nil {
		return VerSegCompInt{}, true, err
	}

	return VerSegCompInt{i}, true, nil
}

func (i VerSegCompInt) Validate() error {
	if i.I < 0 {
		return errors.New("Expected integer component to be greater than or equal to 0")
	}

	return nil
}

func (i VerSegCompInt) Compare(other VerSegComp) int {
	otherTyped := other.(VerSegCompInt)
	switch {
	case i.I < otherTyped.I:
		return -1
	case i.I == otherTyped.I:
		return 0
	case i.I > otherTyped.I:
		return 1
	}
	panic("unreachable")
}

func (i VerSegCompInt) String() string { return i.AsString() }

func (i VerSegCompInt) AsString() string { return strconv.Itoa(i.I) }
