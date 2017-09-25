package version

import (
	"errors"
	"fmt"
	"strings"
)

type VerSegComp interface {
	Validate() error
	// Compare should panic if incompatible interface is given
	Compare(VerSegComp) int
	AsString() string
}

type VersionSegment struct {
	Components []VerSegComp
}

func MustNewVersionSegmentFromString(v string) VersionSegment {
	verSeg, err := NewVersionSegmentFromString(v)
	if err != nil {
		panic(fmt.Sprintf("Invalid version segment '%s': %s", v, err))
	}

	return verSeg
}

func NewVersionSegmentFromString(v string) (VersionSegment, error) {
	pieces := strings.Split(v, ".")

	components := []VerSegComp{}

	for _, p := range pieces {
		i, matchedI, err := NewVerSegCompIntFromString(p)
		if err != nil {
			errMsg := fmt.Sprintf("Expected component '%s' from version segment '%s' to be a parseable integer: %s", p, v, err)
			return VersionSegment{}, errors.New(errMsg)
		}

		if matchedI {
			components = append(components, i)
		} else if s, matched := NewVerSegCompStrFromString(p); matched {
			components = append(components, s)
		} else {
			errMsg := fmt.Sprintf("Expected component '%s' from version segment '%s' to be either an integer or a formatted string", p, v)
			return VersionSegment{}, errors.New(errMsg)
		}
	}

	return VersionSegment{components}, nil
}

func NewVersionSegment(components []VerSegComp) (VersionSegment, error) {
	if len(components) == 0 {
		return VersionSegment{}, errors.New("Expected version segment to be build from at least one component")
	}

	for _, c := range components {
		err := c.Validate()
		if err != nil {
			return VersionSegment{}, err
		}
	}

	return VersionSegment{components}, nil
}

func (s VersionSegment) Increment() (VersionSegment, error) {
	if len(s.Components) == 0 {
		errMsg := "Expected version segment to have at least one component to be incremented"
		return VersionSegment{}, errors.New(errMsg)
	}

	lastComp := s.Components[len(s.Components)-1]

	lastCompInt, isInt := lastComp.(VerSegCompInt)
	if !isInt {
		errMsg := fmt.Sprintf("Expected version segment '%s' to have last component '%s' to be an integer", s, lastComp)
		return VersionSegment{}, errors.New(errMsg)
	}

	copiedComponents := make([]VerSegComp, len(s.Components))
	copy(copiedComponents, s.Components)
	copiedComponents[len(copiedComponents)-1] = VerSegCompInt{I: lastCompInt.I + 1}

	return NewVersionSegment(copiedComponents)
}

func (s VersionSegment) Copy() VersionSegment {
	// Don't use constructor; assuming that original components are valid
	copiedComponents := make([]VerSegComp, len(s.Components))
	copy(copiedComponents, s.Components)
	return VersionSegment{copiedComponents}
}

func (s VersionSegment) Empty() bool { return len(s.Components) == 0 }

func (s VersionSegment) String() string { return s.AsString() }

func (s VersionSegment) AsString() string {
	result := ""

	for i, c := range s.Components {
		result += c.AsString()

		if i < len(s.Components)-1 {
			result += "."
		}
	}

	return result
}

func (s VersionSegment) Compare(other VersionSegment) int {
	a := s.Components
	b := other.Components

	if len(a) > len(b) {
		comparison := s.compareArrays(a[0:len(b)], b)
		if comparison != 0 {
			return comparison
		}
		if !s.isAllZeros(a[len(b):len(a)]) {
			return 1
		}
		return 0
	}

	if len(a) < len(b) {
		comparison := s.compareArrays(a, b[0:len(a)])
		if comparison != 0 {
			return comparison
		}
		if !s.isAllZeros(b[len(a):len(b)]) {
			return -1
		}
		return 0
	}

	return s.compareArrays(a, b)
}

func (s VersionSegment) IsEq(other VersionSegment) bool { return s.Compare(other) == 0 }
func (s VersionSegment) IsGt(other VersionSegment) bool { return s.Compare(other) == 1 }
func (s VersionSegment) IsLt(other VersionSegment) bool { return s.Compare(other) == -1 }

// compareArrays compares 2 equally sized a & b
func (s VersionSegment) compareArrays(a, b []VerSegComp) int {
	for i, v1 := range a {
		v2 := b[i]

		_, v1IsStr := v1.(VerSegCompStr)
		_, v1IsInt := v1.(VerSegCompInt)
		_, v2IsStr := v2.(VerSegCompStr)
		_, v2IsInt := v2.(VerSegCompInt)

		if v1IsStr && v2IsInt {
			return 1
		} else if v1IsInt && v2IsStr {
			return -1
		}

		comparison := v1.Compare(v2)
		if comparison != 0 {
			return comparison
		}
	}

	return 0
}

func (s VersionSegment) isAllZeros(a []VerSegComp) bool {
	for _, v := range a {
		vTyped, ok := v.(VerSegCompInt)
		if !ok || vTyped.I != 0 {
			return false
		}
	}

	return true
}
