package orchestrator_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestBackuper(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Orchestrator Suite")
}
