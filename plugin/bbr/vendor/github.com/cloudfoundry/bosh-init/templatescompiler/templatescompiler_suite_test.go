package templatescompiler_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestTemplatescompiler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Templatescompiler Suite")
}
