package cmdrunner_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCompiler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Command Runner Suite")
}
