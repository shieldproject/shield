package matchers

import (
	"fmt"
)

func MatchOneOf(elements ...interface{}) *OneOfMatcher {
	if len(elements) < 2 {
		panic(fmt.Sprintf("MatchOneOf requires at least two elements. Got: %s", elements))
	}
	return &OneOfMatcher{
		Elements: elements,
	}
}
