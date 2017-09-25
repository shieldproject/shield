package state_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestState(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Instance State Suite")
}
