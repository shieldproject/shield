package agentlogger_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAgentLogger(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Agent Logger Suite")
}
