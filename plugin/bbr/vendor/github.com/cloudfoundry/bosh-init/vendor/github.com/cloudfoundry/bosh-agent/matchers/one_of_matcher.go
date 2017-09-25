package matchers

import (
	"fmt"
	"strings"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
)

type OneOfMatcher struct {
	Elements []interface{}
}

func (m *OneOfMatcher) Match(actual interface{}) (success bool, err error) {
	for _, value := range m.Elements {
		submatcher, elementIsMatcher := value.(types.GomegaMatcher)
		if !elementIsMatcher {
			submatcher = gomega.Equal(value)
		}

		success, err = submatcher.Match(actual)
		if success || err != nil {
			return
		}
	}
	return
}

func (m *OneOfMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n%s\n%s\n%s", format.Object(actual, 1), "to match one of", m.expectedValues())
}

func (m *OneOfMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n%s\n%s\n%s", format.Object(actual, 1), "not to match one of", m.expectedValues())
}

func (m *OneOfMatcher) expectedValues() string {
	expectedValues := make([]string, len(m.Elements), len(m.Elements))
	for i, matcher := range m.Elements {
		expectedValues[i] = format.Object(matcher, 1)
	}
	return strings.Join(expectedValues, "\nor\n")
}
