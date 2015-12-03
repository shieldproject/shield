package api_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestSupervisor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Client Library Test Suite")
}
