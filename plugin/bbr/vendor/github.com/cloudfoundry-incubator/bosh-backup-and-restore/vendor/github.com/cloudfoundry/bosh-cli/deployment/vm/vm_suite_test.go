package vm_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestVM(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VM Suite")
}
